package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const SESSIONCOOKIE = "Session-id"

var (
	defaultExpir = 36 * time.Hour
	sessions     = newSessionMap()
)

// Associate a cookie to a user and provide some utility functions
type session struct {
	user   string
	cookie http.Cookie
}

func (s *session) revitalize() {
	s.cookie.Expires = time.Now().Add(defaultExpir)
}

func (s *session) expired() bool {
	exp := s.cookie.Expires
	if exp.After(time.Now()) && !exp.IsZero() {
		return true
	} else {
		return false
	}
}

type sessionMap struct {
	m map[string]session
	sync.RWMutex
}

func newSessionMap() sessionMap {
	return sessionMap{make(map[string]session), sync.RWMutex{}}
}

func (m *sessionMap) session(r *http.Request) (s session, exists bool) {
	exists = false

	cookie, err := r.Cookie(SESSIONCOOKIE)
	if err != nil {
		log.Printf("ERR: %v\n", err)
		return
	}

	m.RLock()
	s, exists = m.m[cookie.Value]
	m.RUnlock()
	return
}

// A func to be called periodically to remove expired sessions
func cleanSessions() {
	sessions.Lock()
	defer sessions.Unlock()
	for cookie, sess := range sessions.m {
		if sess.expired() {
			delete(sessions.m, cookie)
		}
	}
	time.AfterFunc(time.Minute, func() { cleanSessions() }) // Call this function periodically
}

// Auth checker
func (env *Env) checkUser(r *http.Request) (*http.Cookie, error) {
	uname, pw, rem, err := loginInfo(r)
	if err != nil {
		return nil, err
	}

	// Get the user's password hash and compare it with the received pw.
	// If the user does not exist create a new user
	storedPassword, err := env.dataManager.userExists(env.db, uname)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		password, err := hashPassword(pw)
		if err != nil {
			return nil, err
		}
		if err := env.dataManager.insertUser(env.db, uname, password); err != nil {
			return nil, err
		}
	} else {
		hashCompare(pw, storedPassword)
	}

	// Create the cookie
	cookie, err := makeSession(uname, rem)
	if err != nil {
		return nil, err
	}

	// Create a file directory for the user
	if err := os.Mkdir("/filedir/"+uname, 0664); err != nil {
		if errors.Is(err, os.ErrExist) {
			log.Printf("err: %v\n", err)
		} else {
			return nil, err
		}
	}

	return cookie, nil
}

func loginInfo(r *http.Request) (string, string, string, error) {
	if err := r.ParseForm(); err != nil {
		log.Printf("err: %v\n", err)
		return "", "", "", err // bad request
	}

	// Get username and password from the form
	form := r.PostForm
	uname := form.Get("username")
	pw := form.Get("password")
	rem := form.Get("remember")
	if uname == "" || pw == "" {
		return "", "", "", errors.New("invalid username or password") // bad request
	}

	return uname, pw, rem, nil
}

func makeSession(user, remember string) (*http.Cookie, error) {
	exp := time.Now().Add(defaultExpir)
	if remember == "on" {
		exp = time.Now().AddDate(50, 0, 0)
	}

	cookieBytes := make([]byte, 20)
	if _, err := rand.Read(cookieBytes); err != nil {
		log.Printf("err: %v\n", err)
		return nil, err
	}
	cookieVal := base64.URLEncoding.EncodeToString(cookieBytes[:20])
	cookie := http.Cookie{
		Name:     SESSIONCOOKIE,
		Value:    cookieVal,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
	}

	sessions.Lock()
	sessions.m[cookie.Value] = session{user, cookie}
	sessions.Unlock()
	return &cookie, nil
}
