package sevenbridges

import (
	"net/http"

	"encoding/json"
	"io/ioutil"

	"fmt"

	"github.com/google/go-querystring/query"
	c "go.delic.rs/cliware"
	"go.delic.rs/cliware-middlewares/errors"
)

// listOptions adds provided list options to request (in form of query parameters)
func listOptions(listOptions *ListOptions) c.Middleware {
	return c.RequestProcessor(func(req *http.Request) error {
		q := req.URL.Query()
		newValues, err := query.Values(listOptions)
		if err != nil {
			return err
		}
		for k, v := range newValues {
			for _, vv := range v {
				q.Set(k, vv)
			}
		}
		req.URL.RawQuery = q.Encode()
		return nil
	})
}

// paginatedResponse is struct holding fields returned from SevenBridges API
// for paginated resources.
type paginatedResponse struct {
	TotalMatchingQuery int
	Href               string      `json:"href"`
	Links              []*Link     `json:"links"`
	Items              interface{} `json:"items"`
}

// createPaginatedResponse creates new instance of paginatedResponse and sets
// Items field to provided data.
func createPaginatedResponse(data interface{}) *paginatedResponse {
	resp := new(paginatedResponse)
	resp.Items = data
	return resp
}

// pageResponse is middleware that deserializes response for paginated object.
// provided data should be empty instance of list of entities that will be
// populated by this middleware.
func pageResponse(data interface{}) c.Middleware {
	return c.ResponseProcessor(func(resp *http.Response, err error) error {
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		rawData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(rawData, createPaginatedResponse(data))
	})
}

// HTTPError is error returned by SevenBridges API that contains additional
// information about error that occurred.
type HTTPError struct {
	OriginalError *errors.HTTPError
	Info          *ErrorInfo
}

// ErrorInfo holds information about error that occurred on SevenBridges server.
type ErrorInfo struct {
	Status   int    `json:"status"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
}

// Implementation of error interface
func (he *HTTPError) Error() string {
	if he.Info.Message != "" || he.Info.MoreInfo != "" || he.Info.Code != 0 {
		return fmt.Sprintf(
			"%s [Code: %d, Message: %s, More info: %s]",
			he.OriginalError.Error(), he.Info.Code, he.Info.Message, he.Info.MoreInfo,
		)
	}
	return he.OriginalError.Error()
}

// errorHandler enriches HTTP errors with additional information returned from server.
// It takes HTTPError from cliware-middlewares/error package and wraps it to
// its own errors with parsed body that contains additional info. On any other
// error - error is not modified in any way.
// For this middleware to be useful, errors.Error middleware from cliware-middlewares
// should be used as well.
func errorHandler() c.Middleware {
	return c.ResponseProcessor(func(resp *http.Response, err error) error {
		if err != nil {
			if httpError, ok := err.(*errors.HTTPError); ok {
				info := new(ErrorInfo)
				json.Unmarshal(httpError.Body, info)
				return &HTTPError{
					OriginalError: httpError,
					Info:          info,
				}
			}
			return err
		}
		return nil
	})
}

// tokenAuth adds authentication token to every request.
func tokenAuth(token string) c.Middleware {
	return c.RequestProcessor(func(req *http.Request) error {
		req.Header.Set(headerAuthToken, token)
		return nil
	})
}
