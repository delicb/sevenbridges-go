package sevenbridges

import (
	"context"

	"github.com/delicb/cliware-middlewares/body"
	"github.com/delicb/cliware-middlewares/headers"
	"github.com/delicb/cliware-middlewares/responsebody"
	"github.com/delicb/cliware-middlewares/url"
	"github.com/delicb/gwc"
)

// Project holds information about project on Seven Bridges platform.
type Project struct {
	Href         *string `json:"href"`
	ID           *string `json:"id"`
	Name         *string `json:"name"`
	Type         *string `json:"type"`
	BillingGroup *string `json:"billing_group"`
}

// Member contains information about project member with user info and project
// permission that member has.
type Member struct {
	Href        *string      `json:"href"`
	Username    *string      `json:"username"`
	Permissions *Permissions `json:"permissions"`
}

// Permissions holds set of permissions of a single user on particular project.
type Permissions struct {
	Read    *bool `json:"read"`
	Write   *bool `json:"write"`
	Copy    *bool `json:"copy"`
	Execute *bool `json:"execute"`
	Admin   *bool `json:"admin"`
}

// ProjectCreate is structure that defines body that is required for creating
// new project on SevenBridges platform.
type ProjectCreate struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	BillingGroup *string `json:"billing_group,omitempty"`
}

// ProjectService is interface that defines project related operations available
// on SevenBridges platform.
type ProjectService interface {
	// List returns list of project that current user is member of.
	List(ctx context.Context, opt *ListOptions) ([]*Project, *Response, error)
	// ListForUser returns all projects owned by accessible to a particular user.
	ListForUser(ctx context.Context, username string, opt *ListOptions) ([]*Project, *Response, error)
	// ByID Returns project specified by its ID.
	ByID(ctx context.Context, projectID string) (*Project, *Response, error)
	// Create creates new project with provided information.
	Create(ctx context.Context, pc ProjectCreate) (*Project, *Response, error)
	// Delete removes project from SBG platform, including all stuff stored in
	// project (files, apps, workflows, tasks...).
	Delete(ctx context.Context, projectID string) (*Response, error)
	// Modify edits project with provided ID with provided data. Not all fields
	// for project modification needs to be supplied, only fields that needs to
	// be modified.
	Modify(ctx context.Context, projectID string, pc ProjectCreate) (*Project, *Response, error)
	// Members returns members of provided project
	Members(ctx context.Context, projectID string, opt *ListOptions) ([]*Member, *Response, error)
	// AddMember adds new member to project with provided ID and member
	// information (including permissions).
	AddMember(ctx context.Context, projectID string, member *Member) (*Member, *Response, error)
	// RemoveMember removes user from project membership.
	RemoveMember(ctx context.Context, projectID, username string) (*Response, error)
	// GetMember returns member with provided username from project with provided ID.
	GetMember(ctx context.Context, projectID, username string) (*Member, *Response, error)
	// ChangePermissions updates permissions of a project member in specified
	// project. All fields in provided permissions struct has to be filled,
	// because any unfilled field will be defaulted to false.
	ChangePermissions(ctx context.Context, projectID, username string, permissions Permissions) (*Permissions, *Response, error)
}

type projectService struct {
	*service
}

func newProjectService(client gwc.Doer) ProjectService {
	service := newService(client)
	service.Use(url.AddPath("/projects"))
	return &projectService{service}
}

// just make sure at compile time that projectService implements ProjectService
var _ ProjectService = new(projectService)

func (ps *projectService) List(ctx context.Context, opt *ListOptions) ([]*Project, *Response, error) {
	var p []*Project
	resp, err := ps.Do(
		ctx,
		headers.Method("GET"),
		listOptions(opt),
		pageResponse(&p),
	)
	return p, resp, err
}

func (ps *projectService) ListForUser(ctx context.Context, username string, opt *ListOptions) ([]*Project, *Response, error) {
	var p []*Project
	resp, err := ps.Do(
		ctx,
		headers.Method("GET"),
		listOptions(opt),
		url.AddPath("/"+username),
		pageResponse(&p),
	)
	return p, resp, err
}

func (ps *projectService) ByID(ctx context.Context, projectID string) (*Project, *Response, error) {
	p := new(Project)
	resp, err := ps.Do(
		ctx,
		headers.Method("GET"),
		url.AddPath("/"+projectID),
		responsebody.JSON(p),
	)
	return p, resp, err
}

func (ps *projectService) Create(ctx context.Context, pc ProjectCreate) (*Project, *Response, error) {
	p := new(Project)
	resp, err := ps.Do(
		ctx,
		headers.Method("POST"),
		body.JSON(pc),
		responsebody.JSON(p),
	)
	return p, resp, err
}

func (ps *projectService) Delete(ctx context.Context, projectID string) (*Response, error) {
	return ps.Do(
		ctx,
		headers.Method("DELETE"),
		url.AddPath("/"+projectID),
	)
}

func (ps *projectService) Modify(ctx context.Context, projectID string, pc ProjectCreate) (*Project, *Response, error) {
	p := new(Project)
	resp, err := ps.Do(
		ctx,
		headers.Method("PATCH"),
		url.AddPath("/"+projectID),
		body.JSON(pc),
		responsebody.JSON(p),
	)
	return p, resp, err
}

func (ps *projectService) Members(ctx context.Context, projectID string, opt *ListOptions) ([]*Member, *Response, error) {
	var m []*Member
	resp, err := ps.Do(
		ctx,
		headers.Method("GET"),
		url.AddPath("/:projectID/members"),
		url.Param("projectID", projectID),
		listOptions(opt),
		pageResponse(&m),
	)
	return m, resp, err
}

func (ps *projectService) AddMember(ctx context.Context, projectID string, member *Member) (*Member, *Response, error) {
	m := new(Member)
	resp, err := ps.Do(
		ctx,
		headers.Method("POST"),
		url.AddPath("/:projectID/members"),
		url.Param("projectID", projectID),
		body.JSON(member),
		responsebody.JSON(m),
	)
	return m, resp, err
}

func (ps *projectService) RemoveMember(ctx context.Context, projectID, username string) (*Response, error) {
	return ps.Do(
		ctx,
		headers.Method("DELETE"),
		url.AddPath("/:projectID/members/:username"),
		url.Params(map[string]string{
			"projectID": projectID,
			"username":  username,
		}),
	)
}

func (ps *projectService) GetMember(ctx context.Context, projectID, username string) (*Member, *Response, error) {
	m := new(Member)
	resp, err := ps.Do(
		ctx, headers.Method("GET"),
		url.AddPath("/:projectID/members/:username"),
		url.Params(map[string]string{
			"projectID": projectID,
			"username":  username,
		}),
		responsebody.JSON(m),
	)
	return m, resp, err
}

func (ps *projectService) ChangePermissions(ctx context.Context, projectID, username string, permissions Permissions) (*Permissions, *Response, error) {
	p := new(Permissions)
	resp, err := ps.Do(
		ctx,
		headers.Method("PUT"),
		url.AddPath("/:projectID/members/:username"),
		url.Params(map[string]string{
			"projectID": projectID,
			"username":  username,
		}),
		body.JSON(permissions),
		responsebody.JSON(p),
	)
	return p, resp, err
}
