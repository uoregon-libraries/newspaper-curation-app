package main

import (
	"log"
	"net/http"
	"sftp"
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
	r.Vars.Title = "Publisher List"
	var pubList, err = sftp.BuildPublishers(SFTPPath)
	if err != nil {
		log.Printf("ERROR: Couldn't load publishers in %s: %s", SFTPPath, err)
		http.Error(w, "Unable to load publisher list!", 500)
		return
	}

	r.Vars.Data["Publishers"] = pubList
	r.Render("home")
}
