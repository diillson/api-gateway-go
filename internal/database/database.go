package database

import (
	"database/sql"
	"github.com/diillson/api-gateway-go/internal/config"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	Conn *sql.DB
}

func NewDatabase() *Database {
	conn, err := sql.Open("sqlite3", "./routes.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	db := &Database{Conn: conn}

	if err := db.initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	return db
}

func (db *Database) initialize() error {
	query := `
	CREATE TABLE IF NOT EXISTS routes (
		path TEXT NOT NULL,
		serviceURL TEXT NOT NULL,
		methods TEXT NOT NULL,
		headers TEXT NOT NULL,
		PRIMARY KEY (path)
	)`

	_, err := db.Conn.Exec(query)
	return err
}

func (db *Database) GetRoutes() ([]*config.Route, error) {
	rows, err := db.Conn.Query("SELECT path, serviceURL, methods, headers FROM routes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*config.Route
	for rows.Next() {
		var r config.Route
		if err := rows.Scan(&r.Path, &r.ServiceURL, &r.Methods, &r.Headers); err != nil {
			return nil, err
		}
		routes = append(routes, &r)
	}

	return routes, nil
}

func (db *Database) AddRoute(route *config.Route) error {
	_, err := db.Conn.Exec(
		"INSERT INTO routes (path, serviceURL, methods, headers) VALUES (?, ?, ?, ?)",
		route.Path, route.ServiceURL, strings.Join(route.Methods, ","), strings.Join(route.Headers, ","),
	)
	return err
}

func (db *Database) UpdateRoute(route *config.Route) error {
	_, err := db.Conn.Exec(
		"UPDATE routes SET serviceURL = ?, methods = ?, headers = ? WHERE path = ?",
		route.ServiceURL, strings.Join(route.Methods, ","), strings.Join(route.Headers, ","), route.Path,
	)
	return err
}

func (db *Database) DeleteRoute(path string) error {
	_, err := db.Conn.Exec("DELETE FROM routes WHERE path = ?", path)
	return err
}
