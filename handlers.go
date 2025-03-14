package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// fileValidator is a regex that matches only file ids
var fileValidator = regexp.MustCompile(`^\d+$`)

// ALL //

func (env *Env) mainPage(w HTMLWriter, r *http.Request, _ session) {
	w.Writer.Header().Set("HX-Trigger", "Clipboard-Load")
	sendTemplate(w, "", "", "./html/index.html")
}

// GET //

func (env *Env) getClips(w HTMLWriter, r *http.Request, s session) {
	clips, err := env.dataManager.allClips(env.db)
	if err != nil {
		log.Printf("err: %v\n", err)
		clips = make([]clipboard, 0)
	}

	obj := map[string]any{
		"Username": s.user,
		"Clip":     clips,
	}
	sendTemplate(w, obj, "cliplist", "./html/cliplist.html")
}

func (env *Env) newClip(w HTMLWriter, r *http.Request, _ session) {
	sendTemplate(w, "", "newclip", "./html/newclip.html")
}

func (env *Env) getFiles(w HTMLWriter, _ *http.Request, s session) {
	files, err := env.dataManager.allFiles(env.db, s.user)
	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}

	obj := map[string]any{
		"Username": s.user,
		"Files":    files,
	}
	sendTemplate(w, obj, "files", "./html/files.html")
}

func (env *Env) sendFile(w HTMLWriter, r *http.Request, s session) {
	fileId := r.PathValue("fileId")
	if !fileValidator.MatchString(fileId) {
		w.Status = http.StatusBadRequest
		w.WriteHeader()
		w.Writer.Write([]byte{})
		return
	}
	path := fmt.Sprintf("/filedir/%s/%s", s.user, fileId)
	fileName, err := env.dataManager.fileName(env.db, s.user, fileId)
	if err != nil {
		log.Printf("err: %v\n", err)
		if errors.Is(err, pgx.ErrNoRows) {
			w.Status = http.StatusNotFound
		} else {
			w.Status = http.StatusInternalServerError
		}
		w.WriteHeader()
		w.Writer.Write([]byte{})
		return
	}

	w.Writer.Header().Set("Content-Disposition", "filename="+fileName)
	// w.WriteHeader()
	http.ServeFile(w.Writer, r, path)
}

// POST //

func (env *Env) postLogin(w HTMLWriter, r *http.Request, _ session) {
	cookie, err := env.checkUser(r)
	if err != nil {
		log.Printf("err: %v\n", err)
		sendTemplate(w, "Invalid Credentials, create new user?", "", "./html/login.html")
		return
	}

	http.SetCookie(w.Writer, cookie)
	w.WriteHeader()
	sendTemplate(w, "", "index", "./html/index.html")
}

func (env *Env) postClip(w HTMLWriter, r *http.Request, s session) {
	user := s.user

	if err := r.ParseForm(); err != nil {
		w.Status = http.StatusBadRequest
		w.WriteHeader()
		log.Printf("err: %v\n", err)
	} else if err := env.dataManager.insertClip(env.db, user, r.PostForm.Get("text")); err != nil {
		w.Status = http.StatusInternalServerError
		w.WriteHeader()
		log.Printf("err: %v\n", err)
	}

	env.clipBroker.Publish(s.user, 1)
}

func (env *Env) postFile(w HTMLWriter, r *http.Request, s session) {
	if err := r.ParseMultipartForm(int64(^uint64(0) >> 1)); err != nil {
		log.Printf("err: %v\n", err)
		return
	}
	fileMap := r.MultipartForm.File
	user := s.user

	for _, files := range fileMap {
		for _, f := range files {
			file, err := f.Open()
			if err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
			defer file.Close()

			fname, err := env.dataManager.insertFile(env.db, user, f.Filename)
			if err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
			local, err := os.Create(fmt.Sprintf("/filedir/%s/%s", user, fname))
			if err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
			defer local.Close()

			if _, err = io.Copy(local, file); err != nil {
				log.Printf("err: %v\n", err)
			}
		}
	}

	env.fileBroker.Publish(s.user, 1)
}

// DELETE //

func (env *Env) deleteClip(w HTMLWriter, r *http.Request, s session) {
	ids := r.URL.Query()["id"]

	if err := env.dataManager.deleteClips(env.db, s.user, ids...); err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	}

	env.clipBroker.Publish(s.user, 1)
	w.Status = http.StatusNoContent
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

func (env *Env) deleteAllClips(w HTMLWriter, r *http.Request, s session) {
	if err := env.dataManager.deleteAllClips(env.db, s.user); err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	}

	env.clipBroker.Publish(s.user, 1)
	w.Status = http.StatusNoContent
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

func (env *Env) deleteFile(w HTMLWriter, r *http.Request, s session) {
	ids := r.URL.Query()["id"]

	err := env.dataManager.deleteFiles(env.db, s.user, ids...)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	} else {
		for _, fname := range ids {
			if err := os.Remove(fmt.Sprintf("/filedir/%s/%s", s.user, fname)); err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
		}
	}

	env.fileBroker.Publish(s.user, 1)
	w.Status = http.StatusNoContent
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

// SSE //

func (env *Env) clipUpdate(w HTMLWriter, r *http.Request, s session) {
	writer := w.Writer // set the needed headers
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	env.clipBroker.Subscribe <- s
	done := r.Context().Done()

	rc := http.NewResponseController(writer)
	for {
		select {
		case <-done:
			env.clipBroker.Unsubscribe <- s
			return
		case _, open := <-s.clipEvtCh:
			if !open {
				return
			}
			if _, err := fmt.Fprintf(writer, "event: %s-update-clipboard\ndata:\n\n", s.user); err != nil {
				log.Printf("err: %v", err)
				continue
			}
			if err := rc.Flush(); err != nil {
				log.Printf("err: %v", err)
				continue
			}
			log.Println("ok")
		}
	}
}

func (env *Env) fileUpdate(w HTMLWriter, r *http.Request, s session) {
	writer := w.Writer // set the needed headers
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	env.fileBroker.Subscribe <- s
	done := r.Context().Done()

	rc := http.NewResponseController(writer)
	for {
		select {
		case <-done:
			env.fileBroker.Unsubscribe <- s
			return
		case _, open := <-s.clipEvtCh:
			if !open {
				return
			}
			if _, err := fmt.Fprintf(writer, "event: %s-update-file\ndata:\n\n", s.user); err != nil {
				log.Printf("err: %v", err)
				continue
			}
			if err := rc.Flush(); err != nil {
				log.Printf("err: %v", err)
				continue
			}
			log.Println("ok")
		}
	}
}
