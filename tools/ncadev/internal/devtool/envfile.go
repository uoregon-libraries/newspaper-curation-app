package devtool

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// envVar is one KEY=VALUE assignment from a compose-style env file
type envVar struct {
	key   string
	value string
}

var inlineComment = regexp.MustCompile(`\s+#.*$`)

// parseEnvLine parses one line of a compose-style env file. Blank lines and
// comments aren't assignments, so ok is false for them. Values keep any
// ${...} references (see expand); quotes around a value are removed, and so
// are comments after an unquoted value.
func parseEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")

	var k, v, found = strings.Cut(line, "=")
	k = strings.TrimSpace(k)
	if !found || k == "" {
		return "", "", false
	}

	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		v = v[1 : len(v)-1]
	} else {
		v = inlineComment.ReplaceAllString(v, "")
	}
	return k, v, true
}

// readEnvFile returns the lines of path, and the assignments among them
func readEnvFile(path string) (lines []string, vars []envVar, err error) {
	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		var line = scanner.Text()
		lines = append(lines, line)
		var k, v, ok = parseEnvLine(line)
		if ok {
			vars = append(vars, envVar{k, v})
		}
	}
	return lines, vars, scanner.Err()
}

// resolveEnv expands each variable in order, the way compose reads an env
// file: a value can refer to variables assigned above it or set in the
// environment. PWD is always root, since that's where we run compose. When a
// key is assigned more than once, the last assignment wins.
func resolveEnv(vars []envVar, root string) map[string]string {
	var resolved = make(map[string]string)
	var lookup = func(k string) (string, bool) {
		if k == "PWD" {
			return root, true
		}
		if v, ok := resolved[k]; ok {
			return v, true
		}
		return os.LookupEnv(k)
	}

	for _, v := range vars {
		resolved[v.key] = expand(v.value, lookup)
	}
	return resolved
}

// expand replaces $VAR and ${VAR} references in s, including compose's
// ${VAR:-default} and ${VAR-default} forms. ${VAR:?error} and ${VAR?error}
// are treated as plain references. A default can't have braces of its own:
// podman-compose reads env files with python-dotenv, which can't either.
func expand(s string, lookup func(string) (string, bool)) string {
	return os.Expand(s, func(ref string) string {
		var i = strings.IndexAny(ref, ":-?")
		if i < 0 {
			var v, _ = lookup(ref)
			return v
		}

		var name, op = ref[:i], ref[i:]
		var v, set = lookup(name)
		switch {
		case strings.HasPrefix(op, ":-"):
			if v == "" {
				return expand(op[2:], lookup)
			}
		case strings.HasPrefix(op, "-"):
			if !set {
				return expand(op[1:], lookup)
			}
		}
		return v
	})
}

// mergeEnv returns lines with each assignment to a key in overrides replaced
// by the override, and any overrides that weren't assigned appended. Replacing
// in place, rather than appending everything, keeps the overrides in effect
// for later lines that refer to them (e.g., a URL built from a port).
func mergeEnv(lines []string, overrides []envVar) []string {
	var byKey = make(map[string]string)
	for _, o := range overrides {
		byKey[o.key] = o.value
	}

	var used = make(map[string]bool)
	var out = make([]string, 0, len(lines)+len(overrides))
	for _, line := range lines {
		var k, _, ok = parseEnvLine(line)
		if v, override := byKey[k]; ok && override {
			line = k + "=" + v
			used[k] = true
		}
		out = append(out, line)
	}

	for _, o := range overrides {
		if !used[o.key] {
			out = append(out, o.key+"="+o.value)
		}
	}
	return out
}
