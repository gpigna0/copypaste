package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/migrations.sql
var mig string

type Env struct {
	db          *pgxpool.Pool
	dataManager dbData
	clipBroker  EventBroker
	fileBroker  EventBroker
}

func NewEnv() (*Env, error) {
	addr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)
	pool, err := pgxpool.New(context.Background(), addr)
	if err != nil {
		return nil, err
	}

	if _, err := pool.Exec(context.Background(), mig); err != nil {
		return nil, err
	}

	clipbrk := NewEventBroker()
	clipbrk.Init()
	filebrk := NewEventBroker()
	filebrk.Init()

	return &Env{pool, defaultDbData{}, clipbrk, filebrk}, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	env, err := NewEnv()
	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}
	defer env.db.Close()

	http.HandleFunc("/{$}", handlerWrapper(env.mainPage))
	http.HandleFunc("/logout", handlerWrapper(logout))

	http.HandleFunc("GET /login", handlerWrapper(getLogin))
	http.HandleFunc("GET /register", handlerWrapper(getRegister))
	http.HandleFunc("GET /clipboard", handlerWrapper(env.getClips))
	http.HandleFunc("GET /clipboard/new", handlerWrapper(env.newClip))
	http.HandleFunc("GET /file", handlerWrapper(env.getFiles))
	http.HandleFunc("GET /file/download/{fileId}", handlerWrapper(env.sendFile))
	http.HandleFunc("GET /user", handlerWrapper(getUser))

	http.HandleFunc("POST /login", handlerWrapper(env.postLogin))
	http.HandleFunc("POST /register", handlerWrapper(env.postRegister))
	http.HandleFunc("POST /clipboard/new", handlerWrapper(env.postClip))
	http.HandleFunc("POST /file/new", handlerWrapper(env.postFile))

	http.HandleFunc("DELETE /clipboard", handlerWrapper(env.deleteClip))
	http.HandleFunc("DELETE /clipboard/all", handlerWrapper(env.deleteAllClips))
	http.HandleFunc("DELETE /file", handlerWrapper(env.deleteFile))

	http.HandleFunc("/clipboard/update", handlerWrapper(env.clipUpdate))
	http.HandleFunc("/file/update", handlerWrapper(env.fileUpdate))

	static := http.FileServer(http.Dir("./static"))
	http.Handle("/", static)

	if err := http.ListenAndServe(":2000", nil); err != nil {
		log.Printf("err: %v\n", err)
		return
	}
}
