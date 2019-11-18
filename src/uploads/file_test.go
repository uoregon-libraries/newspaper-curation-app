package uploads

import (
	"strings"
	"testing"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/pdf"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

var dpis []pdf.ImageDPI

// reset clears the DPI list
func reset() {
	dpis = nil
	_dpifunc = testDPIFunc
}

// add puts the given floats into the dpi list
func add(x, y float64) {
	dpis = append(dpis, pdf.ImageDPI{X: x, Y: y})
}

// testDPIFunc overrides the dpi function to return the list of DPIs generated above
func testDPIFunc(loc string) []pdf.ImageDPI {
	return dpis
}

// fakeFile just returns a File with enough data to make tests work.  The need
// for this much fake-data is a wonderful sign our data structures need work.
func fakeFile() *File {
	return &File{
		File: &schema.File{
			Issue:    &schema.Issue{},
			File:     &fileutil.File{Name: "fake.pdf"},
			Location: "/tmp/fake.pdf",
		},
	}

}

func TestValidateDPIGood(t *testing.T) {
	reset()
	add(85, 85)
	add(100, 100)
	add(114, 114)

	var f = fakeFile()
	f.ValidateDPI(100)
	if len(f.Errors) != 0 {
		var elist []string
		for _, err := range f.Errors {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected no errors.  Got: %q", strings.Join(elist, ","))
	}
}

// TestValidateDPIGoodSmallImages verifies that the image is still considered
// okay when half (or more) images are fine
func TestValidateDPIGoodSmallImages(t *testing.T) {
	reset()
	add(1, 1)
	add(100, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if len(f.Errors) != 0 {
		var elist []string
		for _, err := range f.Errors {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected no errors.  Got: %q", strings.Join(elist, ","))
	}
}

// TestValidateDPIBadSmallImages verifies we have failures if there are too many small images
func TestValidateDPIBadSmallImages(t *testing.T) {
	reset()
	add(1, 1)
	add(1, 1)
	add(100, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if len(f.Errors) != 1 {
		var elist []string
		for _, err := range f.Errors {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected 1 error.  Got: %q", strings.Join(elist, ","))
	}
}

// TestValidateDPIBadLargeImage verifies we fail on a single too-large image
func TestValidateDPIBadLargeImage(t *testing.T) {
	reset()
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(100, 100)
	add(116, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if len(f.Errors) != 1 {
		var elist []string
		for _, err := range f.Errors {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected 1 error.  Got: %q", strings.Join(elist, ","))
	}
}
