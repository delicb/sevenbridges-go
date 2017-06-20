package sevenbridges

import (
	"context"

	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/delicb/cliware-middlewares/headers"
	"github.com/delicb/cliware-middlewares/responsebody"
	curl "github.com/delicb/cliware-middlewares/url"
	"github.com/delicb/gwc"
)

const (
	// KB is number of bytes in KiloByte
	KB = 1024
	// MB is number of bytes in MegaByte
	MB = 1024 * KB
	// GB is number of bytes in GigaByte
	GB = 1024 * MB
	// TB is number of byte in TeraByte
	TB = 1024 * GB

	// PartSize is default number of bytes in one part.
	PartSize = 10 * MB
)

// DownloadInfo holds information on where file from SevenBridges platform
// can be download from.
type DownloadInfo struct {
	URL string `json:"url"`
}

// DownloadService is service for downloading files from SevenBridges platform.
type DownloadService interface {
	Info(ctx context.Context, fileID string) (*DownloadInfo, *Response, error)
	Download(ctx context.Context, fileID, destination string) error
}

type downloadService struct {
	*service
}

func newDownloadService(client gwc.Doer) *downloadService {
	return &downloadService{newService(client)}
}

var _ DownloadService = new(downloadService)

func (d *downloadService) Info(ctx context.Context, fileID string) (*DownloadInfo, *Response, error) {
	di := new(DownloadInfo)
	resp, err := d.Do(
		ctx,
		curl.AddPath("/files/:fileID/download_info"),
		curl.Param("fileID", fileID),
		responsebody.JSON(di),
	)
	return di, resp, err
}

type chunk struct {
	StartByte  int64
	EndByte    int64
	PartNumber int64
	Error      error
}

func getNumberOfChunks(totalSize, partSize int64) int64 {
	return int64(math.Ceil(float64(totalSize) / float64(partSize)))
}

func generateChunks(totalParts, totalSize, partSize int64, chunks chan<- chunk) {
	start := int64(0)
	end := int64(PartSize - 1)

	for i := int64(0); i < totalParts; i++ {
		chunks <- chunk{StartByte: start, EndByte: end, PartNumber: i}
		start = end + 1
		end = start + partSize + 1
		if end > totalSize {
			end = totalSize
		}
	}
}

func (d *downloadService) Download(ctx context.Context, fileID, dst string) error {
	info, _, err := d.Info(ctx, fileID)
	if err != nil {
		return err
	}

	// start generating chunks
	chunks := make(chan chunk, 16)
	report := make(chan chunk, 16)
	var wg sync.WaitGroup

	size := fileSize(info.URL)
	totalChunks := getNumberOfChunks(size, PartSize)
	wg.Add(int(totalChunks))

	// start generating chunks
	go generateChunks(totalChunks, size, PartSize, chunks)

	// start download workers
	for i := 0; i < 16; i++ {
		go d.downloadChunk(ctx, dst, info.URL, chunks, report, &wg)
	}
	// start report monitoring to reschedule chunks that have failed
	go func() {
		for chunk := range report {
			if chunk.Error != nil {
				fmt.Println("Fetching chunk failed, retry: ", chunk.Error)
				chunk.Error = nil
				chunks <- chunk
			} else {

			}
		}
	}()
	wg.Wait()
	close(report)
	close(chunks)
	return nil
}

func (d *downloadService) downloadChunk(ctx context.Context, dst, url string, chunks <-chan chunk, report chan<- chunk, wg *sync.WaitGroup) {

	for chunk := range chunks {
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, os.FileMode(0600))
		if err != nil {
			chunk.Error = err
			report <- chunk
			continue
		}
		f.Seek(chunk.StartByte, os.SEEK_SET)
		rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.StartByte, chunk.EndByte)
		_, err = d.Do(
			ctx,
			headers.Method("GET"),
			curl.URL(url),
			headers.Set("Range", rangeHeader),
			responsebody.Writer(f),
		)

		if err != nil {
			chunk.Error = err

		}
		report <- chunk
		f.Close()
		wg.Done()
	}
}

func fileSize(url string) int64 {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	r, _ := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)
	return r
}
