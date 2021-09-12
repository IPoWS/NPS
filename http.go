package main

import (
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

func getIPPortStr(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func methodIs(m string, resp http.ResponseWriter, req *http.Request) bool {
	if req.Method != m {
		http.Error(resp, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func getFirst(key string, q *url.Values) string {
	keys, ok := (*q)[key]
	if ok {
		log.Debugln("[getfirst] get query", key, "=", keys[0], ".")
		return keys[0]
	}
	log.Debugln("[getfirst]", key, "has no query.")
	return ""
}
