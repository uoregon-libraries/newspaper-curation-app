package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const errInvalidDate = Error("invalid date")

// uploadedFile ties a bit of file metadata to an uploaded file
type uploadedFile struct {
	Name string
	Size int64
	path string
	sum  []byte
}

// uploadForm holds onto all validated data the form has already handled.  Form
// parameters will overwrite anything previously validated to account for going
// back/forward on the form.
type uploadForm struct {
	sync.Mutex
	Owner        string
	Date         time.Time
	UID          string
	lastAccessed time.Time
	Files        []*uploadedFile
}

func (f *uploadForm) access() {
	f.Lock()
	f.lastAccessed = time.Now()
	f.Unlock()
}

func (f *uploadForm) String() string {
	return fmt.Sprintf(`&uploadForm{"Owner": %q, "Date": %q, "UID": %q, "lastAccessed": %q}`,
		f.Owner, f.Date.Format("2006-01-02"), f.UID, f.lastAccessed.String())
}

var forml sync.Mutex
var forms = make(map[string]*uploadForm)

func registerForm(owner string) *uploadForm {
	var f = &uploadForm{Owner: owner, lastAccessed: time.Now(), UID: genid(), Files: make([]*uploadedFile, 0)}
	forml.Lock()
	forms[f.UID] = f
	forml.Unlock()

	l.Infof("Registering new form %s", f)
	return f
}

func findForm(uid string) *uploadForm {
	forml.Lock()
	var f = forms[uid]
	forml.Unlock()

	if f != nil {
		f.access()
	}
	return f
}

func cleanForms() {
	for {
		purgeOldForms()
		time.Sleep(time.Hour * 6)
	}
}

func purgeOldForms() {
	var expired = time.Now().Add(-48 * time.Hour)
	l.Debugf("Purging old forms from before %s", expired)
	forml.Lock()
	var keysToPurge []string
	for key, f := range forms {
		l.Debugf("Looking at form %s", f)
		if f.lastAccessed.Before(expired) {
			l.Debugf("Will purge")
			keysToPurge = append(keysToPurge, key)
		} else {
			l.Debugf("Will *not* purge")
		}
	}

	for _, key := range keysToPurge {
		forms[key].destroy()
		delete(forms, key)
	}
	forml.Unlock()
}

func (f *uploadForm) parseMetadata(r *http.Request) error {
	return f.parseDate(r.FormValue("date"))
}

func (f *uploadForm) parseDate(rawDate string) error {
	f.Lock()
	defer f.Unlock()

	if rawDate == "" {
		return nil
	}

	var date, err = time.Parse("2006-01-02", rawDate)
	if err != nil {
		date, err = time.Parse("01-02-2006", rawDate)
	}

	if err != nil {
		return errInvalidDate
	}

	f.Date = date

	return nil
}

func (f *uploadForm) destroy() {
	var err error
	for _, file := range f.Files {
		err = os.Remove(file.path)
		if err != nil {
			l.Errorf("Unable to remove file %q (form %#v): %s", file.path, f, err)
		}
	}
}
