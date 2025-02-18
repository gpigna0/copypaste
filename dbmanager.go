package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type clipboard struct {
	Text     string `db:"clip_text"`
	Username string
	Id       int
}

type file struct {
	Filename string
	Username string
	Id       int
}

type dbData interface {
	// TODO: put named arguments
	allClips(*pgxpool.Pool) ([]clipboard, error)
	insertClip(*pgxpool.Pool, string, string) error
	deleteClips(*pgxpool.Pool, string, ...string) error
	deleteAllClips(*pgxpool.Pool, string) error
	insertFile(db *pgxpool.Pool, user string, filename string) (string, error)
	allFiles(db *pgxpool.Pool, user string) ([]file, error)
	fileName(db *pgxpool.Pool, user string, id string) (string, error)
	deleteFiles(db *pgxpool.Pool, user string, ids ...string) ([]string, error)
	userExists(db *pgxpool.Pool, user string) (string, error)
	insertUser(db *pgxpool.Pool, user string, password string) error
}

type defaultDbData struct{}

func (defaultDbData) allClips(db *pgxpool.Pool) ([]clipboard, error) {
	rows, err := db.Query(context.Background(), "SELECT * FROM clipboard ORDER BY id DESC")
	if err != nil {
		return []clipboard{}, err
	}

	clips, err := pgx.CollectRows(rows, pgx.RowToStructByName[clipboard])
	if err != nil {
		return []clipboard{}, err
	}
	return clips, nil
}

func (defaultDbData) insertClip(db *pgxpool.Pool, user string, text string) error {
	query := fmt.Sprintf("INSERT INTO clipboard (clip_text, username) VALUES ('%s', '%s')", text, user)
	if _, err := db.Exec(context.Background(), query); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) deleteClips(db *pgxpool.Pool, user string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	for i, id := range ids {
		ids[i] = "id=" + id
	}
	idCond := strings.Join(ids, " AND ")
	query := fmt.Sprintf("DELETE FROM clipboard WHERE username = '%s' AND %s", user, idCond)
	if _, err := db.Exec(context.Background(), query); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) deleteAllClips(db *pgxpool.Pool, user string) error {
	query := fmt.Sprintf("DELETE FROM clipboard WHERE username = '%s'", user)

	if _, err := db.Exec(context.Background(), query); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) insertFile(db *pgxpool.Pool, user string, filename string) (string, error) {
	// INFO: The db simply stores the reference to a file, so there's no need to update when an existing name is inserted
	println("ok")
	query := fmt.Sprintf("INSERT INTO files (filename, username) VALUES ('%s', '%s') ON CONFLICT DO NOTHING", filename, user)
	if _, err := db.Exec(context.Background(), query); err != nil {
		return "", err
	}

	s := ""
	query = fmt.Sprintf("SELECT id FROM files WHERE username='%s' AND filename='%s'", user, filename)
	row := db.QueryRow(context.Background(), query)
	if err := row.Scan(&s); err != nil {
		return "", err
	}
	return s, nil
}

func (defaultDbData) allFiles(db *pgxpool.Pool, user string) ([]file, error) {
	rows, err := db.Query(context.Background(), "SELECT * FROM files WHERE username='"+user+"'")
	if err != nil {
		return nil, err
	}

	files, err := pgx.CollectRows(rows, pgx.RowToStructByName[file])
	return files, err
}

func (defaultDbData) fileName(db *pgxpool.Pool, user string, id string) (string, error) {
	query := fmt.Sprintf("SELECT (filename) FROM files WHERE username='%s' AND id=%s", user, id)
	row := db.QueryRow(context.Background(), query)
	var fname string
	if err := row.Scan(&fname); err != nil {
		return "", err
	}
	return fname, nil
}

// deleteFiles deletes file entries based on received ids
// and returns the set of filenames that need to be deleted from the system
func (defaultDbData) deleteFiles(db *pgxpool.Pool, user string, ids ...string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	idSet := strings.Join(ids, ",")

	query := fmt.Sprintf("SELECT (filename) FROM files WHERE username = '%s' AND id IN (%s)", user, idSet)
	rows, err := db.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	tbd, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	query = fmt.Sprintf("DELETE FROM files WHERE username = '%s' AND id IN (%s)", user, idSet)
	if _, err := db.Exec(context.Background(), query); err != nil {
		return nil, err
	}
	return tbd, nil
}

func (defaultDbData) userExists(db *pgxpool.Pool, uname string) (string, error) {
	res := db.QueryRow(context.Background(), "SELECT (password) FROM users WHERE username = '"+uname+"'")
	var pw string
	if err := res.Scan(&pw); err != nil {
		return "", err
	}

	return pw, nil
}

func (defaultDbData) insertUser(db *pgxpool.Pool, user string, password string) error {
	query := fmt.Sprintf("INSERT INTO users (username, password) VALUES ('%s', '%s')", user, password)
	if _, err := db.Exec(context.Background(), query); err != nil {
		return err
	}
	log.Printf("Created user %s\n", user)

	return nil
}
