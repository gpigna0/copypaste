package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Env struct {
	db          *pgxpool.Pool
	dataManager dbData
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

	initDb := `CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  username   VARCHAR(25) PRIMARY KEY,
  password   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS clipboard (
  id        SMALLSERIAL PRIMARY KEY,
  clip_text TEXT NOT NULL,
  username  VARCHAR(25) NOT NULL,

  CONSTRAINT fk_users
    FOREIGN KEY (username) REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);`

	if _, err := pool.Exec(context.Background(), initDb); err != nil {
		log.Printf("err: %v\n", err)
		return nil, err
	}

	return &Env{pool, defaultDbData{}}, nil
}

func main() {
	env, err := NewEnv()
	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}
	defer env.db.Close()

	if err := os.Mkdir("filedir", 0664); err != nil {
		log.Printf("%v\n", err)
	}

	http.HandleFunc("/{$}", handlerWrapper(env.mainPage))

	http.HandleFunc("GET /clipboard", handlerWrapper(env.getClips))
	http.HandleFunc("GET /clipboard/new", handlerWrapper(env.newClip))
	// TODO: http.HandleFunc("GET /file", handlerWrapper(env.getFiles))

	http.HandleFunc("POST /login", handlerWrapper(env.postLogin))
	http.HandleFunc("POST /clipboard/new", handlerWrapper(env.postClip))
	// TODO: http.HandleFunc("POST /file/new", handlerWrapper(env.postFile))

	http.HandleFunc("DELETE /clipboard", handlerWrapper(env.deleteClip))
	http.HandleFunc("DELETE /clipboard/all", handlerWrapper(env.deleteAllClips))

	static := http.FileServer(http.Dir("./static"))
	http.Handle("/", static)
	// TODO: http.HandleFunc("/files/", handlerWrapper(env.serveFiles))

	if err := http.ListenAndServe(":2000", nil); err != nil {
		log.Printf("err: %v\n", err)
		return
	}
}
