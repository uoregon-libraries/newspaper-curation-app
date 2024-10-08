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
func getRPC(t *testing.T) *RPC {
	var r, err = New("foo:2222")
	if err != nil {
		t.Fatalf("Unable to provision new RPC: %s", err)
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
			var r = getRPC(t)

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
			var r = getRPC(t)
			r.call = func(params []string) (data []byte, err error) {
				if params[0] != cmd {
					t.Fatalf("Batch load called, but param 1 was %q, not %q", params[0], cmd)
				}
				if params[1] != tc.batch {
					t.Fatalf("Batch load called, but param 2 was %q, not %q", params[1], tc.batch)
				}
				if len(params) != 2 {
					t.Fatalf("Batch load called, but got %d params instead of 2", len(params))
				}

				return []byte(tc.json), nil
			}

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
			var r = getRPC(t)
			r.call = func(params []string) (data []byte, err error) {
				if params[0] != cmd {
					t.Fatalf("Batch purge called, but param 1 was %q, not %q", params[0], cmd)
				}
				if params[1] != tc.batch {
					t.Fatalf("Batch purge called, but param 2 was %q, not %q", params[1], tc.batch)
				}
				if len(params) != 2 {
					t.Fatalf("Batch purge called, but got %d params instead of 2", len(params))
				}

				return []byte(tc.json), nil
			}

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
			var r = getRPC(t)
			r.call = func(params []string) (data []byte, err error) {
				if params[0] != cmd {
					t.Fatalf("Get job logs called, but param 1 was %q, not %q", params[0], cmd)
				}
				if params[1] != idstr {
					t.Fatalf("Get job logs called, but param 2 was %q, not %q", params[1], idstr)
				}
				if len(params) != 2 {
					t.Fatalf("Get job logs called, but got %d params instead of 2", len(params))
				}

				return []byte(tc.json), nil
			}

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
