package main

import (
	"cmd/server/internal/responder"
	"logger"
	"net/http"
)

// nocache is a Middleware function to send back no-cache header
func nocache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var u = responder.GetUserLogin(w, r)
		var ip = responder.GetUserIP(r)
		if u != "" {
			logger.Info("Request: [%s] [%s] %s", u, ip, r.URL)
		} else {
			logger.Info("Request: [nil] [%s] %s", ip, r.URL)
		}
		next.ServeHTTP(w, r)
	})
}
