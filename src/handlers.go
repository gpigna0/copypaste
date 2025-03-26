package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/jackc/pgx/v5"
)

// ALL //

func (env *Env) mainPage(w HTMLWriter, r *http.Request, _ session) {
	w.Writer.Header().Set("HX-Trigger", "Clipboard-Load")
	sendTemplate(w, "", "", "./html/index.html")
}

// GET //

func getLogin(w HTMLWriter, r *http.Request, _ session) {
	sendTemplate(w, "", "login", "./html/login.html")
}

func getRegister(w HTMLWriter, r *http.Request, _ session) {
	sendTemplate(w, "", "register", "./html/register.html")
}

func (env *Env) getClips(w HTMLWriter, r *http.Request, s session) {
	clips, err := env.dataManager.allClips(env.db)
	if err != nil {
		log.Printf("err: %v\n", err)
		clips = make([]clipboard, 0)
	}

	obj := map[string]any{
		"UserId": s.user.Id.String(),
		"Clip":   clips,
	}
	sendTemplate(w, obj, "cliplist", "./html/cliplist.html")
}

func (env *Env) newClip(w HTMLWriter, r *http.Request, _ session) {
	sendTemplate(w, "", "newclip", "./html/newclip.html")
}

func (env *Env) getFiles(w HTMLWriter, _ *http.Request, s session) {
	files, err := env.dataManager.allFiles(env.db, s.user.Username)
	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}

	obj := map[string]any{
		"UserId": s.user.Id.String(),
		"Files":  files,
	}
	sendTemplate(w, obj, "files", "./html/files.html")
}

func (env *Env) sendFile(w HTMLWriter, r *http.Request, s session) {
	fileId := r.PathValue("fileId")
	pth := path.Join("./filedir", s.user.Id.String(), fileId)

	fileName, err := env.dataManager.fileName(env.db, s.user.Username, fileId)
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
	http.ServeFile(w.Writer, r, pth)
}

// POST //

func (env *Env) postLogin(w HTMLWriter, r *http.Request, _ session) {
	cookie, err := env.checkUser(r)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			sendTemplate(w, "This user doesn't exists, create a new one?", "", "./html/login.html")
		} else if errors.Is(err, &ErrWrongPassword{}) {
			sendTemplate(w, "The password is not correct", "", "./html/login.html")
		} else {
			log.Printf("err: %v\n", err)
			w.Status = http.StatusInternalServerError
			w.WriteHeader()
		}
		return
	}

	http.SetCookie(w.Writer, cookie)
	w.WriteHeader()
	sendTemplate(w, "", "index", "./html/index.html")
}

func (env *Env) postRegister(w HTMLWriter, r *http.Request, _ session) {
	cookie, err := env.registerUser(r)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
		w.WriteHeader()
		return
	}

	http.SetCookie(w.Writer, cookie)
	w.WriteHeader()
	sendTemplate(w, "", "index", "./html/index.html")
}

func (env *Env) postClip(w HTMLWriter, r *http.Request, s session) {
	if err := r.ParseForm(); err != nil {
		w.Status = http.StatusBadRequest
		w.WriteHeader()
		log.Printf("err: %v\n", err)
	} else if err := env.dataManager.insertClip(env.db, s.user.Username, r.PostForm.Get("text")); err != nil {
		w.Status = http.StatusInternalServerError
		w.WriteHeader()
		log.Printf("err: %v\n", err)
	}

	env.clipBroker.Publish(s.user.Username, 1)
}

func (env *Env) postFile(w HTMLWriter, r *http.Request, s session) {
	if err := r.ParseMultipartForm(int64(^uint64(0) >> 1)); err != nil {
		log.Printf("err: %v\n", err)
		return
	}
	fileMap := r.MultipartForm.File

	for _, files := range fileMap {
		for _, f := range files {
			file, err := f.Open()
			if err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
			defer file.Close()

			fname, err := env.dataManager.insertFile(env.db, s.user.Username, f.Filename)
			if err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
			pth := path.Join("./filedir", s.user.Id.String(), fname)
			local, err := os.Create(pth)
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

	env.fileBroker.Publish(s.user.Username, 1)
}

// DELETE //

func (env *Env) deleteClip(w HTMLWriter, r *http.Request, s session) {
	ids := r.URL.Query()["id"]

	if err := env.dataManager.deleteClips(env.db, s.user.Username, ids...); err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	}

	env.clipBroker.Publish(s.user.Username, 1)
	w.Status = http.StatusNoContent
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

func (env *Env) deleteAllClips(w HTMLWriter, r *http.Request, s session) {
	if err := env.dataManager.deleteAllClips(env.db, s.user.Username); err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	}

	env.clipBroker.Publish(s.user.Username, 1)
	w.Status = http.StatusNoContent
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

func (env *Env) deleteFile(w HTMLWriter, r *http.Request, s session) {
	ids := r.URL.Query()["id"]

	err := env.dataManager.deleteFiles(env.db, s.user.Username, ids...)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	} else {
		for _, fname := range ids {
			pth := path.Join("./filedir", s.user.Id.String(), fname)
			if err := os.Remove(pth); err != nil {
				log.Printf("err: %v\n", err)
				continue
			}
		}
	}

	env.fileBroker.Publish(s.user.Username, 1)
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
			if _, err := fmt.Fprintf(writer, "event: %s-update-clipboard\ndata:\n\n", s.user.Id); err != nil {
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
			if _, err := fmt.Fprintf(writer, "event: %s-update-file\ndata:\n\n", s.user.Id); err != nil {
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
