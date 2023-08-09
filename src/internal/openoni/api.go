package openoni

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// RPC is used to execute administrative commands against an ONI instance, such
// as batch loading, adding awardees, etc.
type RPC struct {
	uri *url.URL
}

// New returns a new ONI RPC instance for running remote admin commands
func New(baseurl string) (*RPC, error) {
	var u, err = url.Parse(baseurl)
	if err != nil {
		return nil, err
	}
	if u.Host == "" || u.Scheme == "" {
		return nil, fmt.Errorf("host and scheme must both be set")
	}
	return &RPC{uri: u}, nil
}

// rpcURL returns a copy of the base url with its path replaced with the ONI admin
// path and the given function path
func (r *RPC) rpcURL(funcpath string) *url.URL {
	var endpoint = *r.uri
	endpoint.Path = path.Join("api", "admin", funcpath)
	return &endpoint
}

func (r *RPC) get(funcpath string, args map[string]string) (data []byte, response int, err error) {
	var endpoint = r.rpcURL(funcpath)
	var query url.Values
	for k, v := range args {
		query.Set(k, v)
	}
	endpoint.RawQuery = query.Encode()

	return do("GET", endpoint.String(), nil)
}

func (r *RPC) post(funcpath string, args map[string]string) (data []byte, response int, err error) {
	// Ignore the error here - json.Marshal cannot error on a map of primitives
	var postData, _ = json.Marshal(args)
	return do("POST", r.rpcURL(funcpath).String(), postData)
}

func do(method string, uri string, body []byte) (data []byte, response int, err error) {
	var req *http.Request
	var r = bytes.NewBuffer(body)
	req, err = http.NewRequest(method, uri, r)
	if err != nil {
		return nil, -1, err
	}

	var c = &http.Client{Timeout: time.Minute}
	var resp *http.Response
	resp, err = c.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err == nil {
		response = resp.StatusCode
	}
	return data, response, err
}

// LoadBatch sends a command to ONI to load the batch at the given path. This
// path must be absolute, but from the ONI system's perspective.
func (r *RPC) LoadBatch(path string) {
	logger.Errorf("Not implemented")
	// _, _, _ = r.post("admin/batch/load", map[string]string{"batch_path": path})
}

// PurgeBatch sends a command to ONI to purge the batch identified by name.
// This simply requests ONI starts purging the batch, but doesn't wait for the
// command to complete. The returned job data should be queried to determine
// when the job finishes and its status.
func (r *RPC) PurgeBatch(name string) {
	logger.Errorf("Not implemented")
	// _, _, _ = r.post("admin/batch/purge", map[string]string{"batch_name": name})
}

// CheckJobStatus returns the status of an ONI job as well as any messages ONI
// returns in the status request
func (r *RPC) CheckJobStatus(jobid string) {
	logger.Errorf("Not implemented")
	// _, _, _ = r.get("job/status", map[string]string{"job_id": jobid})
}
