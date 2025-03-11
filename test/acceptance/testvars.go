package main

import (
	"fmt"
	"net/url"
	"os"
)

var ncaURL string
var oniStagingURL string
var oniProdURL string
var headlessURL string

func getEnvVars() {
	var urls = map[string]*string{
		"NCA_URL":         &ncaURL,
		"ONI_STAGING_URL": &oniStagingURL,
		"ONI_PROD_URL":    &oniProdURL,
		"HEADLESS_URL":    &headlessURL,
	}

	var errors []string
	for key, uptr := range urls {
		var val = os.Getenv(key)
		var err = validateURL(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Var %q has an invalid value %q: %s\n", key, val, err))
		}

		*uptr = val
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "Cannot run tests due to errors:\n")
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, e)
		}
		os.Exit(-1)
	}
}

func validateURL(s string) error {
	var u, err = url.Parse(s)
	if err != nil {
		return err
	}
	if s == "" {
		return fmt.Errorf("URL is blank")
	}
	if u.Scheme == "" {
		return fmt.Errorf("scheme is blank")
	}
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ws" {
		return fmt.Errorf("scheme (%q) must be http, https, or ws", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("host is blank")
	}

	return nil
}
