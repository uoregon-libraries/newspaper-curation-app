// Package openoni provides methods for calling the ONI Agent's RPCs
package openoni

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"golang.org/x/crypto/ssh"
)

// RPC manages the ssh connections to a single ONI Agent
type RPC struct {
	connection string
	call       func([]string) ([]byte, error)
}

// New parses the connection string into a server and port. Its format must be
// <server>:<port>.
func New(connection string) (*RPC, error) {
	var parts = strings.Split(connection, ":")
	if len(parts) != 2 {
		return nil, errors.New("connection must have the form <server>:<port>")
	}

	var port, _ = strconv.Atoi(parts[1])
	if port < 1 {
		return nil, errors.New("connection must contain a valid port number")
	}

	return &RPC{connection: connection}, nil
}

func (r *RPC) defaultCall(params []string) (data []byte, err error) {
	var cfg = &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 10,
	}

	var client *ssh.Client
	client, err = ssh.Dial("tcp", r.connection, cfg)
	if err != nil {
		return data, fmt.Errorf("dialing %q: %w", r.connection, err)
	}

	var s *ssh.Session
	s, err = client.NewSession()
	if err != nil {
		return data, fmt.Errorf("starting ssh session %q: %w", r.connection, err)
	}

	var cmd = strings.Join(params, " ")
	data, err = s.Output(cmd)
	if err != nil {
		return data, fmt.Errorf("sending command %q to server: %w", cmd, err)
	}

	return data, nil
}

func (r *RPC) do(params ...string) (result gjson.Result, err error) {
	if r.call == nil {
		r.call = r.defaultCall
	}

	var data []byte
	data, err = r.call(params)
	if err != nil {
		return result, err
	}

	result = gjson.Parse(string(data))
	var status = result.Get("status").String()
	switch status {
	case "success":
		return result, nil
	case "error":
		return result, fmt.Errorf("calling %q: %s", strings.Join(params, " "), result.Get("message").String())
	default:
		return result, fmt.Errorf("parsing status for call to %q: invalid value %q", strings.Join(params, " "), status)
	}
}

// LoadBatch sends a request to the agent to load a batch into the ONI
// instance. If successful, returns a job id which can be used to query the
// job's status for completion.
func (r *RPC) LoadBatch(name string) (id int64, err error) {
	var result gjson.Result
	result, err = r.do("load-batch", name)
	if err != nil {
		return 0, fmt.Errorf("requesting batch load: %w", err)
	}

	return result.Get("job").Get("id").Int(), nil
}

// PurgeBatch sends a request to the agent to purge the batch from the ONI
// instance. If successful, returns a job id which can be used to query the
// job's status for completion.
func (r *RPC) PurgeBatch(name string) (id int64, err error) {
	var result gjson.Result
	result, err = r.do("purge-batch", name)
	if err != nil {
		return 0, fmt.Errorf("requesting batch purge: %w", err)
	}

	return result.Get("job").Get("id").Int(), nil
}

// GetVersion returns the version string of the ONI Agent
func (r *RPC) GetVersion() (version string, err error) {
	var result gjson.Result
	result, err = r.do("version")
	if err != nil {
		return "", fmt.Errorf("requesting ONI Agent version: %w", err)
	}

	return result.Get("version").String(), nil
}

// EnsureAwardee tells the agent to verify or create the given MOC in ONI
func (r *RPC) EnsureAwardee(moc *models.MOC) (message string, err error) {
	var result gjson.Result
	result, err = r.do("ensure-awardee", moc.Code, moc.Name)
	if err != nil {
		return "", fmt.Errorf("calling ensure-awardee: %w", err)
	}

	return result.Get("message").String(), nil
}

// JobStatus is the "controlled" version of an ONI Agent's job-status response
type JobStatus string

// All valid job statuses¬
const (
	JobStatusPending    JobStatus = "pending"
	JobStatusStarted    JobStatus = "started"
	JobStatusFailStart  JobStatus = "couldn't start"
	JobStatusSuccessful JobStatus = "successful"
	JobStatusFailed     JobStatus = "failed"
)

var validStatuses = []JobStatus{JobStatusPending, JobStatusStarted, JobStatusFailStart, JobStatusSuccessful, JobStatusFailed}

func (js JobStatus) valid() bool {
	for _, status := range validStatuses {
		if js == status {
			return true
		}
	}

	return false
}

// GetJobStatus returns the status response from ONI Agent for the given job id
func (r *RPC) GetJobStatus(id int64) (js JobStatus, err error) {
	var result gjson.Result
	result, err = r.do("job-status", strconv.FormatInt(id, 10))
	if err == nil {
		result = result.Get("job")
		if !result.Exists() {
			err = errors.New(`bad response from service: missing "job" object`)
		}
	}
	if err != nil {
		return js, fmt.Errorf("requesting status for job %d: %w", id, err)
	}

	js = JobStatus(result.Get("status").String())
	if !js.valid() {
		return js, fmt.Errorf("requesting status for job %d: unknown status %q", id, js)
	}
	return js, nil
}

// GetJobLogs returns the list of log entries by combining and sorting the
// job's output streams
func (r *RPC) GetJobLogs(id int64) (logs []string, err error) {
	var result gjson.Result
	result, err = r.do("job-logs", strconv.FormatInt(id, 10))
	if err == nil {
		result = result.Get("job")
		if !result.Exists() {
			err = errors.New(`bad response from service: missing "job" object`)
		}
	}
	if err != nil {
		return logs, fmt.Errorf("requesting logs for job %d: %w", id, err)
	}

	var lines = result.Get("stdout").Array()
	for _, line := range lines {
		logs = append(logs, line.String())
	}
	lines = result.Get("stderr").Array()
	for _, line := range lines {
		logs = append(logs, line.String())
	}

	sort.Strings(logs)
	return logs, nil
}
