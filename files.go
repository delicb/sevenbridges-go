package sevenbridges

import (
	"context"
	"time"

	"github.com/delicb/cliware-middlewares/headers"
	"github.com/delicb/cliware-middlewares/query"
	"github.com/delicb/cliware-middlewares/responsebody"
	"github.com/delicb/cliware-middlewares/url"
	"github.com/delicb/gwc"
)

// Metadata is custom data attached to file.
type Metadata map[string]interface{}

// File contains information about single file on SevenBridges platform.
type File struct {
	Href       string      `json:"href"`
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Size       int64       `json:"size"`
	Project    string      `json:"project"`
	CreatedOn  time.Time   `json:"created_on"`
	ModifiedOn time.Time   `json:"modified_on"`
	Origin     interface{} `json:"origin"` // TODO: what is this?
	Metadata   Metadata    `json:"metadata"`
}

// FileService is interface that defines operations on files on SevenBridges platform.
type FileService interface {
	// List returns users files (single page).
	List(ctx context.Context, projectID string) ([]*File, *Response, error)
	// ByID returns singe file by its ID.
	ByID(ctx context.Context, fileID string) (*File, *Response, error)
	// Delete removes file with provided ID from platform.
	Delete(ctx context.Context, fileID string) (*Response, error)
}

type fileService struct {
	*service
}

func newFileService(client gwc.Doer) FileService {
	return &fileService{newService(client)}
}

func (fs *fileService) List(ctx context.Context, projectID string) ([]*File, *Response, error) {
	var files []*File
	resp, err := fs.Do(
		ctx,
		headers.Method("GET"),
		url.AddPath("/files"),
		query.Add("project", projectID),
		pageResponse(&files),
	)
	return files, resp, err
}

func (fs *fileService) ByID(ctx context.Context, fileID string) (*File, *Response, error) {
	f := new(File)
	resp, err := fs.Do(
		ctx,
		headers.Method("GET"),
		url.AddPath("/files/"+fileID),
		responsebody.JSON(f),
	)
	return f, resp, err
}

func (fs *fileService) Delete(ctx context.Context, fileID string) (*Response, error) {
	return fs.Do(
		ctx,
		headers.Method("GET"),
		url.AddPath("/files/"+fileID),
	)
}
