package mysql

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Open abre conexão usando a env MYSQL_DSN (ex.: user:pass@tcp(127.0.0.1:3306)/db?parseTime=true)
func Open() (*sql.DB, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Println("MYSQL_DSN vazio — defina a string de conexão")
	}
	return sql.Open("mysql", dsn)
}

func Ping(db *sql.DB) error { return db.Ping() }
