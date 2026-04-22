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
	ArchiveDir     string `long:"archive-dir" short:"a" description:"Directory to search for archived batch directories" required:"true"`
	CacheFile      string `long:"cache-file" short:"C" description:"Path to the validation cache JSON file (default: $XDG_CACHE_HOME/nca/archive-batches-validation.json)"`
	SkipValidation bool   `long:"skip-validation" description:"Skip full BagIt validation; only check that the archive directory exists"`
}

var opts _opts

func main() {
	var c = cli.New(&opts)
	c.AppendUsage("Marks live batches as archived if they exist in the given archive directory.")
	c.AppendUsage("Each archive is validated as a BagIt bag (full manifest check) before being " +
		"marked archived. Successful validations are cached locally so cancelled runs don't " +
		"force re-validation.")
	var conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	var cachePath = opts.CacheFile
	if cachePath == "" {
		cachePath, err = defaultCachePath()
		if err != nil {
			log.Fatalf("Error resolving cache path: %s", err)
		}
	}

	var cache *validationCache
	var cacheNote string
	cache, cacheNote, err = loadCache(cachePath)
	if err != nil {
		log.Fatalf("Error loading validation cache: %s", err)
	}
	if cacheNote != "" {
		log.Printf("WARNING: %s", cacheNote)
	}

	var batches []*models.Batch
	batches, err = models.FindLiveBatches()
	if err != nil {
		log.Fatalf("Error querying live batches: %s", err)
	}
	fmt.Printf("Database has %d batch(es) with 'live' status.\n", len(batches))
	if len(batches) == 0 {
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

	fmt.Printf("Matched %d of %d in archive dir %s.\n\n", len(found), len(batches), opts.ArchiveDir)

	if len(notFound) > 0 {
		fmt.Printf("%d live batch(es) not found in archive directory:\n", len(notFound))
		for _, name := range notFound {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println()
	}

	if len(found) == 0 {
		fmt.Println("No live batches found in the archive directory. Check that --archive-dir is correct.")
		return
	}

	// Walk each found batch, validate, show details, and confirm individually
	var stdin = bufio.NewReader(os.Stdin)
	var archived, skipped, invalid, failed int
	for _, b := range found {
		var archivePath = filepath.Join(opts.ArchiveDir, b.FullName)

		var issues []*models.Issue
		issues, err = b.Issues()
		if err != nil {
			fmt.Printf("Cannot load issues for %s: %s\nSkipping.\n\n", b.FullName, err)
			invalid++
			continue
		}

		var xmlCount int
		xmlCount, err = countIssuesInBatchXML(archivePath)
		if err != nil {
			fmt.Printf("Cannot read batch.xml for %s: %s\nSkipping.\n\n", b.FullName, err)
			invalid++
			continue
		}
		if xmlCount != len(issues) {
			fmt.Printf("Issue count mismatch for %s: DB has %d, batch.xml has %d. Skipping.\n\n",
				b.FullName, len(issues), xmlCount)
			invalid++
			continue
		}

		var validStatus string
		if opts.SkipValidation {
			validStatus = "SKIPPED (--skip-validation)"
		} else {
			var ok bool
			validStatus, ok = ensureValidated(b, archivePath, cache)
			if !ok {
				invalid++
				fmt.Println()
				continue
			}
		}

		fmt.Println("----------------------------------------")
		fmt.Printf("Batch:      %s\n", b.FullName)
		fmt.Printf("MARC Org:   %s\n", b.MARCOrgCode)
		fmt.Printf("Created:    %s\n", b.CreatedAt.Format("2006-01-02"))
		fmt.Printf("Went live:  %s\n", b.WentLiveAt.Format("2006-01-02"))
		fmt.Printf("Issues:     %d\n", len(issues))
		fmt.Printf("Archive at: %s\n", archivePath)
		fmt.Printf("Validated:  %s\n", validStatus)
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
	if invalid > 0 {
		fmt.Printf(", %d invalid", invalid)
	}
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
}

// ensureValidated returns a human-readable status string for the batch's
// validation state and a boolean indicating whether the bag is valid (and
// therefore safe to offer for archiving). It consults the cache first; on a
// miss it runs a full BagIt validation and records the result.
func ensureValidated(b *models.Batch, archivePath string, cache *validationCache) (string, bool) {
	var fingerprint, err = tagmanifestFingerprint(archivePath)
	if err != nil {
		fmt.Printf("Cannot read tagmanifest for %s: %s\nSkipping (archive may not be a valid bag).\n", b.FullName, err)
		return "", false
	}

	var entry = cache.lookup(b.FullName)
	if entry != nil && entry.TagFingerprint == fingerprint {
		return fmt.Sprintf("YES (cached %s)", entry.ValidatedAt.Format("2006-01-02")), true
	}

	fmt.Printf("Validating %s (this may take a while) ...\n", b.FullName)
	var start = time.Now()
	var discrepancies []string
	discrepancies, err = validateArchive(archivePath)
	var elapsed = time.Since(start).Round(time.Second)
	if err != nil {
		fmt.Printf("Validation error for %s: %s\nSkipping.\n", b.FullName, err)
		return "", false
	}
	if len(discrepancies) > 0 {
		fmt.Printf("Validation FAILED for %s (%s):\n", b.FullName, elapsed)
		for _, d := range discrepancies {
			fmt.Printf("  - %s\n", d)
		}
		fmt.Println("Skipping.")
		return "", false
	}

	var now = time.Now()
	err = cache.record(b.FullName, &cacheEntry{
		BatchID:        b.ID,
		ArchivePath:    archivePath,
		TagFingerprint: fingerprint,
		ValidatedAt:    now,
	})
	if err != nil {
		log.Printf("WARNING: could not save cache entry for %s: %s", b.FullName, err)
	}
	return fmt.Sprintf("YES (just now, %s)", elapsed), true
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
