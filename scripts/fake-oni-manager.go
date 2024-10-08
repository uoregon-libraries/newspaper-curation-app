// fake-oni-manager.go is a useful replacement to manage.py for NCA dev setups
// if you don't want to stand up the ONI stack and figure out all the
// inter-dependencies. It simply returns success, delaying a certain time based
// on the job requested, so that there's a sort of real-world feel to it
// (though the delay is very fast compared to the real operations).

package main

import (
	"log/slog"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("You must specify an operation and at least one argument")
		os.Exit(1)
	}

	var op, args = os.Args[1], os.Args[2:]
	switch op {
	case "load_batch":
		var batch = args[0]
		slog.Info("Requested batch load", "batch", batch)
		var dur = time.Second * 15
		slog.Info("Adding fake batch-load delay", "duration", dur)
		time.Sleep(dur)
		slog.Info("Done with fake batch load")
	case "purge_batch":
		var batch = args[0]
		slog.Info("Requested batch purge", "batch", batch)
		var dur = time.Second
		slog.Info("Adding fake batch-purge delay", "duration", dur)
		time.Sleep(dur)
		slog.Info("Done with fake batch purge")
	default:
		slog.Error("Invalid operation", "operation", op)
		os.Exit(1)
	}
}
