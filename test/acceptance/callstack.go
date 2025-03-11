package main

import (
	"runtime"
	"strings"
)

type frame struct {
	file     string
	line     int
	function string
}

func stack() []*frame {
	var pclist = make([]uintptr, 10)

	// Skip 2 frames: Callers itself and this function, stack()
	var n = runtime.Callers(2, pclist)

	// This should be nearly impossible, but better to avoid panics
	if n == 0 {
		return nil
	}

	// Loop to get frames - max of 10 for sanity
	var frames = runtime.CallersFrames(pclist[:n])
	var framelist []*frame
	var more = true
	var f runtime.Frame
	for more && len(framelist) < 10 {
		f, more = frames.Next()
		if strings.HasPrefix(f.Function, "testing.") {
			break
		}
		framelist = append(framelist, &frame{file: f.File, line: f.Line, function: f.Function})

		// Check whether there are more frames to process after this one.
		if !more {
			break
		}
	}
	return framelist
}
