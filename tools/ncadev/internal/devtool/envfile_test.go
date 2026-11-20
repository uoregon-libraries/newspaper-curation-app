package devtool

import (
	"slices"
	"testing"
)

func TestParseEnvLine(t *testing.T) {
	var tests = map[string]struct {
		line  string
		key   string
		value string
		ok    bool
	}{
		"simple":          {"FOO=bar", "FOO", "bar", true},
		"spaces":          {"  FOO = bar  ", "FOO", "bar", true},
		"export":          {"export FOO=bar", "FOO", "bar", true},
		"reference":       {"URL=${SCHEME}://${HOST}", "URL", "${SCHEME}://${HOST}", true},
		"double quotes":   {`FOO="bar # baz"`, "FOO", "bar # baz", true},
		"single quotes":   {`FOO='bar'`, "FOO", "bar", true},
		"inline comment":  {"FOO=bar # baz", "FOO", "bar", true},
		"hash in value":   {"FOO=bar#baz", "FOO", "bar#baz", true},
		"empty value":     {"FOO=", "FOO", "", true},
		"comment":         {"# FOO=bar", "", "", false},
		"commented value": {"#FOO=bar", "", "", false},
		"blank":           {"   ", "", "", false},
		"no equals":       {"FOO", "", "", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var k, v, ok = parseEnvLine(tc.line)
			if k != tc.key || v != tc.value || ok != tc.ok {
				t.Errorf("parseEnvLine(%q) = (%q, %q, %v), want (%q, %q, %v)", tc.line, k, v, ok, tc.key, tc.value, tc.ok)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	var vars = map[string]string{"HOST": "localhost", "EMPTY": ""}
	var lookup = func(k string) (string, bool) {
		var v, ok = vars[k]
		return v, ok
	}

	var tests = map[string]struct {
		in   string
		want string
	}{
		"plain":                {"http://${HOST}", "http://localhost"},
		"no braces":            {"http://$HOST/x", "http://localhost/x"},
		"unset":                {"x${NOPE}y", "xy"},
		"colon default unset":  {"${NOPE:-8080}", "8080"},
		"colon default empty":  {"${EMPTY:-8080}", "8080"},
		"colon default set":    {"${HOST:-other}", "localhost"},
		"dash default unset":   {"${NOPE-8080}", "8080"},
		"dash default empty":   {"${EMPTY-8080}", ""},
		"required is a lookup": {"${HOST:?must be set}", "localhost"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got = expand(tc.in, lookup)
			if got != tc.want {
				t.Errorf("expand(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveEnv(t *testing.T) {
	var vars = []envVar{
		{"PORT", "3333"},
		{"SUFFIX", ":${PORT}"},
		{"URL", "http://localhost${SUFFIX}"},
		{"LOGS", "${PWD}/logs"},
		{"PORT", "4444"},
	}
	var got = resolveEnv(vars, "/nca")

	// Later lines see the value assigned above them, as compose does, but the
	// last assignment wins for the key itself
	var want = map[string]string{
		"PORT":   "4444",
		"SUFFIX": ":3333",
		"URL":    "http://localhost:3333",
		"LOGS":   "/nca/logs",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestMergeEnv(t *testing.T) {
	var lines = []string{
		"# A comment",
		"PORT=3333",
		"SUFFIX=:${PORT}",
		"#DEBUG=true",
		"",
		"HOST=localhost # where you browse",
	}
	var overrides = []envVar{
		{"PORT", "4333"},
		{"DEBUG", "false"},
		{"HOST", "example.edu"},
	}

	var got = mergeEnv(lines, overrides)
	var want = []string{
		"# A comment",
		"PORT=4333",
		"SUFFIX=:${PORT}",
		"#DEBUG=true",
		"",
		"HOST=example.edu",
		"DEBUG=false",
	}
	if !slices.Equal(got, want) {
		t.Errorf("mergeEnv() = %q, want %q", got, want)
	}

	// Overriding in place means later references see the override
	var _, vars, _ = parseLines(got)
	if r := resolveEnv(vars, "/"); r["SUFFIX"] != ":4333" {
		t.Errorf("SUFFIX resolved to %q, want %q", r["SUFFIX"], ":4333")
	}
}

func parseLines(lines []string) ([]string, []envVar, error) {
	var vars []envVar
	for _, line := range lines {
		if k, v, ok := parseEnvLine(line); ok {
			vars = append(vars, envVar{k, v})
		}
	}
	return lines, vars, nil
}

func TestComposeSubcommand(t *testing.T) {
	var args = []string{"--env-file", "x.env", "-f", "a.yml", "-f", "b.yml", "-p", "proj", "--profile", "tools", "run", "--rm", "cli"}
	var got = composeSubcommand(args)
	var want = []string{"run", "--rm", "cli"}
	if !slices.Equal(got, want) {
		t.Errorf("composeSubcommand() = %q, want %q", got, want)
	}
}
