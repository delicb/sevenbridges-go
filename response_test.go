package sevenbridges_test

import (
	"testing"
	"fmt"
	"net/http"
	"go.sbgenomics.com/sevenbridges-go"
)

func TestNewResponse(t *testing.T) {
	for _, resp := range []*http.Response {
		&http.Response{
			Header: http.Header(map[string][]string{
				"X-Total-Matching-Query": []string{"1"},
			}),
		},
	} {
		sr, err := sevenbridges.NewResponse(resp)
		if err != nil {
			t.Error("Got error: ", err)
		}
		fmt.Println(sr.TotalMatchingQuery)
	}
}
