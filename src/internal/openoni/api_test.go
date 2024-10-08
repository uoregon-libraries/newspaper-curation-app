package openoni

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	var tests = map[string]struct {
		connection string
		hasError   bool
	}{
		"valid":     {connection: "foo:2222", hasError: false},
		"invalid":   {connection: "foo", hasError: true},
		"bad port":  {connection: "foo:0", hasError: true},
		"text port": {connection: "foo:bar", hasError: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var _, err = New(tc.connection)

			if tc.hasError == true && err == nil {
				t.Fatalf("Expected connection %q to have an error", tc.connection)
			}
			if tc.hasError == false && err != nil {
				t.Fatalf("Expected connection %q to succeed, but got error %s", tc.connection, err)
			}
		})
	}
}

// getRPC returns an RPC, crashing if an error occurs since this is using a
// hard-coded connection string that should always work
func getRPC(t *testing.T, name string, expectedParams []string, jsonOut []byte) *RPC {
	var r, err = New("foo:2222")
	if err != nil {
		t.Fatalf("Unable to provision new RPC: %s", err)
	}
	r.call = func(params []string) (data []byte, err error) {
		if len(expectedParams) > 0 {
			var diff = cmp.Diff(expectedParams, params)
			if diff != "" {
				t.Fatalf("%s called with invalid params: %s", name, diff)
			}
		}

		return jsonOut, nil
	}

	return r
}

func TestDo(t *testing.T) {
	var tests = map[string]struct {
		json     string
		hasError bool
	}{
		"simple":         {json: `{"status": "success"}`, hasError: false},
		"invalid json":   {json: `"status": "success"`, hasError: true},
		"normal error":   {json: `{"status": "error"}`, hasError: true},
		"invalid status": {json: `{"status": "foo"}`, hasError: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var r, _ = New("foo:2222")

			r.call = func(_ []string) (data []byte, err error) {
				if tc.hasError {
					err = errors.New("fake error")
				}
				return []byte(tc.json), err
			}

			var _, err = r.do("param")
			if tc.hasError == true && err == nil {
				t.Fatalf("JSON %q should have given an error", tc.json)
			}
			if tc.hasError == false && err != nil {
				t.Fatalf("JSON %q should have succeeded, but got error %s", tc.json, err)
			}
		})
	}
}

func TestLoadBatch(t *testing.T) {
	var tests = map[string]struct {
		batch       string
		json        string
		jobID       int64
		hasError    bool
		errContains string
	}{
		"valid":          {batch: "foo", json: `{"status": "success", "job": {"id": 101}}`, jobID: 101, hasError: false},
		"error response": {batch: "foo", json: `{"status": "error", "message": "fake err"}}`, hasError: true, errContains: "fake err"},
	}

	var cmd = "load-batch"
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var r = getRPC(t, "LoadBatch", []string{cmd, tc.batch}, []byte(tc.json))

			var id, err = r.LoadBatch(tc.batch)
			if tc.hasError {
				if err == nil {
					t.Fatalf("LoadBatch(%q) (json %q): expected error, got none", tc.batch, tc.json)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("LoadBatch(%q) (json %q): expected error to contain %q, got %s", tc.batch, tc.json, tc.errContains, err)
				}
				return
			}

			if tc.jobID != id {
				t.Fatalf("LoadBatch(%q) (json %q): expected job id %d, got %d", tc.batch, tc.json, tc.jobID, id)
			}
		})
	}
}

func TestPurgeBatch(t *testing.T) {
	var tests = map[string]struct {
		batch       string
		json        string
		jobID       int64
		hasError    bool
		errContains string
	}{
		"valid":          {batch: "foo", json: `{"status": "success", "job": {"id": 101}}`, jobID: 101, hasError: false},
		"error response": {batch: "foo", json: `{"status": "error", "message": "fake err"}}`, hasError: true, errContains: "fake err"},
	}

	var cmd = "purge-batch"
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var r = getRPC(t, "PurgeBatch", []string{cmd, tc.batch}, []byte(tc.json))

			var id, err = r.PurgeBatch(tc.batch)
			if tc.hasError {
				if err == nil {
					t.Fatalf("PurgeBatch(%q) (json %q): expected error, got none", tc.batch, tc.json)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("PurgeBatch(%q) (json %q): expected error to contain %q, got %s", tc.batch, tc.json, tc.errContains, err)
				}
				return
			}

			if tc.jobID != id {
				t.Fatalf("PurgeBatch(%q) (json %q): expected job id %d, got %d", tc.batch, tc.json, tc.jobID, id)
			}
		})
	}
}

func TestGetJobLogs(t *testing.T) {
	var tests = map[string]struct {
		json         string
		expectedLogs []string
		hasError     bool
		errContains  string
	}{
		"simple": {
			json:         `{"status": "success", "job": {"id": 27, "stdout": ["2024-01-01T00:00:00.0000 foo"], "stderr": ["2024-01-01T00:00:00.0000 foo"]}}`,
			expectedLogs: []string{"2024-01-01T00:00:00.0000 foo", "2024-01-01T00:00:00.0000 foo"},
			hasError:     false,
		},
		"sorted logs": {
			json:         `{"status": "success", "job": {"id": 27, "stdout": ["1", "5", "3", "z"], "stderr": ["2", "8", "a", "4"]}}`,
			expectedLogs: []string{"1", "2", "3", "4", "5", "8", "a", "z"},
			hasError:     false,
		},
		"no logs": {
			json:         `{"status": "success", "job": {"id": 27}}`,
			expectedLogs: nil,
			hasError:     false,
		},
		"response with no job data": {
			json:         `{"status": "success"}`,
			expectedLogs: []string{},
			hasError:     true,
			errContains:  `missing "job" object`,
		},
	}

	var cmd = "job-logs"
	var idstr = "27"
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var r = getRPC(t, "GetJobLogs", []string{cmd, idstr}, []byte(tc.json))

			var logs, err = r.GetJobLogs(27)
			if tc.hasError {
				if err == nil {
					t.Fatalf("GetJobLogs(%q) (json %q): expected error, got none", idstr, tc.json)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("GetJobLogs(%q) (json %q): expected error to contain %q, got %s", idstr, tc.json, tc.errContains, err)
				}
				return
			}

			var diff = cmp.Diff(tc.expectedLogs, logs)
			if diff != "" {
				t.Fatalf("GetJobLogs(%q) (json %q): job logs not as expected: %s", idstr, tc.json, diff)
			}
		})
	}
}

func TestGetJobStatus(t *testing.T) {
	var tests = map[string]struct {
		json        string
		status      JobStatus
		hasError    bool
		errContains string
	}{
		"pending": {
			json:     `{"status": "success", "job": {"id": 27, "status": "pending"}}`,
			status:   JobStatusPending,
			hasError: false,
		},

		"success": {
			json:     `{"status": "success", "job": {"id": 27, "status": "successful"}}`,
			status:   JobStatusSuccessful,
			hasError: false,
		},

		"failed": {
			json:     `{"status": "success", "job": {"id": 27, "status": "failed"}}`,
			status:   JobStatusFailed,
			hasError: false,
		},

		"missing job obj": {
			json:        `{"status": "success"}`,
			hasError:    true,
			errContains: `missing "job" object`,
		},

		"invalid status": {
			json:        `{"status": "success", "job": {"id": 27, "status": "foo"}}`,
			hasError:    true,
			errContains: `unknown status`,
		},
	}

	var cmd = "job-status"
	var idstr = "27"
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var r = getRPC(t, "GetJobStatus", []string{cmd, idstr}, []byte(tc.json))

			var st, err = r.GetJobStatus(27)
			if tc.hasError {
				if err == nil {
					t.Fatalf("GetJobStatus(%q) (json %q): expected error, got none", idstr, tc.json)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("GetJobStatus(%q) (json %q): expected error to contain %q, got %s", idstr, tc.json, tc.errContains, err)
				}
				return
			}

			if st != tc.status {
				t.Fatalf("GetJobStatus(%q) (json %q): job status should be %s, got %s", idstr, tc.json, tc.status, st)
			}
		})
	}
}
