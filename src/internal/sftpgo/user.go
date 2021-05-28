package sftpgo

// User is a structured object for SFTPGo JSON requests / responses
type User struct {
	Status      int                 `json:"status,omitempty"`
	Username    string              `json:"username,omitempty"`
	Password    string              `json:"password,omitempty"`
	Description string              `json:"description,omitempty"`
	Permissions map[string][]string `json:"permissions,omitempty"`
}
