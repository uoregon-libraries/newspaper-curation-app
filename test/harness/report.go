package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/hasher"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// reportCmd writes a report on the workflow tree and the database to
// <dir>/<name>-report, scrubbed of anything that changes from one run to the
// next (timestamps, database IDs, batch names), so two runs' reports can be
// diffed
type reportCmd struct {
	Name string `long:"name" required:"true" description:"the report's name, e.g., \"baseline\""`
	Dir  string `long:"dir" default:"test" description:"directory to write the report in"`
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

// cacheBatchData stores what we need to rename each batch in report output
// text and filenames
func cacheBatchData() {
	var q = `
		SELECT
			b.marc_org_code, b.created_at, b.name, b.version,
			GROUP_CONCAT(DISTINCT i.lccn ORDER BY i.lccn SEPARATOR ''),
			SUM(i.page_count)
		FROM batches b
		JOIN issues i ON (i.batch_id = b.id) GROUP BY b.id
	`
	var op = dbi.DB.Operation()
	var rows = op.Query(q)
	if op.Err() != nil {
		logger.Fatalf("Unable to query database for batch rename map: %s", op.Err())
	}
	var moc, name, titles, pages string
	var version int
	var created time.Time
	for rows.Next() {
		rows.Scan(&moc, &created, &name, &version, &titles, &pages)
		if op.Err() != nil {
			logger.Fatalf("Unable to query database for batch rename map: %s", op.Err())
		}
		var b = &models.Batch{MARCOrgCode: moc, CreatedAt: created, Name: name, Version: version}
		b.GenerateFullName()
		var bnormal = &models.Batch{
			MARCOrgCode: moc,
			CreatedAt:   time.UnixMilli(1136243045000),
			Name:        "Pages" + pages + "Titles" + titles,
			Version:     version,
		}
		bnormal.GenerateFullName()
		batchRenames = append(batchRenames, replacer{
			search:  regexp.MustCompile(regexp.QuoteMeta(b.FullName)),
			replace: bnormal.FullName,
		})
		slog.Info("Rename batch: full name", "source", b.FullName, "dest", bnormal.FullName)
		batchRenames = append(batchRenames, replacer{
			search:  regexp.MustCompile(regexp.QuoteMeta(b.Name)),
			replace: bnormal.Name,
		})
		slog.Info("Rename batch: short name", "source", b.Name, "dest", bnormal.Name)
	}
}

// renameBatches replaces any batch name in s with a name based on what the
// batch contains. Jobs can run in a different order from one run to the next,
// so the same batch can get a different name.
//
// Both partial names (e.g., "T4PineGendenwithaTramplingNightshade") and full
// names (e.g., "batch_oru_20230101T4PineGendenwithaTramplingNightshade_ver01")
// are replaced.
//
// The first call queries the database for every batch's information.
func renameBatches(s string) string {
	if batchRenames == nil {
		cacheBatchData()
	}

	for _, r := range batchRenames {
		s = r.ReplaceAllString(s)
	}
	return s
}

// idRegexes strip datestamps and issue database IDs from workflow paths:
// matches are replaced with the first group
var idRegexes = []replacer{
	{regexp.MustCompile(`(..........-..........)-[0-9]+`), "$1"},
	{regexp.MustCompile(`(notouchie-..........)-[0-9]+`), "$1"},
	{regexp.MustCompile(`(split-..........)-[0-9]+`), "$1"},
}

// stripIdentifiers removes values that aren't important to the report and
// change from one run to the next, such as database IDs in workflow paths
func stripIdentifiers(s string) string {
	s = strings.ReplaceAll(s, ".wip-", "XXWIPXX")
	for _, r := range idRegexes {
		s = r.ReplaceAllString(s)
	}
	return renameBatches(s)
}

// xmlRegexes scrub XML files
var xmlRegexes = []replacer{
	{regexp.MustCompile(`<softwareVersion>.*</softwareVersion>`), `<softwareVersion>XYZZY</softwareVersion>`},
	{regexp.MustCompile(`<fileName>.*</fileName>`), `<fileName>XYZZY</fileName>`},
	{regexp.MustCompile(`\bID="TB\.[^"]*"`), `ID="XYZZY"`},
	{regexp.MustCompile(`<metsHdr CREATEDATE="....-..-..T..:..:..">`), `<metsHdr CREATEDATE="2006-01-02T15:04:05">`},
}

// actionsRegexes remove datestamps and job IDs from actions.txt
var actionsRegexes = []replacer{
	{regexp.MustCompile(`on .* [0-9]+, 20[0-9][0-9] at .*:`), `on DAY at TIME:`},
	{regexp.MustCompile(`Job [0-9]+ `), `Job N`},
}

// clean returns a copy of raw after running it through replacers
func clean(raw []byte, replacers []replacer) []byte {
	var out = make([]byte, len(raw))
	copy(out, raw)
	for _, r := range replacers {
		out = r.ReplaceAll(out)
	}
	return out
}

// cleanXML returns a copy of raw with dates and other identifiers scrubbed
func cleanXML(raw []byte) []byte {
	return []byte(renameBatches(string(clean(raw, xmlRegexes))))
}

// cleanActions returns a copy of raw with dates, times, and job IDs scrubbed
func cleanActions(raw []byte) []byte {
	return clean(raw, actionsRegexes)
}

// sqldump returns q's results as tab-separated values, the way "mysql -Ne"
// writes them when its output isn't a terminal
func sqldump(q string) string {
	var op = dbi.DB.Operation()
	var rows = op.Query(q)

	var cols = rows.Columns()
	if op.Err() != nil {
		logger.Fatalf("Unable to read columns: %s", op.Err())
	}

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
		logger.Fatalf("Error dumping sql with query %q: %s", q, op.Err())
	}

	return strings.Join(tsv, "\n") + "\n"
}

// reportPath is a file in the workflow tree, with its path as the report
// shows it and the report file it's written to (if it gets one)
type reportPath struct {
	absPath    string
	relPath    string
	outputPath string
}

// newReportPath builds a reportPath for pth, a path under the news root,
// scrubbed of identifiers. Paths are shown as they were under the old
// test/fakemount-based scripts ("./fakemount/..."), so reports from before
// the harness can still be compared with reports from after.
func newReportPath(reportDir, pth string) *reportPath {
	var p = &reportPath{absPath: pth}
	var tmp = "fakemount" + strings.TrimPrefix(pth, newsRoot)
	tmp = stripIdentifiers(tmp)
	p.relPath = "./" + tmp

	tmp = strings.ReplaceAll(tmp, "/", "__")
	tmp = strings.Replace(tmp, "fakemount__", "", 1)
	p.outputPath = filepath.Join(reportDir, tmp)

	return p
}

// Execute implements flags.Commander
func (c *reportCmd) Execute([]string) error {
	var err = connectDB()
	if err != nil {
		return err
	}

	var repdir string
	repdir, err = filepath.Abs(filepath.Join(c.Dir, c.Name+"-report"))
	if err != nil {
		return fmt.Errorf("finding report directory: %w", err)
	}
	if fileutil.Exists(repdir) {
		return fmt.Errorf("report directory %q already exists: remove or rename it first", repdir)
	}

	// Find all files in the workflow tree and sort them
	var filelist []string
	err = filepath.WalkDir(newsRoot, func(pth string, _ fs.DirEntry, err error) error {
		filelist = append(filelist, pth)
		return err
	})
	if err != nil {
		return fmt.Errorf("reading %q: %w", newsRoot, err)
	}
	sort.Strings(filelist)

	var allPaths = make([]*reportPath, len(filelist))
	for i, pth := range filelist {
		allPaths[i] = newReportPath(repdir, pth)
	}

	// The file manifest
	var manifestEntries []string
	for _, p := range allPaths {
		manifestEntries = append(manifestEntries, p.relPath)
	}
	var files = map[string][]byte{
		filepath.Join(repdir, "raw-files.txt"): []byte(strings.Join(manifestEntries, "\n") + "\n"),
	}

	// XMLs, with dates and IDs scrubbed, to be sure our ALTO conversion isn't
	// busted, and actions.txt files, scrubbed
	var tiffsums []string
	for _, p := range allPaths {
		var ext = filepath.Ext(p.absPath)
		switch {
		case ext == ".xml":
			var data, err = os.ReadFile(p.absPath)
			if err != nil {
				return err
			}
			files[p.outputPath] = cleanXML(data)

		case filepath.Base(p.absPath) == "actions.txt":
			var data, err = os.ReadFile(p.absPath)
			if err != nil {
				return err
			}
			files[p.outputPath] = cleanActions(data)

		case ext == ".tif" || ext == ".tiff":
			var sum, err = hasher.NewMD5().FileSum(p.absPath)
			if err != nil {
				return fmt.Errorf("checksumming %q: %w", p.absPath, err)
			}
			tiffsums = append(tiffsums, sum+"  "+p.relPath)
		}
	}
	files[filepath.Join(repdir, "tiffsums.txt")] = []byte(strings.Join(tiffsums, "\n") + "\n")

	// Critical info from the database, without churn like timestamps or
	// fields based on database IDs. This won't cover everything, but it should
	// be enough to show an end-to-end test isn't totally hosed.
	files[filepath.Join(repdir, "dump-batches.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT marc_org_code, name, status, location
		FROM batches
		ORDER BY marc_org_code, name, status
	`)))
	files[filepath.Join(repdir, "dump-actions.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT a.object_type, a.action_type, u.login, a.message
		FROM actions a
		LEFT OUTER JOIN users u ON (a.user_id = u.id)
		ORDER BY a.object_type, a.action_type, u.login, a.message
	`)))
	files[filepath.Join(repdir, "dump-issues.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT
			marc_org_code, date, date_as_labeled, volume, issue, edition,
			edition_label, page_labels_csv, is_from_scanner, workflow_step,
			location, ignored
		FROM issues
		ORDER BY lccn, date, edition
	`)))
	files[filepath.Join(repdir, "dump-jobs.sql")] = []byte(stripIdentifiers(sqldump(`
		SELECT p.name, p.description, p.object_type, j.job_type, j.status, j.object_type, j.extra_data
		FROM jobs j
		JOIN pipelines p ON (j.pipeline_id = p.id)
		ORDER BY p.name, p.description, p.object_type, j.job_type, j.status, j.object_type, j.extra_data
	`)))

	err = os.MkdirAll(repdir, 0755)
	if err != nil {
		return err
	}
	for pth, data := range files {
		err = os.WriteFile(pth, data, 0644)
		if err != nil {
			return err
		}
	}

	logger.Infof("Wrote report to %q", repdir)
	return nil
}
