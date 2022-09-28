package httpcache

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"
)

// Client makes HTTP requests and caches the results
type Client struct {
	BeforeRequest func(*http.Request)
	HTTPClient    *http.Client
	CachePath     string
	ThrottleMS    int
}

// NewClient initializes a default client for use in reading (and optionally
// caching) from remote servers.  cachepath should be set to the directory
// where cached files' subdirectories reside.
func NewClient(cachepath string, tms int) *Client {
	return &Client{CachePath: cachepath, ThrottleMS: tms}
}

// Get is the most basic function for a Client.  No caching is done, and just
// the response body is returned.  All external fetching eventually lands here.
func (c *Client) Get(u string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	if c.BeforeRequest != nil {
		c.BeforeRequest(req)
	}

	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("non-200 response for GET %s: %s", u, resp.Status)
	}

	if c.ThrottleMS > 0 {
		time.Sleep(time.Millisecond * time.Duration(c.ThrottleMS))
	}

	return resp.Body, nil
}

// GetCached attempts to find a file for the given request, and fetches it from
// its source if the file isn't locally available
func (c *Client) GetCached(r *Request) (io.ReadCloser, error) {
	fullpath, err := c.PrepCacheFile(r)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(fullpath)
	if err == nil {
		return f, nil
	}

	return c.GetAndStore(r.URL, fullpath)
}

// GetCachedBytes functions just like GetCached, but automatically reads all
// bytes from the reader and returns them instead of just returning the reader
func (c *Client) GetCachedBytes(r *Request) ([]byte, error) {
	var body, err = c.GetCached(r)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	return ioutil.ReadAll(body)
}

// ForceGet operates like GetCached except it overwrites a previously cached
// file if one exists
func (c *Client) ForceGet(r *Request) (io.ReadCloser, error) {
	fullpath, err := c.PrepCacheFile(r)
	if err != nil {
		return nil, err
	}
	return c.GetAndStore(r.URL, fullpath)
}

// ForceGetBytes functions just like ForceGet, but automatically reads all
// bytes from the reader and returns them instead of just returning the reader
func (c *Client) ForceGetBytes(r *Request) ([]byte, error) {
	var body, err = c.ForceGet(r)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	return ioutil.ReadAll(body)
}

// PrepCacheFile ensures the directory a request will store its cached file
// exists or else can be created, and returns the full path to the file to be
// cached if successful
func (c *Client) PrepCacheFile(r *Request) (string, error) {
	if c.CachePath == "" {
		return "", fmt.Errorf("CachePath hasn't been initialized")
	}

	fullpath := r.CachePath(c.CachePath)
	dir := path.Dir(fullpath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", fmt.Errorf("unable to create directory '%s': %w", dir, err)
	}

	return fullpath, nil
}

// GetAndStore downloads an external file and stores it at the given path
func (c *Client) GetAndStore(u, filepath string) (io.ReadCloser, error) {
	body, err := c.Get(u)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(filepath)
	if err != nil {
		body.Close()
		return nil, fmt.Errorf("unable to cache URL request: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, body)
	body.Close()
	if err != nil {
		return nil, fmt.Errorf("unable to cache URL request: %w", err)
	}

	// This is really stupid, but when I built this library apparently I didn't
	// bother to figure out how to rewind the http body.  And hey, why start
	// doing things the right way at this point?
	return os.Open(filepath)
}
