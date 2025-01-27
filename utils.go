package main

import (
	"html/template"
	"log"
	"net/http"
)

type HTMLWriter struct {
	Writer http.ResponseWriter
	Status int
	HTMX   bool
}

func (w *HTMLWriter) WriteHeader() {
	w.Writer.WriteHeader(w.Status)
}

// Return true or false based on r.Header.Get("Hx-Request")
func isHTMX(r *http.Request) bool {
	if r.Header.Get("HX-Request") == "true" {
		return true
	} else {
		return false
	}
}

// Create and send an html template
func sendTemplate(w HTMLWriter, obj any, tname string, tmplPath ...string) {
	if !w.HTMX {
		index := []string{"./html/index.html"}
		tname = "index"
		tmplPath = append(index, tmplPath...)
	}
	if tname == "" {
		tname = "index"
	}

	tmpl := template.Must(template.ParseFiles(tmplPath...))
	if err := tmpl.ExecuteTemplate(w.Writer, tname, obj); err != nil {
		log.Printf("err: %v\n", err)
	}
}
