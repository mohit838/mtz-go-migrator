package database

import (
	"database/sql"
	"net/url"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func ConnectDB(dbConn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", withUTCTimeZone(dbConn))

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("SET TIME ZONE 'UTC'"); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func withUTCTimeZone(dbConn string) string {
	parsed, err := url.Parse(dbConn)
	if err != nil || parsed.Scheme == "" {
		return dbConn
	}

	query := parsed.Query()
	if query.Get("timezone") == "" && query.Get("TimeZone") == "" {
		query.Set("timezone", "UTC")
	}
	parsed.RawQuery = query.Encode()

	return parsed.String()
}
