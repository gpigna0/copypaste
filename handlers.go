package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// ALL //

func (env *Env) mainPage(w HTMLWriter, r *http.Request, _ session) {
	w.Writer.Header().Set("HX-Trigger", "Clipboard-Load")
	sendTemplate(w, "", "", "./html/index.html")
}

// GET //

func (env *Env) getClips(w HTMLWriter, r *http.Request, _ session) {
	clips, err := env.dataManager.allClips(env.db)
	if err != nil {
		log.Printf("ERR: %v\n", err)
		clips = make([]clipboard, 0)
	}

	sendTemplate(w, clips, "cliplist", "./html/cliplist.html")
}

func (env *Env) newClip(w HTMLWriter, r *http.Request, _ session) {
	sendTemplate(w, "", "newclip", "./html/newclip.html")
}

func (env *Env) getFiles(w HTMLWriter, _ *http.Request, s session) {
	ftree, err := os.ReadDir("./filedir/" + s.user)
	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}
	sendTemplate(w, ftree, "files", "./html/files.html")
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

	w.Writer.Header().Set("HX-Trigger", "Clipboard-Load")
	sendTemplate(w, "", "newclip", "./html/newclip.html")
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

			local, err := os.Create(fmt.Sprintf("./filedir/%s/%s", user, f.Filename))
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

	ftree, err := os.ReadDir("./filedir/" + user)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.Status = http.StatusInternalServerError
	}
	w.WriteHeader()
	sendTemplate(w, ftree, "files", "./html/files.html")
}

// DELETE //

func (env *Env) deleteClip(w HTMLWriter, r *http.Request, s session) {
	ids := r.URL.Query()["id"]

	if err := env.dataManager.deleteClips(env.db, s.user, ids...); err != nil {
		log.Printf("ERR: %v\n", err)
		w.Status = http.StatusInternalServerError
	}
	w.Writer.Header().Set("HX-Trigger", "Clipboard-Load")
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

func (env *Env) deleteAllClips(w HTMLWriter, r *http.Request, s session) {
	if err := env.dataManager.deleteAllClips(env.db, s.user); err != nil {
		log.Printf("ERR: %v\n", err)
		w.Status = http.StatusInternalServerError
	}
	w.Writer.Header().Set("HX-Trigger", "Clipboard-Load")
	w.WriteHeader()
	sendTemplate(w, "", "nil", "./html/index.html")
}

// SERVE //

func (env *Env) serveFiles(w HTMLWriter, r *http.Request, s session) {
	user := s.user

	files := http.StripPrefix("/files", http.FileServer(http.Dir("./filedir/"+user)))
	files.ServeHTTP(w.Writer, r)
}
