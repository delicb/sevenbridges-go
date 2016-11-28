package sevenbridges

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/peterhellberg/link"
)

// Link holds information about navigation between pages in SevenBridges API.
type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// intQueryField returns value from URL of Href field in Link and assumes
// that it is int. On any error, 0 is returned.
func (l *Link) intQueryField(field string) int {
	u, err := url.Parse(l.Href)
	if err != nil {
		return 0
	}
	val, _ := strconv.Atoi(u.Query().Get(field))
	return val
}

// Limit returns "limit" query parameter value from Href field of Link.
func (l *Link) Limit() int {
	return l.intQueryField("limit")
}

// Offset returns "offset" query parameter value from Href field on Link.
func (l *Link) Offset() int {
	return l.intQueryField("offset")
}

// ListOptions specifies optional parameter to various List methods that
// support pagination.
type ListOptions struct {
	Limit  int      `url:"limit,omitempty"`
	Offset int      `url:"offset,omitempty"`
	Fields []string `url:"fields,omitempty,comma"`
}

// IsZero returns true if current value of ListOptions can not be
// distinguished from zero value.
func (lo *ListOptions) IsZero() bool {
	return lo == nil || (lo.Limit == 0 && lo.Offset == 0)
}

// Page is base structure for all paginated responses returned from
// SevenBridges API.
type Page struct {
	TotalMatchingQuery int
	Links              map[string]*Link
}

// NewPage creates and returns Page object initialized from provided Response.
// Link headers from response are read and page information are initialized
// based on Link header value.
func NewPage(resp *http.Response) *Page {
	var totalMatchingQuery int
	if total := resp.Header.Get(headerTotalMatchingQuery); total != "" {
		totalMatchingQuery, _ = strconv.Atoi(total)
	}
	links := map[string]*Link{}
	for rel, l := range link.ParseResponse(resp) {
		links[rel] = &Link{
			Rel:  rel,
			Href: l.URI,
		}
	}
	return &Page{
		TotalMatchingQuery: totalMatchingQuery,
		Links:              links,
	}
}

// generateListOptions creates list options from information in Page structure
// for relation provided in rel. Two values are valid - "next" and "prev"
// as defined by SevenBridges server, but this might change by adding
// first and last page or something similar.
func (p *Page) generateListOptions(rel string) *ListOptions {
	if p == nil {
		return &ListOptions{}
	}
	if rel, ok := p.Links[rel]; ok {
		return &ListOptions{
			Limit:  rel.Limit(),
			Offset: rel.Offset(),
		}
	}
	return &ListOptions{}
}

// NextPage creates and returns ListOptions for accessing next page of
// paginated resource.
func (p *Page) NextPage() *ListOptions {
	return p.generateListOptions("next")
}

// PrevPage creates and returns ListOptions for accessing previous page of
// paginated resource.
func (p *Page) PrevPage() *ListOptions {
	return p.generateListOptions("prev")
}

// HasNextPage returns flag indicating if next page exists for paginated resource.
func (p *Page) HasNextPage() bool {
	return !p.NextPage().IsZero()
}

// HasPrevPage returns flag indicating if previous page exists for paginated resource.
func (p *Page) HasPrevPage() bool {
	return !p.PrevPage().IsZero()
}
