package sevenbridges

import (
	"context"

	"go.delic.rs/cliware-middlewares/responsebody"
	"go.delic.rs/cliware-middlewares/url"
	"go.delic.rs/gwc"
)

// User holds information about user available on SevenBridges platform.
type User struct {
	Href        string `json:"href"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Affiliation string `json:"affiliation"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	Country     string `json:"country"`
	ZipCode     string `json:"zip_code"`
}

// UserService is interface for accessing user information on SevenBridges platform.
type UserService interface {
	// Me returns user information about user whose authentication token has
	// been used to communication with SevenBridges API.
	Me(ctx context.Context) (*User, *Response, error)
	// User returns information about user with provided username.
	User(ctx context.Context, username string) (*User, *Response, error)
}

type userService struct {
	*service
}

func newUserService(client gwc.Doer) UserService {
	return &userService{newService(client)}
}

// just to verify in compile time that userService implements UserService
var _ UserService = new(userService)

func (us *userService) Me(ctx context.Context) (*User, *Response, error) {
	user := new(User)
	resp, err := us.Do(
		ctx,
		url.AddPath("/user"),
		responsebody.JSON(user),
	)
	return user, resp, err
}

func (us *userService) User(ctx context.Context, username string) (*User, *Response, error) {
	user := new(User)
	resp, err := us.Do(
		ctx,
		url.AddPath("/users/:username"),
		url.Param("username", username),
		responsebody.JSON(user),
	)
	return user, resp, err
}
