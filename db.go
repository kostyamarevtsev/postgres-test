package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Op string

const (
	Create Op = "create"
	Modify Op = "modify"
	Remove Op = "remove"
)

type postgres struct {
	db *sql.DB
}

func initDB() (*postgres, error) {

	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS log_table (
			id SERIAL PRIMARY KEY,
			path TEXT,
			op TEXT,
			data JSONB
		);
	`)
	if err != nil {
		return nil, err
	}

	return &postgres{db: db}, nil
}

func (r *postgres) insert(path string, op Op, data []diffmatchpatch.Diff) error {
	var filteredData []diffmatchpatch.Diff
	for _, d := range data {
		if d.Type != diffmatchpatch.DiffEqual {
			filteredData = append(filteredData, d)
		}
	}

	jsonData, err := json.Marshal(filteredData)
	if err != nil {
		return err
	}

	query := "INSERT INTO log_table (path, op, data) VALUES ($1, $2, $3::jsonb)"
	_, err = r.db.Exec(query, path, string(op), jsonData)
	if err != nil {
		return err
	}

	return nil
}
