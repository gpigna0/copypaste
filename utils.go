package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/argon2"
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

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Encode salt and hash
	encSalt := base64.RawStdEncoding.EncodeToString(salt)
	encHash := base64.RawStdEncoding.EncodeToString(hash)
	encodedPw := fmt.Sprintf("%s$%s", encSalt, encHash)

	return encodedPw, nil
}

func hashCompare(password string, hash string) (bool, error) {
	hashParts := strings.Split(hash, "$")
	baseHash, err := base64.RawStdEncoding.DecodeString(hashParts[1])
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(hashParts[0])
	if err != nil {
		return false, err
	}
	newHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	if subtle.ConstantTimeCompare(baseHash, newHash) == 1 {
		return true, nil
	}
	return false, nil
}
