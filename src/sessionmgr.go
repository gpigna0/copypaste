package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

const SESSIONCOOKIE = "Session-id"

var (
	defaultExpir = 36 * time.Hour
	sessions     = newSessionMap()
)

type ErrWrongPassword struct {
	message string
}

func (e *ErrWrongPassword) Error() string {
	return e.message
}

// Associate a cookie to a user and provide some utility functions
type session struct {
	user      user
	clipEvtCh chan int8 // there is no need to send actual data through the channel
	cookie    http.Cookie
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

func newSessionMap() *sessionMap {
	newMap := sessionMap{make(map[string]session), sync.RWMutex{}}
	newMap.cleanRoutine()
	return &newMap
}

func (m *sessionMap) session(r *http.Request) (s session, exists bool) {
	exists = false

	cookie, err := r.Cookie(SESSIONCOOKIE)
	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}

	m.RLock()
	s, exists = m.m[cookie.Value]
	m.RUnlock()
	return
}

func (m *sessionMap) cleanRoutine() {
	time.AfterFunc(6*time.Hour, func() {
		log.Println("log: Removing expired sessions")
		for k, session := range m.m {
			if session.expired() {
				close(m.m[k].clipEvtCh)
				m.Lock()
				delete(m.m, k)
				m.Unlock()
			}
		}
	})
}

// checkUser is the authentication function for already registered users.
// If the user isn't found, *pgx.ErrNoRows will be returned.
// If the password is incorrect *ErrWrongPassword will be returned.
func (env *Env) checkUser(r *http.Request) (*http.Cookie, error) {
	uname, pw, rem, err := loginInfo(r)
	if err != nil {
		return nil, err
	}

	// Get the user's password hash and compare it with the received pw.
	user, err := env.dataManager.userExists(env.db, uname)
	if err != nil {
		return nil, err
	} else {
		ok, err := hashCompare(pw, user.Password)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &ErrWrongPassword{"user was found, but password is incorrect"}
		}
	}

	// Create the cookie
	cookie, err := makeSession(user, rem)
	if err != nil {
		return nil, err
	}

	// Check if there is a file directory for the user.
	// If there is an error try to make a new one.
	pth := path.Join("./filedir/", user.Id.String())
	if _, err := os.Lstat(pth); err != nil {
		log.Printf("err: %v --- trying to create a new directory\n", err)
		if err := os.Mkdir(pth, 0664); err != nil {
			if errors.Is(err, os.ErrExist) {
				log.Printf("err: %v\n", err)
			} else {
				return nil, err
			}
		}
	}

	return cookie, nil
}

// registerUser creates a new user and make a directory to store files.
func (env *Env) registerUser(r *http.Request) (*http.Cookie, error) {
	username, pw, rem, err := loginInfo(r)
	if err != nil {
		return nil, err
	}

	if err := env.dataManager.insertUser(env.db, username, pw); err != nil {
		return nil, err
	}
	user, err := env.dataManager.userExists(env.db, username)
	if err != nil {
		return nil, err
	}

	// Create the cookie
	cookie, err := makeSession(user, rem)
	if err != nil {
		return nil, err
	}

	// Create a file directory for the user
	pth := path.Join("./filedir/", user.Id.String())
	if err := os.Mkdir(pth, 0664); err != nil {
		return nil, err
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

func makeSession(user user, remember string) (*http.Cookie, error) {
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
	sessions.m[cookie.Value] = session{user, make(chan int8), cookie}
	sessions.Unlock()
	return &cookie, nil
}
