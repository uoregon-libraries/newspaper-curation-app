// Package bashconf is built to read bash-like environmental variables in order
// to maximize cross-language configurability.  Very minimal processing occurs
// here, leaving that to the caller.  i.e., "FOO=1" will yield a key of "FOO"
// and a value of "1" since bash has no types.  This supports only the simplest
// of bash variable assignment: no arrays, no substitutions of other variables,
// just very basic key/value pairs.
package bashconf

import (
	"io/ioutil"
	"strings"
)

// Config is a simple alias for holding the key/value pairs
type Config map[string]string

// ReadFile reads config from a file and returns a Config structure.  If the
// file can't be read or parsed, an error will be returned.
func ReadFile(filename string) (Config, error) {
	var content, err = ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	return ReadString(string(content)), nil
}

// ReadString parses lines in a string into a Config structure.  Any problems
// attempting to parse are ignored.
func ReadString(content string) Config {
	var c = make(Config)
	var lines = strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line[0] == '#' {
			continue
		}

		var kvparts = strings.SplitN(line, "=", 2)
		if len(kvparts) != 2 {
			continue
		}
		var key, val = kvparts[0], kvparts[1]

		if key == "" {
			continue
		}

		// Remove surrounding quotes if any exist, but only one level of quotes
		if val[0] == '"' {
			val = strings.Trim(val, `"`)
		} else if val[0] == '\'' {
			val = strings.Trim(val, `'`)
		}
		c[key] = val
	}

	return c
}
