package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// uploadForm holds onto all validated data the form has already handled.  Form
// parameters will overwrite anything previously validated to account for going
// back/forward on the form.
type uploadForm struct {
	sync.Mutex
	Owner        string
	Date         time.Time
	UID          string
	lastAccessed time.Time
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
	var f = &uploadForm{Owner: owner, lastAccessed: time.Now(), UID: genid()}
	forml.Lock()
	forms[f.UID] = f
	forml.Unlock()

	logger.Infof("Registering new form %s", f)
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
	logger.Debugf("Purging old forms from before %s", expired)
	forml.Lock()
	var keysToPurge []string
	for key, f := range forms {
		logger.Debugf("Looking at form %s", f)
		if f.lastAccessed.Before(expired) {
			logger.Debugf("Will purge")
			keysToPurge = append(keysToPurge, key)
		} else {
			logger.Debugf("Will *not* purge")
		}
	}

	for _, key := range keysToPurge {
		delete(forms, key)
	}
	forml.Unlock()
}
