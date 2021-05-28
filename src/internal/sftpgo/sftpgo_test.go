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
	headers  http.Header
	url      string
}

const exampleToken = `{"access_token":"faketoken","expires_at":"2021-01-17T09:32:29Z"}`

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
	// Default to have a valid token response in all cases
	s.responses["token"] = []byte(exampleToken)
	s.errors["token"] = nil
	return s
}

func (s *spy) do(c *http.Client, req *http.Request) ([]byte, error) {
	var function = strings.Replace(req.URL.Path, "/api/v2/", "", 1)
	s.requests = append(s.requests, request{function: function, headers: req.Header, url: req.URL.String()})
	if s.responses[function] == nil {
		return nil, fmt.Errorf("No response for function %q", function)
	}
	return s.responses[function], s.errors[function]
}

// newAPI returns a hacked-up API and its spy (mock?  double?  Meh).  By
// default the API will use the spy to send fake requests, with a token request
// already pre-set-up.  The API's "now" method will also default to a time just
// before the token expiry for easier testing.
func newAPI(t *testing.T) (*API, *spy) {
	var u, _ = url.Parse("http://example.org/api/v2")
	var a, err = New(u, "user", "pass")
	if err != nil {
		t.Errorf("Couldn't instantiate a simple API: %s", err)
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

func TestToken(t *testing.T) {
	var a, s = newAPI(t)
	var err = a.getToken()
	if err != nil {
		t.Fatalf("Couldn't retrieve token: %s", err)
	}
	if a.token.AccessToken != "faketoken" {
		t.Errorf(`Access token should have been "faketoken", but was %q`, a.token.AccessToken)
	}

	var expected = time.Date(2021, 1, 17, 9, 32, 29, 0, time.UTC)
	if a.token.ExpiresAt != expected {
		t.Errorf(`ExpiresAt should have been %q, but was %q`, expected.Format(time.RFC3339Nano), a.token.ExpiresAt.Format(time.RFC3339Nano))
	}

	// Make sure the request was set up properly
	if len(s.requests) != 1 {
		t.Errorf("More requests than expected: %#v", s.requests)
	}
	var r = s.requests[0]
	var expectedURL = "http://user:pass@example.org/api/v2/token"
	var gotURL = r.url
	if expectedURL != gotURL {
		t.Errorf("Expected request URL %q, got %q", expectedURL, gotURL)
	}

	// Make sure the token isn't requested a second time if it hasn't expired
	a.getToken()
	if len(s.requests) != 1 {
		t.Errorf("doToken caused an extra HTTP request despite token still being valid")
	}

	// If the token's about to expire, a new one should be issued
	a.token.ExpiresAt = time.Date(2021, 1, 17, 9, 26, 0, 0, time.UTC)
	a.getToken()
	if len(s.requests) != 2 {
		t.Errorf("doToken should have requested a new token since the current is expired")
	}
}

func TestCreateUser(t *testing.T) {
}
