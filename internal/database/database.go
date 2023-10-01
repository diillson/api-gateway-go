package database

import (
	"database/sql"
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/config" // Certifique-se de que este import está correto
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
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
        description TEXT,
        isActive BOOLEAN,
        callCount INTEGER,
        totalResponse INTEGER,
        PRIMARY KEY (path)
    )`

	_, err := db.Conn.Exec(query)
	return err
}

func (db *Database) GetRoutes() ([]*config.Route, error) {
	rows, err := db.Conn.Query("SELECT path, serviceURL, methods, headers, description, isActive, callCount, totalResponse FROM routes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*config.Route
	for rows.Next() {
		var r config.Route
		var methods, headers string
		if err := rows.Scan(&r.Path, &r.ServiceURL, &methods, &headers, &r.Description, &r.IsActive, &r.CallCount, &r.TotalResponse); err != nil {
			return nil, err
		}
		r.Methods = strings.Split(methods, ",")
		r.Headers = strings.Split(headers, ",")
		routes = append(routes, &r)
	}

	return routes, nil
}

func (db *Database) AddRoute(route *config.Route) error {
	methods, _ := json.Marshal(route.Methods)
	headers, _ := json.Marshal(route.Headers)

	_, err := db.Conn.Exec(
		"INSERT INTO routes (path, serviceURL, methods, headers, description, isActive, callCount, totalResponse) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		route.Path, route.ServiceURL, methods, headers, route.Description, route.IsActive, route.CallCount, route.TotalResponse,
	)
	return err
}

func (db *Database) UpdateRoute(route *config.Route) error {
	methods, _ := json.Marshal(route.Methods)
	headers, _ := json.Marshal(route.Headers)

	_, err := db.Conn.Exec(
		"UPDATE routes SET serviceURL = ?, methods = ?, headers = ?, description = ?, isActive = ?, callCount = ?, totalResponse = ? WHERE path = ?",
		route.ServiceURL, methods, headers, route.Description, route.IsActive, route.CallCount, route.TotalResponse, route.Path,
	)
	return err
}

func (db *Database) DeleteRoute(path string) error {
	_, err := db.Conn.Exec("DELETE FROM routes WHERE path = ?", path)
	return err
}
