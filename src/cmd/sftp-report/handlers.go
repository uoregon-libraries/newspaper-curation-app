package main

import (
	"log"
	"net/http"
	"presenter"
	"sftp"
	"webutil"

	"github.com/gorilla/mux"
)

// Response generates a Responder with basic data all pages will need: request,
// response writer, and user
func Response(w http.ResponseWriter, req *http.Request) *Responder {
	var u = &User{req.Header.Get("X-Remote-User")}
	return &Responder{Writer: w, Request: req, Vars: &PageVars{User: u, Data: make(GenericVars)}}
}

// Middleware function to send back no-cache header
func nocache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var u = r.Header.Get("X-Remote-User")
		if u != "" {
			log.Printf("Request: [%s] %s", u, r.URL)
		} else {
			log.Printf("Request: [nil] %s", r.URL)
		}
		next.ServeHTTP(w, r)
	})
}

// HomeHandler spits out the publisher list
func HomeHandler(w http.ResponseWriter, req *http.Request) {
	var r = Response(w, req)
	r.Vars.Title = "SFTP Publisher List"
	var pubList, err = sftp.BuildPublishers(SFTPPath)
	if err != nil {
		log.Printf("ERROR: Couldn't load publishers in %s: %s", SFTPPath, err)
		http.Error(w, "Unable to load publisher list!", 500)
		return
	}

	r.Vars.Data["Publishers"] = presenter.PublisherList(pubList)
	r.Render("home")
}

// PublisherHandler prints a list of issues for a given publisher
func PublisherHandler(w http.ResponseWriter, req *http.Request) {
	var r = Response(w, req)

	var pubList, err = sftp.BuildPublishers(SFTPPath)
	if err != nil {
		log.Printf("ERROR: Couldn't load publishers in %s: %s", SFTPPath, err)
		http.Error(w, "Unable to load publisher list!", 500)
		return
	}

	var pubName = mux.Vars(req)["publisher"]
	var publisher *presenter.Publisher
	for _, p := range pubList {
		if p.Name == pubName {
			publisher = presenter.DecoratePublisher(p)
		}
	}

	if publisher == nil {
		http.Redirect(w, req, webutil.HomePath(), http.StatusFound)
		return
	}

	r.Vars.Data["Publisher"] = publisher
	r.Vars.Title = "SFTP Issues for " + publisher.Name
	r.Render("publisher")
}
