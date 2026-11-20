// Command ncadev runs NCA's dev stack: starting it, running one-off commands,
// resetting its data, and so on. Its twin, ncatest (cmd/ncatest), does the
// same for the separate test stack and runs test recipes. Build both with
// "make tools" from the NCA root, then run "bin/ncadev help".
package main

import "github.com/uoregon-libraries/newspaper-curation-app/tools/ncadev/internal/devtool"

func main() {
	devtool.Main("ncadev", devtool.NewDevStack)
}
