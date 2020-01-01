package auth

type UserInfo struct {
	Name     string
	PhotoURL string
	Email    string
	Phone    string
	Location string
	Provider string
}

// Service -
type Service interface {
	VerifyToken(token interface{}) (uid string, err error)
	GetUserInfo(uid string) (UserInfo, error)
}
