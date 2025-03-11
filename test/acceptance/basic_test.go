package main

import (
	"testing"
)

func TestMain(m *testing.M) {
	getEnvVars()
	m.Run()
}

func TestNCAHome(t *testing.T) {
	var b = newBrowser(t)

	b.visit(ncaURL)
	b.getBody().assertHasText("Welcome to NCA!")
}

func TestONIProdHome(t *testing.T) {
	var b = newBrowser(t)

	b.visit(oniProdURL)
	b.getBody().assertHasText("Welcome to the YOUR_LONG_PROJECT_NAME")
}

func TestONIStagingHome(t *testing.T) {
	var b = newBrowser(t)

	b.visit(oniStagingURL)
	b.getBody().assertHasText("Welcome to the YOUR_LONG_PROJECT_NAME")
}
