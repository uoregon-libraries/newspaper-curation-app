//go:build ignore

package main

import (
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/hasher"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

var conf *config.Config

type _opts struct {
	cli.BaseOptions
	Name    string `long:"name" description:"The name you want to give this report" required:"true"`
	TestDir string `long:"dir" description:"Where the test directory is located" required:"true"`
}

var opts _opts
var l = logger.New(logger.Debug, false)

func getOpts() {
	var c = cli.New(&opts)
	conf = c.GetConf()
}

type replacer struct {
	search  *regexp.Regexp
	replace string
}

func (r replacer) ReplaceAllString(s string) string {
	return r.search.ReplaceAllString(s, r.replace)
}
func (r replacer) ReplaceAll(b []byte) []byte {
	return r.search.ReplaceAll(b, []byte(r.replace))
}

var batchRenames []replacer

// cacheBatchData store the information about a batch for renaming report
// output text and filenames
func cacheBatchData() {
	var sql = `
		SELECT
			b.marc_org_code, b.created_at, b.name,
			GROUP_CONCAT(DISTINCT i.lccn ORDER BY i.lccn SEPARATOR ''),
			SUM(i.page_count)
		FROM batches b
		JOIN issues i ON (i.batch_id = b.id) GROUP BY b.id
	`
	var op = dbi.DB.Operation()
	var rows = op.Query(sql)
	if op.Err() != nil {
		l.Fatalf("Unable to query database for batch rename map: %s", op.Err())
	}
	var moc, name, titles, pages string
	var created time.Time
	for rows.Next() {
		rows.Scan(&moc, &created, &name, &titles, &pages)
		if op.Err() != nil {
			l.Fatalf("Unable to query database for batch rename map: %s", op.Err())
		}
		var b = &models.Batch{MARCOrgCode: moc, CreatedAt: created, Name: name}
		var bnormal = &models.Batch{
			MARCOrgCode: moc,
			CreatedAt:   time.UnixMilli(1136243045000),
			Name:        "Pages" + pages + "Titles" + titles,
		}
		bnormal.GenerateFullName()
		batchRenames = append(batchRenames, replacer{
			search:  regexp.MustCompile(regexp.QuoteMeta(b.FullName)),
			replace: bnormal.FullName,
		})
		batchRenames = append(batchRenames, replacer{
			search:  regexp.MustCompile(regexp.QuoteMeta(b.Name)),
			replace: bnormal.Name,
		})
	}
}

// renameBatches finds any occurrence of any batch names in the given text and
// replaces them with a name based on what the batch contains so that testing
// is more accurate. Jobs can run out of order from one test run to the next,
// which has made validation very cumbersome lately.
//
// A batch name will be replaced based on whether it's a partial name or a full
// name. e.g., "T4PineGendenwithaTramplingNightshade" is a partial name and
// "batch_oru_20230101T4PineGendenwithaTramplingNightshade_ver01" would be a
// full name.
//
// The first time this is called, a query is run against the database to cache
// all the batch information we need for renaming.
func renameBatches(s string) string {
	if batchRenames == nil {
		cacheBatchData()
	}

	for _, r := range batchRenames {
		s = r.ReplaceAllString(s)
	}
	return s
}

// These are all here to strip out anything not in the matched group -
// basically datestamps and workflow issue database ids. Matches are replaced
// with the first group only.
var idRegexes = []replacer{
	{regexp.MustCompile(`(..........-..........)-[0-9]+`), "$1"},
	{regexp.MustCompile(`(notouchie-..........)-[0-9]+`), "$1"},
	{regexp.MustCompile(`(split-..........)-[0-9]+`), "$1"},
}

// stripIdentifiers removes things which aren't important to reporting, and
// identify a value in a way that isn't consistent from one run to the next,
// such as database identifiers in the workflow paths
func stripIdentifiers(s string) string {
	s = strings.Replace(s, ".wip-", "XXWIPXX", -1)
	for _, r := range idRegexes {
		s = r.ReplaceAllString(s)
	}
	s = renameBatches(s)

	return s
}

// These are used when scrubbing XML files
var xmlRegexes = []replacer{
	{regexp.MustCompile(`<softwareVersion>.*</softwareVersion>`), `<softwareVersion>XYZZY</softwareVersion>`},
	{regexp.MustCompile(`<fileName>.*</fileName>`), `<fileName>XYZZY</fileName>`},
	{regexp.MustCompile(`\bID="TB\.[^"]*"`), `ID="XYZZY"`},
	{regexp.MustCompile(`<metsHdr CREATEDATE="....-..-..T..:..:..">`), `<metsHdr CREATEDATE="2006-01-02T15:04:05">`},
}

// actionsRegexes removes datestamps and job ids from actions.txt
var actionsRegexes = []replacer{
	{regexp.MustCompile(`on .* [0-9]+, 20[0-9][0-9] at .*:`), `on DAY at TIME:`},
	{regexp.MustCompile(`Job [0-9]+ `), `Job N`},
}

// clean returns a copy of the raw data after running it through the given list
// of replacers
func clean(raw []byte, replacers []replacer) []byte {
	var out = make([]byte, len(raw))
	copy(out, raw)
	for _, r := range replacers {
		out = r.ReplaceAll(out)
	}
	return out
}

// cleanXML returns a copy of the raw data with dates and other identifiers
// scrubbed for easier report diffing
func cleanXML(raw []byte) []byte {
	var cleaned = clean(raw, xmlRegexes)
	cleaned = []byte(renameBatches(string(cleaned)))
	return cleaned
}

// cleanActions returns a copy of `raw` with date, time, and job id scrubbed
func cleanActions(raw []byte) []byte {
	return clean(raw, actionsRegexes)
}

// sqldump returns a string that should match the output of "mysql -Ne ..."
// when in non-interactive mode (redirected to a file for instance)
func sqldump(q string) string {
	var op = dbi.DB.Operation()
	var rows = op.Query(q)

	var cols = rows.Columns()
	if op.Err() != nil {
		l.Fatalf("Unable to read columns: %s", op.Err())
	}

	// Set up all the weird structures we need to deal with an unknown table structure
	var raw = make([]sql.NullString, len(cols))
	var iface = make([]any, len(cols))
	var slist = make([]string, len(cols))
	for i := range raw {
		iface[i] = &raw[i]
	}

	var tsv []string
	for rows.Next() {
		rows.Scan(iface...)
		for i, val := range raw {
			var s = val.String
			if !val.Valid {
				s = "NULL"
			}
			slist[i] = s
		}
		tsv = append(tsv, strings.Join(slist, "\t"))
	}

	if op.Err() != nil {
		l.Fatalf("Error dumping sql with query %q: %s", q, op.Err())
	}

	return strings.Join(tsv, "\n") + "\n"
}

type Path struct {
	absPath    string
	relPath    string
	outputPath string
}

// newPath creates a [Path] from the given base directory and absolute path to
// a file. It generates a relative path, scrubbed of identifiers, for
// outputting a new file or reporting information about a file.
func newPath(basedir, reportDir, pth string) *Path {
	var p = &Path{absPath: pth}
	var tmp = strings.Replace(pth, basedir+"/", "", 1)
	tmp = stripIdentifiers(tmp)
	p.relPath = "./" + tmp

	tmp = strings.Replace(tmp, "/", "__", -1)
	tmp = strings.Replace(tmp, "fakemount__", "", 1)
	p.outputPath = filepath.Join(reportDir, tmp)

	return p
}

func main() {
	getOpts()
	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		l.Fatalf("Error trying to connect to database: %s", err)
	}

	/**
	 * Phase 1: Sanity checks
	 */

	// Set up variables, validate that things exist on disk where we expect them
	var basedir string
	basedir, err = filepath.Abs(opts.TestDir)
	if err != nil {
		l.Fatalf("Failed to get absolute path to report directory: %s", err)
	}
	var repdir = filepath.Join(basedir, opts.Name+"-report")
	if fileutil.Exists(repdir) {
		l.Fatalf("Report directory %q already exists: manually remove or rename it to proceed", repdir)
	}
	var fakemount = filepath.Join(basedir, "fakemount")
	if !fileutil.Exists(fakemount) {
		l.Fatalf("Missing fakemount directory")
	}

	/**
	 * Phase 2: Read files and database entries and process them for reporting
	 */

	// Find all files under the workflow dir and sort them
	var filelist []string
	filepath.Walk(fakemount, func(path string, info fs.FileInfo, err error) error {
		filelist = append(filelist, path)
		return err
	})
	if err != nil {
		l.Fatalf("Unable to search for files in %q: %s", fakemount, err)
	}
	if len(filelist) == 0 {
		l.Fatalf("Unable to search for files in %q: nothing found", fakemount)
	}
	sort.Strings(filelist)

	// Create a complete list of all [Path]s
	var allPaths = make([]*Path, len(filelist))
	for i, pth := range filelist {
		allPaths[i] = newPath(basedir, repdir, pth)
	}

	// Generate the raw data for our manifest
	var manifestEntries []string
	for _, p := range allPaths {
		manifestEntries = append(manifestEntries, p.relPath)
	}
	var manifest = []byte(strings.Join(manifestEntries, "\n") + "\n")

	// Gather XMLs (with dates and IDs faked for ease of compare) to be sure our
	// ALTO conversion isn't busted
	var xmlfiles = make(map[string][]byte)
	for _, p := range allPaths {
		if filepath.Ext(p.outputPath) != ".xml" {
			continue
		}
		var data, err = os.ReadFile(p.absPath)
		if err != nil {
			l.Fatalf("Unable to read %q: %s", p.absPath, err)
		}

		xmlfiles[p.outputPath] = cleanXML(data)
	}

	// Gather and scrub actions.txt files
	var actionFiles = make(map[string][]byte)
	for _, p := range allPaths {
		if filepath.Base(p.absPath) != "actions.txt" {
			continue
		}

		var data, err = os.ReadFile(p.absPath)
		if err != nil {
			l.Fatalf("Unable to read %q: %s", p.absPath, err)
		}
		actionFiles[p.outputPath] = cleanActions(data)
	}

	// Create a TIFF checksum manifest
	var tiffsums []string
	for _, p := range allPaths {
		var ext = filepath.Ext(p.absPath)
		if ext != ".tiff" && ext != ".tif" {
			continue
		}
		var sum, err = hasher.NewMD5().FileSum(p.absPath)
		if err != nil {
			l.Fatalf("Unable to get checksum of %q: %s", p.absPath, err)
		}
		tiffsums = append(tiffsums, sum+"  "+p.relPath)
	}

	// Dump critical info from the database without having useless churn like
	// timestamps or fields that are based on database ids.  This won't cover
	// everything, but it should cover enough to have confidence that an
	// end-to-end test isn't totally hosed.
	var sqlFiles = make(map[string][]byte)
	sqlFiles[filepath.Join(repdir, "dump-batches.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT marc_org_code, name, status, location
		FROM batches
		ORDER BY marc_org_code, name, status
	`)))
	sqlFiles[filepath.Join(repdir, "dump-actions.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT a.object_type, a.action_type, u.login, a.message
		FROM actions a
		LEFT OUTER JOIN users u ON (a.user_id = u.id)
		ORDER BY a.object_type, a.action_type, u.login, a.message
	`)))
	sqlFiles[filepath.Join(repdir, "dump-issues.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT
			marc_org_code, date, date_as_labeled, volume, issue, edition,
			edition_label, page_labels_csv, is_from_scanner, workflow_step,
			location, ignored
		FROM issues
		ORDER BY lccn, date, edition
	`)))
	sqlFiles[filepath.Join(repdir, "dump-jobs.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT p.name, p.description, p.object_type, j.job_type, j.status, j.object_type, j.extra_data
		FROM jobs j
		JOIN pipelines p ON (j.pipeline_id = p.id)
		ORDER BY p.name, p.description, p.object_type, j.job_type, j.status, j.object_type, j.extra_data
	`)))

	/*
	 * Phase 3: Write it all out to disk
	 */

	// Build the file manifest
	os.MkdirAll(repdir, 0755)
	err = os.WriteFile(filepath.Join(repdir, "raw-files.txt"), manifest, 0644)
	if err != nil {
		l.Fatalf("Unable to write file manifest: %s", err)
	}

	// Write out all the scrubbed XML files
	for pth, data := range xmlfiles {
		err = os.WriteFile(pth, data, 0644)
		if err != nil {
			l.Fatalf("Unable to write xml file %q: %s", pth, err)
		}
	}

	// Write out the sanitized action.txt files
	for pth, data := range actionFiles {
		err = os.WriteFile(pth, data, 0644)
		if err != nil {
			l.Fatalf("Unable to write action log %q: %s", pth, err)
		}
	}

	// Write tiff sums
	err = os.WriteFile(filepath.Join(repdir, "tiffsums.txt"), []byte(strings.Join(tiffsums, "\n")+"\n"), 0644)
	if err != nil {
		l.Fatalf("Unable to write TIFF checksum manifest: %s", err)
	}

	// Write out database dumps
	for pth, data := range sqlFiles {
		err = os.WriteFile(pth, data, 0644)
		if err != nil {
			l.Fatalf("Unable to write SQL dump %q: %s", pth, err)
		}
	}

	// TODO: Write database records

	l.Infof("Report will be put into %q", repdir)
}
