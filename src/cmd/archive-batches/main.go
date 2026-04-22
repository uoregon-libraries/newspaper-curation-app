package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type _opts struct {
	cli.BaseOptions
	ArchiveDir string `long:"archive-dir" short:"a" description:"Directory to search for archived batch directories" required:"true"`
}

var opts _opts

func main() {
	var c = cli.New(&opts)
	c.AppendUsage("Marks live batches as archived if they exist in the given archive directory.")
	c.AppendUsage("Searches the archive directory for subdirectories matching each live batch's FullName.")
	var conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	var batches []*models.Batch
	batches, err = models.FindLiveBatches()
	if err != nil {
		log.Fatalf("Error querying live batches: %s", err)
	}
	if len(batches) == 0 {
		fmt.Println("No batches with 'live' status found.")
		return
	}

	// Check which batches exist in the archive directory
	var found []*models.Batch
	var notFound []string
	for _, b := range batches {
		var archivePath = filepath.Join(opts.ArchiveDir, b.FullName)
		var info, statErr = os.Stat(archivePath)
		if statErr == nil && info.IsDir() {
			found = append(found, b)
		} else {
			notFound = append(notFound, b.FullName)
		}
	}

	if len(notFound) > 0 {
		fmt.Printf("%d live batch(es) not found in archive directory:\n", len(notFound))
		for _, name := range notFound {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println()
	}

	if len(found) == 0 {
		fmt.Println("No live batches found in the archive directory.")
		return
	}

	// Walk each found batch, show details, and confirm individually
	fmt.Printf("Found %d live batch(es) in %s\n\n", len(found), opts.ArchiveDir)
	var stdin = bufio.NewReader(os.Stdin)
	var archived, skipped, failed int
	for _, b := range found {
		var issues []*models.Issue
		issues, err = b.Issues()
		if err != nil {
			log.Printf("WARNING: could not load issues for %s: %s", b.FullName, err)
		}

		fmt.Println("----------------------------------------")
		fmt.Printf("Batch:      %s\n", b.FullName)
		fmt.Printf("MARC Org:   %s\n", b.MARCOrgCode)
		fmt.Printf("Created:    %s\n", b.CreatedAt.Format("2006-01-02"))
		fmt.Printf("Went live:  %s\n", b.WentLiveAt.Format("2006-01-02"))
		fmt.Printf("Issues:     %d\n", len(issues))
		fmt.Printf("Archive at: %s\n", filepath.Join(opts.ArchiveDir, b.FullName))
		fmt.Println("----------------------------------------")

		if !prompt(stdin, fmt.Sprintf("Mark %s as archived? [y/N] ", b.FullName)) {
			fmt.Printf("  Skipped: %s\n\n", b.FullName)
			skipped++
			continue
		}

		b.Status = models.BatchStatusLiveArchived
		b.ArchivedAt = time.Now()
		err = b.SaveWithoutAction()
		if err != nil {
			fmt.Printf("  ERROR archiving %s: %s\n\n", b.FullName, err)
			failed++
			continue
		}
		fmt.Printf("  Archived: %s\n\n", b.FullName)
		archived++
	}

	fmt.Printf("Done: %d archived, %d skipped", archived, skipped)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
}

// prompt writes msg, reads a line from r, and returns true only for an
// explicit "y" or "yes" response (case-insensitive)
func prompt(r *bufio.Reader, msg string) bool {
	fmt.Print(msg)
	var line, err = r.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading input: %s", err)
	}
	var answer = strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
