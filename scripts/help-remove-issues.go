// help-remove-issues.go does a few tasks to help with rebuilding batches that
// have Vernonia Eagle issues we need removed:
//
// - "Unrolls" a doc full of "<date a> thru <date b>" to generate issue keys
// - Invokes `bin/find-issues` to get a full feed of valid issues within the
//   various date ranges
// - Parses the issue feed to produce the precise commands needed for the issue
//   removal tool (see https://github.com/uoregon-libraries/batch-issue-remover)
//
// If this kind of thing needs to happen more than once, we'll make this script
// more general-case / customizable, or else build something "real" into NCA.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 5 {
		log.Fatalf("Usage: go run ./scripts/help-remove-issues.go <lccn> <path to unparsed issue dates file> <path to nca dir> <path to batches>")
	}
	var lccn = os.Args[1]
	var inputFile = os.Args[2]
	var pathToNCA = os.Args[3]
	var pathToBatches = os.Args[4]

	var data, err = os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Unable to read file %q: %s", inputFile, err)
	}

	var keys []string
	for _, line := range strings.Split(string(data), "\n") {
		keys = append(keys, parseDate(line)...)
	}

	for i, date := range keys {
		keys[i] = lccn + "/" + date
	}

	err = os.WriteFile("issuekeys", []byte(strings.Join(keys, "\n")), 0640)
	if err != nil {
		log.Fatalf("Unable to write issuekeys: %s", err)
	}

	var cmd = exec.Command("./bin/find-issues", "-c", "settings", "--issue-list", "issuekeys")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	cmd.Run()

	type locData struct {
		Location string
		Batch    string
		Errors   []string
	}

	var locmap = make(map[string][]*locData)
	err = json.Unmarshal(buf.Bytes(), &locmap)
	if err != nil {
		log.Fatalf("Unable to unmarshal find-issues output: %s", err)
	}

	var keysByBatch = make(map[string][]string)
	var batchedKeys = make(map[string]bool)
	var unbatchedKeys = make(map[string]bool)
	for key, locs := range locmap {
		for _, loc := range locs {
			if loc.Location == "" {
				log.Fatalf("Issue key %q must be bad: no location data", key)
			}
			if loc.Batch == "" {
				unbatchedKeys[key] = true
				continue
			}
			batchedKeys[key] = true
			keysByBatch[loc.Batch] = append(keysByBatch[loc.Batch], key)
		}
	}

	for k := range unbatchedKeys {
		if batchedKeys[k] == false {
			log.Printf("Warning: issue key %q doesn't exist in any batches", k)
		}
	}
	for k := range batchedKeys {
		if unbatchedKeys[k] == false {
			log.Printf("Warning: issue key %q would be removed without being replaced", k)
		}
	}

	// Make the output consistent
	var names []string
	for batchname := range keysByBatch {
		names = append(names, batchname)
	}
	sort.Strings(names)

	// Print out commands for creating the replacement batches first - this is a
	// non-destructive operation
	fmt.Println("#####")
	fmt.Println("# Run on NCA server")
	fmt.Println("#####")
	fmt.Println()
	fmt.Println("# Create batches that replace existing batches but without the bad issues")
	fmt.Printf("cd %s\n", pathToNCA)
	for _, batchname := range names {
		var currentVersion = batchname[len(batchname)-2:]
		var vnum, _ = strconv.ParseInt(currentVersion, 10, 64)
		var newname = batchname[:len(batchname)-2] + fmt.Sprintf("%02d", vnum+1)

		fmt.Printf("./bin/remove-issues %s %s %s\n",
			filepath.Join(pathToBatches, batchname),
			filepath.Join(pathToBatches, newname),
			strings.Join(keysByBatch[batchname], " "),
		)
	}

	// Next print out commands to run on ONI server, which actually remove the old batch and replace it
	fmt.Println()
	fmt.Println("#####")
	fmt.Println("# Run on ONI server")
	fmt.Println("#####")
	fmt.Println()
	fmt.Println("# Unload and then replace batches")
	fmt.Println("# NOTE: your site will be missing every issue in each batch until the new batch loads,")
	fmt.Println("# but nothing is irreversibly lost at this point.")
	fmt.Println()
	fmt.Println("cd /opt/openoni")
	fmt.Println("source ENV/bin/activate")
	for _, batchname := range names {
		var currentVersion = batchname[len(batchname)-2:]
		var vnum, _ = strconv.ParseInt(currentVersion, 10, 64)
		var newname = batchname[:len(batchname)-2] + fmt.Sprintf("%02d", vnum+1)

		fmt.Printf("./manage.py purge_batch %s\n", batchname)
		fmt.Printf("./manage.py load_batch %s\n", filepath.Join(pathToBatches, newname))
	}

	// Finally print out the commands to remove the original batch from disk.
	// These are *final*: after this there is no going back.
	fmt.Println()
	fmt.Println("#####")
	fmt.Println("# Run on whichever server can remove entire batches")
	fmt.Println("#####")
	fmt.Println()
	fmt.Println("# Remove batches from disk")
	fmt.Println("# NOTE: These operations ARE PERMANENT and CANNOT BE UNDONE.")
	fmt.Println()
	for _, batchname := range names {
		fmt.Printf("rm -rf %s\n", filepath.Join(pathToBatches, batchname))
	}
}

func parseDate(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	if strings.Contains(line, " thru ") {
		return parseRange(line)
	}

	// Parse various possible formats
	var layouts = []string{"2006-01-02", "02-Jan-2006", "Jan. _2, 2006"}
	for _, layout := range layouts {
		var dt, err = time.Parse(layout, line)
		if err == nil {
			return []string{dt.Format("2006-01-02")}
		}
	}

	log.Fatalf("Unable to parse line %q with any format", line)
	return nil
}

func parseRange(line string) []string {
	var parts = strings.Split(line, " thru ")
	var start, err = time.Parse("2006-01-02", parts[0])
	if err != nil {
		log.Fatalf("Invalid line %q: date %q is unparseable: %s", line, parts[0], err)
	}
	var end time.Time
	end, err = time.Parse("2006-01-02", parts[1])
	if err != nil {
		log.Fatalf("Invalid line %q: date %q is unparseable: %s", line, parts[1], err)
	}

	var out []string
	for start.Before(end) {
		out = append(out, start.Format("2006-01-02"))
		start = start.AddDate(0, 0, 1)
	}
	out = append(out, end.Format("2006-01-02"))

	return out
}
