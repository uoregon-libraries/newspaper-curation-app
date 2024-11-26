package sftpgo

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/tidwall/sjson"
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
		var r = mrand.New(mrand.NewSource(time.Now().UnixNano()))
		_, _ = r.Read(data)
	}

	return hex.EncodeToString(data)
}

// API is used to send API requests to the SFTPGo daemon
type API struct {
	url     *url.URL
	apikey  string
	now     func() time.Time
	do      func(c *http.Client, req *http.Request) ([]byte, error)
	rndPass func() string
	LastErr error
}

// New returns a new API instance for sending requests to SFTPGo
func New(apiURL *url.URL, apikey string) (*API, error) {
	if apiURL == nil {
		return nil, fmt.Errorf("cannot instantiate sftpgo.API with no URL")
	}

	var a = &API{
		apikey:  apikey,
		url:     apiURL,
		now:     time.Now,
		rndPass: rndPass,
	}
	a.do = a._do

	return a, nil
}

// CreateUser adds a new user to the sftpgo daemon with the given password and
// description.  If pass is empty, a random password is generated.  The
// password and any errors are returned.
func (a *API) CreateUser(user, pass string, quota int64, desc string) (password string, err error) {
	if a.LastErr != nil {
		return "", fmt.Errorf("creating user: uninitialized sftpgo.API instance: %w", a.LastErr)
	}

	password = pass
	if password == "" {
		password = a.rndPass()
	}

	var u = User{
		Status:      1,
		Username:    user,
		Password:    password,
		Description: desc,
		Permissions: map[string][]string{"/": {"*"}},
		QuotaSize:   quota,
	}

	// JSON errors only occur with complex types that can't be marshaled, so this
	// error can be safely ignored
	var userData, _ = json.Marshal(u)
	_, err = a.rpc("POST", "users", string(userData))

	return password, err
}

// GetUser calls the SFTPGo API to retrieve some information about the given user.
//
// Note that SFTPGo does not return raw password data.  Passwords can be reset
// but never viewed.
func (a *API) GetUser(user string) (u *User, err error) {
	if a.LastErr != nil {
		return nil, fmt.Errorf("retrieving user: uninitialized sftpgo.API instance: %w", a.LastErr)
	}

	u = &User{}
	var data []byte
	data, err = a.rpc("GET", path.Join("users", user), "")
	if err != nil {
		return nil, fmt.Errorf("unable to request user from SFTPGo: %w", err)
	}

	err = json.Unmarshal(data, u)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal user JSON: %w", err)
	}

	return u, err
}

// UpdateUser tells SFTPGo to change the password and/or quota for a
// publisher's SFTP user
func (a *API) UpdateUser(user, pass string, quota int64) error {
	if a.LastErr != nil {
		return fmt.Errorf("updating user: uninitialized sftpgo.API instance: %w", a.LastErr)
	}

	// Get the raw user JSON and modify it - SFTPGo will reset *all fields* we
	// omit in a PUT request. The simple "User" type works great for creation
	// (since we want the default values) and retrieval (we only display a few
	// fields in NCA). But for updates, we have to get the full user record and
	// carefully modify it.
	var data, err = a.rpc("GET", path.Join("users", user), "")
	if err != nil {
		return fmt.Errorf("unable to request user from SFTPGo: %w", err)
	}

	data, err = sjson.SetBytes(data, "quota_size", quota)
	if err == nil {
		data, err = sjson.SetBytes(data, "password", pass)
	}
	if err != nil {
		return fmt.Errorf("error setting user data: %w", err)
	}

	_, err = a.rpc("PUT", path.Join("users", user), string(data))
	return err
}

func (a *API) rpc(method, function string, data string) ([]byte, error) {
	var endpoint = *a.url
	endpoint.Path = path.Join(endpoint.Path, function)

	var c = &http.Client{Timeout: time.Minute}
	var req, err = http.NewRequest(method, endpoint.String(), bytes.NewBuffer([]byte(data)))
	if err != nil {
		return nil, err
	}

	// Supply API key with all requests
	req.Header.Set("X-SFTPGO-API-KEY", a.apikey)

	return a.do(c, req)
}

func (a *API) _do(c *http.Client, req *http.Request) ([]byte, error) {
	var resp, err = c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode >= 400 {
		return data, fmt.Errorf("sftpgo server returned an unsuccessful operation: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return data, err
}

// GetStatus requests SFTPgo server status to verify API key use is successful
func (a *API) GetStatus() error {
	if a.LastErr != nil {
		return fmt.Errorf("getting status: uninitialized sftpgo.API instance: %w", a.LastErr)
	}

	var _, err = a.rpc("GET", "status", "")
	if err != nil {
		return fmt.Errorf("Unable to retrieve server status: %w", err)
	}

	return nil
}
