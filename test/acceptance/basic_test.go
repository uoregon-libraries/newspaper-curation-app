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
	b.getBody().assertHasText("This text will not appear anywhere in the output")
}

func TestONIProdHome(t *testing.T) {
	var b = newBrowser(t)

	b.visit(oniProdURL)
	b.getBody().assertHasText("This text will not appear anywhere in the output")
}

func TestONIStagingHome(t *testing.T) {
	var b = newBrowser(t)

	b.visit(oniStagingURL)
	b.getBody().assertHasText("This text will not appear anywhere in the output")
}
