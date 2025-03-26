package main

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type clipboard struct {
	Text     string `db:"clip_text"`
	Username string
	Id       uuid.UUID
}

type file struct {
	Filename string
	Username string
	Id       uuid.UUID
}

type user struct {
	Username string
	Password string
	Id       uuid.UUID
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
	deleteFiles(db *pgxpool.Pool, user string, ids ...string) error

	userExists(db *pgxpool.Pool, user string) (user, error)
	insertUser(db *pgxpool.Pool, user string, password string) error
}

type defaultDbData struct{}

func (defaultDbData) allClips(db *pgxpool.Pool) ([]clipboard, error) {
	rows, err := db.Query(context.Background(), "SELECT * FROM clipboard")
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
	id := uuid.New()
	query := "INSERT INTO clipboard (clip_text, username, id) VALUES ($1, $2, $3)"
	if _, err := db.Exec(context.Background(), query, text, user, id); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) deleteClips(db *pgxpool.Pool, user string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	idSet := strings.Join(ids, ",")
	query := "DELETE FROM clipboard WHERE username=$1 AND id IN ($2)"
	if _, err := db.Exec(context.Background(), query, user, idSet); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) deleteAllClips(db *pgxpool.Pool, user string) error {
	query := "DELETE FROM clipboard WHERE username=$1"
	if _, err := db.Exec(context.Background(), query, user); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) insertFile(db *pgxpool.Pool, user string, filename string) (string, error) {
	// INFO: The db simply stores the reference to a file, so there's no need to update when an existing name is inserted
	id := uuid.New()
	query := "INSERT INTO files (filename, username, id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING"
	if _, err := db.Exec(context.Background(), query, filename, user, id); err != nil {
		return "", err
	}

	s := ""
	query = "SELECT id FROM files WHERE username=$1 AND filename=$2"
	row := db.QueryRow(context.Background(), query, user, filename)
	if err := row.Scan(&s); err != nil {
		return "", err
	}
	return s, nil
}

func (defaultDbData) allFiles(db *pgxpool.Pool, user string) ([]file, error) {
	rows, err := db.Query(context.Background(), "SELECT * FROM files WHERE username=$1", user)
	if err != nil {
		return nil, err
	}

	files, err := pgx.CollectRows(rows, pgx.RowToStructByName[file])
	return files, err
}

func (defaultDbData) fileName(db *pgxpool.Pool, user string, id string) (string, error) {
	query := "SELECT (filename) FROM files WHERE username=$1 AND id=$2"
	row := db.QueryRow(context.Background(), query, user, id)
	var fname string
	if err := row.Scan(&fname); err != nil {
		return "", err
	}
	return fname, nil
}

// deleteFiles deletes file entries based on received ids
// and returns the set of filenames that need to be deleted from the system
func (defaultDbData) deleteFiles(db *pgxpool.Pool, username string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	idSet := strings.Join(ids, ",")

	query := "DELETE FROM files WHERE username=$1 AND id IN ($2)"
	if _, err := db.Exec(context.Background(), query, username, idSet); err != nil {
		return err
	}
	return nil
}

func (defaultDbData) userExists(db *pgxpool.Pool, username string) (user, error) {
	rows, err := db.Query(context.Background(), "SELECT * FROM users WHERE username=$1", username)
	if err != nil {
		return user{}, err
	}
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[user])
	if err != nil {
		return user{}, err
	}

	return u, nil
}

// insertUser first hashes the password and then creates a new user using the hashed password
func (defaultDbData) insertUser(db *pgxpool.Pool, username string, password string) error {
	id := uuid.New()
	pw, err := hashPassword(password)
	if err != nil {
		return err
	}

	query := "INSERT INTO users (id, username, password) VALUES ($1, $2, $3)"
	if _, err := db.Exec(context.Background(), query, id, username, pw); err != nil {
		return err
	}

	return nil
}
