// rewrite-mets reads a METS XML file, uses the data to generate unsaved db
// structures, then generates new METS XML.  The main purpose of this tool is
// to verify that the input roughly matches the output.

package main

import (
	"chronam"
	"cli"
	"config"
	"db"
	"derivatives/mets"
	"fmt"
	"mods"
	"os"
	"strconv"
	"strings"
	"time"
)

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

var issue = new(db.Issue)
var title = new(db.Title)
var createdAt time.Time
var conf *config.Config

// Command-line options
type _opts struct {
	cli.BaseOptions
	SourceXML string `short:"i" description:"path to existing METS XML file" required:"true"`
	DestXML   string `short:"o" description:"path to destination file" required:"true"`
}

var opts _opts

func main() {
	var c = cli.New(&opts)
	conf = c.GetConf()

	var sourceFile = opts.SourceXML
	var destFile = opts.DestXML

	var mxml, err = chronam.ParseMETSIssueXML(sourceFile)
	if err != nil {
		fail("Unable to parse %q: %s\n", sourceFile, err)
	}

	createdAt, err = time.Parse(mets.TimeFormat, mxml.Header.CreateDate)
	if err != nil {
		fail("Bad METS header time %q: %s", mxml.Header.CreateDate, err)
	}

	// Parse the issue metadata separately from the page metadata
	for _, dmd := range mxml.DMDSecs {
		if dmd.ID == "issueModsBib" {
			parseIssueData(dmd.Data)
		} else {
			parsePageData(dmd.Data)
		}
	}

	// Now we should have issue date, so we can split up the label to get the title
	var parts = strings.Split(mxml.Label, ", "+issue.Date)
	title.Title = parts[0]

	err = mets.New(conf.XMLTemplatePath, destFile, issue, title, createdAt).Transform()
	if err == nil {
		fmt.Println("Generated XML successfully")
		os.Exit(0)
	}
	fail("Unable to generate METS XML: %s", err)
}

func parseIssueData(data mods.Data) {
	title.Rights = data.Rights
	// Go through the "relatedItem" tags to pull out the LCCN and dive into the
	// issue metadata "detail" parts
	for _, item := range data.RelatedItems {
		if item.Type == "host" {
			for _, id := range item.IDs {
				if id.Type == "lccn" {
					title.LCCN = id.Label
					issue.LCCN = id.Label
				}
			}
			// Each "part" can have multiple details, each of which contain our issue
			// metadata: volume, issue, edition number, edition label
			for _, part := range item.Parts {
				for _, detail := range part.Details {
					switch detail.Type {
					case "volume":
						issue.Volume = detail.Number
					case "issue":
						issue.Issue = detail.Number
					case "edition":
						var err error
						issue.Edition, err = strconv.Atoi(detail.Number)
						if err != nil || issue.Edition == 0 {
							fail("Invalid value for issue edition: %q", detail.Number)
						}
						issue.EditionLabel = detail.Caption
					}
				}
			}
		}
	}

	// Origin info gives us the issue date and a possible date-as-labeled value
	for _, info := range data.OriginInfos {
		for _, date := range info.Dates {
			switch date.Qualifier {
			case "":
				if issue.Date != "" {
					fail("Too many dates found")
				}
				issue.Date = date.Date

			case "questionable":
				if issue.DateAsLabeled != "" {
					fail("Too many date with 'questionable' qualifier found")
				}
				issue.DateAsLabeled = date.Date

			default:
				fail("Unknown date qualifier: %q", date.Qualifier)
			}
		}
	}

	// If there weren't any "questionable" dates, then DateAsLabeled is the same
	// as the actual issue date
	if issue.DateAsLabeled == "" {
		issue.DateAsLabeled = issue.Date
	}
}

func parsePageData(data mods.Data) {
	// Iterate over all parts to get page and optionally page labels
	for pNum, part := range data.Parts {
		// "extent" must be present, and gives us the page number so we can sort properly
		var pageNumber int
		for eNum, extent := range part.Extents {
			if eNum > 0 {
				fail("Too many 'extent' elements in page data part %d", pNum)
			}
			var err error
			pageNumber, err = strconv.Atoi(extent.Start)
			if err != nil {
				fail("Invalid page number in page data part %d: %q", pNum, extent.Start)
			}
		}

		if pageNumber == 0 {
			fail("Missing page number in page data part %d", pNum)
		}

		var pageLabel string

		// "detail" may or may not be present; if so, its "number" is our page label
		for dNum, detail := range part.Details {
			if dNum > 0 {
				fail("Too many 'detail' elements in page data part %d", pNum)
			}
			pageLabel = detail.Number
		}

		if pageLabel == "" {
			pageLabel = "0"
		}

		for pageNumber > len(issue.PageLabels) {
			issue.PageLabels = append(issue.PageLabels, "")
		}
		issue.PageLabels[pageNumber-1] = pageLabel
	}
}
