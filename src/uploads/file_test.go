package uploads

import (
	"strconv"
	"strings"
	"testing"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/pdf"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

var dpis []*pdf.Image

// reset clears the DPI list
func reset() {
	dpis = nil
	_dpifunc = testDPIFunc
}

// add puts the given floats into the dpi list
func add(x, y, w, h int) {
	dpis = append(dpis, &pdf.Image{
		XPPI:   strconv.Itoa(x),
		YPPI:   strconv.Itoa(y),
		Width:  strconv.Itoa(w),
		Height: strconv.Itoa(h),
	})
}

// testDPIFunc overrides the dpi function to return the list of DPIs generated above
func testDPIFunc(loc string) ([]*pdf.Image, error) {
	return dpis, nil
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
	add(85, 85, 100, 100)
	add(100, 100, 100, 100)
	add(114, 114, 100, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if f.Errors.Len() != 0 {
		var elist []string
		for _, err := range f.Errors.All() {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected no errors.  Got: %q", strings.Join(elist, ","))
	}
}

// TestValidateDPIGoodSmallImages verifies that the image is still considered
// okay when a very tiny image is too small
func TestValidateDPIGoodSmallImages(t *testing.T) {
	reset()
	add(1, 1, 1, 1)
	add(100, 100, 100, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if f.Errors.Len() != 0 {
		var elist []string
		for _, err := range f.Errors.All() {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected no errors.  Got: %q", strings.Join(elist, ","))
	}
}

// TestValidateDPIBadSmallImages verifies we have failures if there are any
// low-res images that aren't trivially small
func TestValidateDPIBadSmallImages(t *testing.T) {
	reset()
	add(1, 1, 50, 50)
	add(100, 100, 100, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if f.Errors.Len() != 1 {
		var elist []string
		for _, err := range f.Errors.All() {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected 1 error.  Got: %q", strings.Join(elist, ","))
	}
}

// TestValidateDPIBadLargeImage verifies we fail on a single too-large image
func TestValidateDPIBadLargeImage(t *testing.T) {
	reset()
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(100, 100, 100, 100)
	add(116, 100, 100, 100)

	var f = fakeFile()
	f.ValidateDPI(100)
	if f.Errors.Len() != 1 {
		var elist []string
		for _, err := range f.Errors.All() {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected 1 error.  Got: %q", strings.Join(elist, ","))
	}
}

func TestValidateDPIBadNoImages(t *testing.T) {
	reset()
	var f = fakeFile()
	f.Name = "fake.pdf"
	f.ValidateDPI(100)
	if f.Errors.Len() != 1 {
		var elist []string
		for _, err := range f.Errors.All() {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected an error.  Got: %q", strings.Join(elist, ","))
	}
}
func TestValidateDPINonPDF(t *testing.T) {
	reset()
	add(0, 0, 100, 100)
	var f = fakeFile()
	f.Name = "fake.tiff"
	f.ValidateDPI(100)
	if f.Errors.Len() != 0 {
		var elist []string
		for _, err := range f.Errors.All() {
			elist = append(elist, err.Message())
		}
		t.Errorf("Expected no errors.  Got: %q", strings.Join(elist, ","))
	}
}
