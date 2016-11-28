package sevenbridges

import (
	"net/http"

	"context"

	c "go.delic.rs/cliware"
	"go.delic.rs/cliware-middlewares/errors"
	"go.delic.rs/cliware-middlewares/headers"
	"go.delic.rs/cliware-middlewares/url"
	"go.delic.rs/gwc"
)

const (
	libraryVersion = "0.1"
	userAgent      = "go-sevenbridges/" + libraryVersion

	headerRateLimit          = "X-RateLimit-Limit"
	headerRateRemaining      = "X-RateLimit-Remaining"
	headerRateReset          = "X-RateLimit-Reset"
	headerAuthToken          = "X-Sbg-Auth-Token"
	headerTotalMatchingQuery = "X-Total-Matching-Query"
)

// SevenBridges is main entry point for communicating with SevenBridges API.
type SevenBridges struct {
	client   *gwc.Client
	User     UserService
	Project  ProjectService
	Files    FileService
	Download DownloadService
	Upload   UploadService
}

// New returns new instance of SevenBridges that can be used to issue requests
// to Seven Bridges API.
func New(baseURL, token string) *SevenBridges {
	client := gwc.New(
		http.DefaultClient,
		url.URL(baseURL),
		tokenAuth(token),
		headers.Set("User-Agent", userAgent),
		errorHandler(),
		errors.Errors(),
	)

	sb := &SevenBridges{
		client: client,
	}
	sb.User = newUserService(client)
	sb.Project = newProjectService(client)
	sb.Files = newFileService(client)
	sb.Download = newDownloadService(client)
	sb.Upload = newUploadService(client)
	return sb
}

// service is thin wrapper around gwc.Layer with purpose of allowing group of
// endpoints to share same middlewares and provide utility stuff commonly
// needed by most of endpoints.
type service struct {
	*gwc.Layer
}

// newService creates and returns new service that will use provided client
// to send requests.
func newService(client gwc.Doer) *service {
	return &service{gwc.NewLayer(client)}
}

// Do uses client for this service with its specific middlewares and, after
// adding provided middlewares, sends request. Response is converted to
// SevenBridges response and returned.
func (s *service) Do(ctx context.Context, middlewares ...c.Middleware) (*Response, error) {
	resp, err := s.DoCtx(ctx, middlewares...)
	if err != nil {
		return nil, err
	}
	response, err := NewResponse(resp.Response)
	return response, err
}
