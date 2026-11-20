// Command ncatest runs NCA's test stack, a separate copy of the dev stack for
// manual and semi-automated testing, and the test recipes that drive it. Build
// it with "make tools" from the NCA root, then run "bin/ncatest help".
package main

import "github.com/uoregon-libraries/newspaper-curation-app/tools/ncadev/internal/devtool"

func main() {
	devtool.Main("ncatest", devtool.NewTestStack)
}
