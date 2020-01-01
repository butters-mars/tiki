package auth

import "context"

type UserInfo struct {
	Name          string
	PhotoURL      string
	Email         string
	Phone         string
	Location      string
	Provider      string
	Disabled      bool
	EmailVerified bool
}

// Service -
type Service interface {
	VerifyToken(ctx context.Context, token interface{}) (uid string, err error)
	GetUserInfo(ctx context.Context, uid string) (UserInfo, error)
}
