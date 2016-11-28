package sevenbridges

import (
	"net/http"
	"strconv"
	"time"
)

// Response is thin wrapper around http.Response holding additional data
// that might be useful, like query options for next and previous page for
// paginated responses.
//
// NextPage, PrevPage and TotalMatchingQuery are not guarantied to be
// populated. It is only populated for paginated responses.
//
// Rate holds information about request rate returned by server for this
// request.
type Response struct {
	*http.Response
	*Page
	*Rate
}

// Rate tracks info about number of requests sent to SevenBridges server and
// time when count will be restarted.
type Rate struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     Timestamp `json:"reset"`
}

// NewResponse constructs SevenBridges response from provided *http.Response
func NewResponse(resp *http.Response) (*Response, error) {

	r := &Response{
		Response: resp,
		Page:     NewPage(resp),
	}
	rate, err := getRateLimit(resp)
	if err != nil {
		return r, err
	}
	r.Rate = rate
	//r.populatePageValues(body)
	totalMatchingQuery, err := getTotalMatchingQuery(resp)
	if err != nil {
		return r, err
	}
	r.TotalMatchingQuery = totalMatchingQuery
	return r, nil
}

// getRateLimit extracts rate limit information from response headers and
// returns populated Rate object.
func getRateLimit(resp *http.Response) (*Rate, error) {
	rate := new(Rate)
	var err error
	if limit := resp.Header.Get(headerRateLimit); limit != "" {
		rate.Limit, err = strconv.Atoi(limit)
	}
	if err != nil {
		return rate, err
	}
	if remaining := resp.Header.Get(headerRateRemaining); remaining != "" {
		rate.Remaining, err = strconv.Atoi(remaining)
	}
	if err != nil {
		return rate, err
	}
	if reset := resp.Header.Get(headerRateReset); reset != "" {
		if v, err := strconv.ParseInt(reset, 10, 64); v != 0 && err == nil {
			rate.Reset = Timestamp{time.Unix(v, 0)}
		}
	}
	return rate, err
}

// getTotalMatchingQuery extracts total matching query from response, converts it
// to integer and returns it.
func getTotalMatchingQuery(resp *http.Response) (int, error) {
	if totalMatchingQuery := resp.Header.Get(headerTotalMatchingQuery); totalMatchingQuery != "" {
		return strconv.Atoi(totalMatchingQuery)
	}
	return 0, nil
}
