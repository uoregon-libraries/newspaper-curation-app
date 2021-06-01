package sftpgo

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"net/http"
	"net/url"
	"path"
	"time"
)

func rndPass() string {
	// Gather 8 random bytes from crypto/rand
	var data = make([]byte, 8)
	var _, err = rand.Read(data)
	// This should realistically be an impossible error: it can only occur if the
	// system is basically broken and /dev/urandom is somehow unreadable.  So
	// instead of failing this process for no good reason, we just inject a
	// regular random call, which is both "good enough" for an sftp password
	// *and* very unlikely to happen anyway.
	if err != nil {
		mrand.Seed(time.Now().UnixNano())
		mrand.Read(data)
	}

	return hex.EncodeToString(data)
}

type token struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// API is used to send API requests to the SFTPGo daemon
type API struct {
	url     *url.URL
	login   string
	pass    string
	token   *token
	now     func() time.Time
	do      func(c *http.Client, req *http.Request) ([]byte, error)
	rndPass func() string
}

// New returns a new API instance for sending requests to SFTPGo
func New(apiURL *url.URL, login, pass string) (*API, error) {
	if apiURL == nil {
		return nil, fmt.Errorf("no API URL specified")
	}

	var a = &API{
		login:   login,
		pass:    pass,
		url:     apiURL,
		now:     time.Now,
		token:   &token{},
		rndPass: rndPass,
	}
	a.do = a._do

	return a, nil
}

// CreateUser adds a new user to the sftpgo daemon with a random password.  The
// password and any errors are returned.
func (a *API) CreateUser(user, desc string) (password string, err error) {
	password = a.rndPass()
	var u = User{
		Status:      1,
		Username:    user,
		Password:    password,
		Description: desc,
		Permissions: map[string][]string{"/": {"*"}},
	}

	// JSON errors only occur with complex types that can't be marshaled, so this
	// error can be safely ignored
	var userData, _ = json.Marshal(u)
	_, err = a.rpc("POST", "users", string(userData))

	return password, err
}

func (a *API) rpc(method, function string, data string) ([]byte, error) {
	var endpoint = *a.url
	endpoint.Path = path.Join(endpoint.Path, function)

	// if function is "token", we have to supply credentials
	if function == "token" {
		endpoint.User = url.UserPassword(a.login, a.pass)
	} else {
		var err = a.getToken()
		if err != nil {
			return nil, err
		}
	}

	var c = &http.Client{Timeout: time.Minute}
	var req, err = http.NewRequest(method, endpoint.String(), bytes.NewBuffer([]byte(data)))
	if err != nil {
		return nil, err
	}

	// If the function is *not* token, we have to supply a bearer token header
	if function != "token" {
		req.Header.Set("Authorization", "Bearer "+a.token.AccessToken)
	}

	return a.do(c, req)
}

func (a *API) _do(c *http.Client, req *http.Request) ([]byte, error) {
	var resp, err = c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(resp.Body)
	if err == nil && resp.StatusCode >= 400 {
		return data, fmt.Errorf("sftpgo server returned an unsuccessful operation: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return data, err
}

func (a *API) getToken() error {
	if a.token.ExpiresAt.Sub(a.now()) > (3 * time.Minute) {
		return nil
	}

	var data, err = a.rpc("GET", "token", "")
	if err != nil {
		return fmt.Errorf("unable to retrieve token: %w", err)
	}

	return json.Unmarshal(data, a.token)
}
