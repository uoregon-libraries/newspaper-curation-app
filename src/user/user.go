package user

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID    int
	Login string
	Roles []string
}

// New returns an empty user with no roles or ID
func New(login string) *User {
	return &User{Login: login}
}
