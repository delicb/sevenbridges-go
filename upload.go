package sevenbridges

import (
	"context"
	"os"
	"time"

	"bytes"
	"fmt"
	"path/filepath"
	"sync"

	"go.delic.rs/cliware-middlewares/body"
	"go.delic.rs/cliware-middlewares/headers"
	"go.delic.rs/cliware-middlewares/query"
	"go.delic.rs/cliware-middlewares/responsebody"
	"go.delic.rs/cliware-middlewares/url"
	"go.delic.rs/gwc"
)

// UploadInitResponse holds information about upload that is in progress with
// all necessary information.
type UploadInitResponse struct {
	Name            string `json:"name"`
	Size            int64  `json:"size"`
	PartSize        int64  `json:"part_size"`
	ParallelUploads bool   `json:"parallel_uploads"`
	Href            string `json:"href"`
	UploadID        string `json:"upload_id"`
	Project         string `json:"project"`
}

// PartUploadInitResponse holds API response for upload initialization of a
// single part request.
type PartUploadInitResponse struct {
	Method       string                 `json:"method"`
	URL          string                 `json:"url"`
	Expires      time.Time              `json:"expires"`
	Headers      map[string]interface{} `json:"headers"`
	SuccessCodes []int                  `json:"success_codes"`
	Report       map[string]interface{} `json:"report"`
}

// UploadInfo holds information about file that is being uploads, its destination
// and various options.
type UploadInfo struct {
	// Path is path to file that should be uploaded on local filesystem.
	Path string
	// Name is desired name of uploaded file. If not provided, same name
	// as on local file system will be used.
	Name string `json:"name"`
	// Overwrite is flag marking if upload should be continued even if file
	// with same name exists on platform.
	Overwrite bool
	// Project is ID of project to which to upload file.
	Project string `json:"project"`

	// there are private and will be populated by library
	stat os.FileInfo
}

type part struct {
	ID        int
	StartByte int64
	EndByte   int64
	ETag      string
	Error     error
}

// MultipartUpload holds information about upload of a file to platform.
type MultipartUpload struct {
	Href      string `json:"href"`
	UploadID  string `json:"upload_id"`
	Project   string `json:"project"`
	Name      string `json:"name"`
	Initiated string `json:"initiated"`
}

type multipartUploadPage struct {
	*Page
	Items []*MultipartUpload `json:"items"`
}

// UploadService is a service for uploading files to SevenBridges platform.
type UploadService interface {
	// Upload uploads file to SevenBridges platform. Which files is uploaded
	// and to which project can be defined in provided options.
	Upload(ctx context.Context, info UploadInfo) error
	// List returns all ongoing uploads.
	List(ctx context.Context) ([]*MultipartUpload, *Response, error)
	// About stops upload with provided ID. Note that this has nothing to do
	// with current running process, this only aborts upload on server.
	// User should be careful which upload it is safe to abort.
	Abort(ctx context.Context, uploadID string) (*Response, error)
}

type uploadService struct {
	*service
}

func newUploadService(client gwc.Doer) UploadService {
	return &uploadService{newService(client)}
}

var _ UploadService = new(uploadService)

func (u *uploadService) List(ctx context.Context) ([]*MultipartUpload, *Response, error) {
	var multipartUpload []*MultipartUpload
	resp, err := u.Do(
		ctx,
		headers.Method("GET"),
		url.AddPath("/upload/multiplart"),
		pageResponse(&multipartUpload),
	)
	return multipartUpload, resp, err
}

func (u *uploadService) Abort(ctx context.Context, uploadID string) (*Response, error) {
	return u.Do(
		ctx,
		headers.Method("DELETE"),
		url.AddPath("upload/multipart/"+uploadID),
	)
}

func (u *uploadService) Upload(ctx context.Context, uploadInfo UploadInfo) error {
	/*
		List of thinks to do for file upload:
		- determine file size and local file name
		- determine remote file name
			- might be the same, might be different, provide override param
		- initialize upload
			- call to upload/multipart with file name, project ID, desired
			  part size and total size
			- fetch response that has part size that has to be used, ID of upload,
			  URL and some other stuff
		- calculate offsets based on part size and total file size
		- for each part
			- get URL where to upload PART
			- upload part to fetched URL
			- report that part upload is finished
		- mark upload as completed

	*/

	stat, err := os.Stat(uploadInfo.Path)
	if err != nil {
		return err
	}

	uploadInfo.stat = stat
	info, err := u.initUpload(ctx, uploadInfo, PartSize)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	totalParts := int(getNumberOfChunks(stat.Size(), info.PartSize))
	parts := make(chan *part, intMin(totalParts, 8))
	report := make(chan *part, intMin(totalParts, 8))
	wg.Add(totalParts)

	go getParts(stat.Size(), info.PartSize, parts)
	for i := 0; i < intMin(totalParts, 8); i++ {
		go u.processPart(ctx, info, uploadInfo, parts, report)
	}
	go func() {
		for p := range report {
			if p.Error != nil {
				fmt.Println("Failed to upload part... ", p.Error)
				p.Error = nil
				parts <- p
			} else {
				wg.Done()
			}
		}
	}()

	wg.Wait()
	close(parts)
	close(report)
	time.Sleep(1 * time.Second)
	return u.uploadFinalize(ctx, info)
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (u *uploadService) processPart(ctx context.Context, info *UploadInitResponse, uploadInfo UploadInfo, in <-chan *part, report chan<- *part) {
	for p := range in {
		fmt.Println("Got part: ", p.ID)
		partInit, err := u.partUploadInit(ctx, info, p)
		if err != nil {
			p.Error = err
			report <- p
		}
		err = u.partUpload(ctx, uploadInfo, partInit, p)
		if err != nil {
			p.Error = err
			report <- p
		}
		err = u.partReportUploaded(ctx, info, p)
		if err != nil {
			p.Error = err
			report <- p
		}

		report <- p
		//wg.Done()
		fmt.Println("Processed part: ", p.ID)
	}
	fmt.Println("Finishing process part worker")
}

func getParts(fileSize int64, partSize int64, out chan<- *part) {
	start := int64(0)
	end := int64(0)
	count := 1
	for end < fileSize {
		end = start + partSize
		if end > fileSize {
			end = fileSize
		}
		p := &part{
			ID:        count,
			StartByte: start,
			EndByte:   end,
		}
		out <- p
		fmt.Println("Generated part: ", p.ID)
		count++
		start = end
	}
}

func (u *uploadService) initUpload(ctx context.Context, info UploadInfo, partSize int64) (*UploadInitResponse, error) {
	var uploadName string
	var overwrite string

	if info.Name == "" {
		uploadName = filepath.Base(info.Path)
	} else {
		uploadName = info.Name
	}

	if info.Overwrite {
		overwrite = "true"
	} else {
		overwrite = "false"
	}

	uploadInfo := map[string]interface{}{
		"project":   info.Project,
		"name":      uploadName,
		"part_size": partSize,
		"size":      info.stat.Size(),
	}
	initResponse := new(UploadInitResponse)
	_, err := u.Do(
		ctx,
		url.AddPath("/upload/multipart"),
		query.Add("overwrite", overwrite),
		body.JSON(uploadInfo),
		responsebody.JSON(initResponse),
	)
	return initResponse, err
}

func (u *uploadService) partUploadInit(ctx context.Context, info *UploadInitResponse, p *part) (*PartUploadInitResponse, error) {
	m := new(PartUploadInitResponse)
	_, err := u.Do(
		ctx,
		url.AddPath("/upload/multipart/:uploadID/part/:partID"),
		url.Params(map[string]string{
			"uploadID": info.UploadID,
			"partID":   fmt.Sprintf("%d", p.ID),
		}),
		responsebody.JSON(m),
	)
	return m, err
}

func (u *uploadService) partUpload(ctx context.Context, uploadInfo UploadInfo, info *PartUploadInitResponse, p *part) error {
	f, err := os.Open(uploadInfo.Path)
	if err != nil {
		return err
	}
	buff := make([]byte, p.EndByte-p.StartByte)
	f.Seek(p.StartByte, os.SEEK_SET)
	f.Read(buff)

	resp, err := u.Do(
		ctx,
		url.URL(info.URL),
		headers.Method(info.Method),
		body.Reader(bytes.NewReader(buff)),
	)
	if err != nil {
		return err
	}
	p.ETag = resp.Header.Get("ETag")

	f.Close()
	return nil
}

func (u *uploadService) partReportUploaded(ctx context.Context, info *UploadInitResponse, p *part) error {
	data := map[string]interface{}{
		"part_number": p.ID,
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"ETag": p.ETag,
			},
		},
	}
	resp, err := u.Do(
		ctx,
		url.AddPath("/upload/multipart/:uploadID/part/"),
		url.Param("uploadID", info.UploadID),
		headers.Method("POST"),
		body.JSON(data),
	)
	fmt.Println("Submit part status code:", resp.StatusCode)
	return err
}

func (u *uploadService) uploadFinalize(ctx context.Context, info *UploadInitResponse) error {
	resp, err := u.Do(
		ctx,
		headers.Method("POST"),
		url.AddPath("/upload/multipart/:uploadID/complete"),
		url.Param("uploadID", info.UploadID),
		headers.Set("Content-Type", "application/json"),
		headers.Set("Accept", "application/json"),
	)
	if err != nil {
		return err
	}
	fmt.Println("Complete upload status code: ", resp.StatusCode)
	return nil
}
