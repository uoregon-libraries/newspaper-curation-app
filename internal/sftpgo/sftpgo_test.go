package sftpgo

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type request struct {
	function string
	method   string
	headers  http.Header
	url      string
}

const exampleStatus = `{"message":"Server fully operational"}`

type spy struct {
	requests  []request
	responses map[string][]byte
	errors    map[string]error
}

func makeSpy() *spy {
	var s = &spy{
		responses: make(map[string][]byte),
		errors:    make(map[string]error),
	}
	// Default to have a valid API key and successful server status response
	s.responses["status"] = []byte(exampleStatus)
	s.errors["status"] = nil
	return s
}

func (s *spy) do(_ *http.Client, req *http.Request) ([]byte, error) {
	var function = strings.Replace(req.URL.Path, "/api/v2/", "", 1)
	s.requests = append(s.requests, request{
		function: function,
		method:   req.Method,
		headers:  req.Header,
		url:      req.URL.String(),
	})
	if s.responses[function] == nil {
		return nil, fmt.Errorf("No response for function %q", function)
	}
	return s.responses[function], s.errors[function]
}

// newAPI returns a hacked-up API and its spy (mock?  double?  Meh).  By
// default the API will use the spy to send fake requests, with an api request
// already pre-set-up.  The API's "now" method will also default to a time just
// before the token expiry for easier testing.
func newAPI(t *testing.T) (*API, *spy) {
	var u, _ = url.Parse("http://example.org/api/v2")
	var a, err = New(u, "pass")
	if err != nil {
		t.Fatalf("Unable to create sftpgo API: %s", err)
	}
	var s = makeSpy()
	a.now = makeNow(time.Date(2021, 1, 17, 9, 25, 0, 0, time.UTC))
	a.do = s.do
	return a, s
}

func makeNow(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}

func TestApiKey(t *testing.T) {
	var a, s = newAPI(t)
	var err = a.GetStatus()
	if err != nil {
		t.Fatalf("Couldn't retrieve status: %s", err)
	}

	// Make sure the request was set up properly
	if len(s.requests) != 1 {
		t.Errorf("More requests than expected: %#v", s.requests)
	}
	var r = s.requests[0]
	var expectedURL = "http://example.org/api/v2/status"
	var gotURL = r.url
	if expectedURL != gotURL {
		t.Errorf("Expected request URL %q, got %q", expectedURL, gotURL)
	}
}

func TestCreateUser(t *testing.T) {
	var a, s = newAPI(t)
	var fakepass = "password"
	a.rndPass = func() string { return fakepass }
	// Right now the response is basically just ignored, so anything here works
	s.responses["users"] = []byte(`{"foo": "bar"}`)
	s.errors["users"] = nil
	var pass, err = a.CreateUser("fakename", "", 0, "description")
	if err != nil {
		t.Errorf("CreateUser should have had no errors, but it returned %s", err)
	}
	if len(s.requests) != 1 {
		t.Errorf("Expected one request, but got %d", len(s.requests))
	}
	if s.requests[0].function != "users" {
		t.Errorf("Request should have been for the user, but it was %#v", s.requests[1])
	}
	if pass != fakepass {
		t.Errorf("Expected password to be %q, got %q", fakepass, pass)
	}
}

func TestCreateUser_WithPassword(t *testing.T) {
	var a, s = newAPI(t)
	var fakepass = "nonrandom password"
	a.rndPass = func() string { panic("this should not be called") }

	// Right now the response is basically just ignored, so anything here works
	s.responses["users"] = []byte(`{"foo": "bar"}`)
	s.errors["users"] = nil
	var pass, err = a.CreateUser("fakename", fakepass, 0, "description")
	if err != nil {
		t.Errorf("CreateUser should have had no errors, but it returned %s", err)
	}
	if len(s.requests) != 1 {
		t.Errorf("Expected one request, but got %d", len(s.requests))
	}
	if s.requests[0].function != "users" {
		t.Errorf("Request should have been for the user, but it was %#v", s.requests[1])
	}
	if pass != fakepass {
		t.Errorf("Expected password to be %q, got %q", fakepass, pass)
	}
}

func TestUpdateUser(t *testing.T) {
	var a, s = newAPI(t)
	// Right now the response is basically just ignored, so anything here works
	s.responses["users/fakename"] = []byte(`{"foo": "bar"}`)
	s.errors["users/fakename"] = nil
	var err = a.UpdateUser("fakename", "newpass", 0)
	if err != nil {
		t.Errorf("UpdateUser should have had no errors, but it returned %s", err)
	}
	if len(s.requests) != 2 {
		t.Errorf("Expected one requests, but got %d", len(s.requests))
	}
	if s.requests[0].method != "GET" {
		t.Errorf("First request should have been to GET the current user")
	}
	if s.requests[0].function != "users/fakename" {
		t.Errorf("GET request should have been for the user, but it was %#v", s.requests[1])
	}
	if s.requests[1].method != "PUT" {
		t.Errorf("Second request should have been to PUT the updated user, but method was %q", s.requests[1].method)
	}
	if s.requests[1].function != "users/fakename" {
		t.Errorf("POST request should have been for the user, but it was %#v", s.requests[1])
	}
}
