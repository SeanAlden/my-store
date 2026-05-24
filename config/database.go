package config

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {
	dsn := "root:@tcp(127.0.0.1:3306)/gycora_database?parseTime=true"
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Error connecting to database: ", err)
	}
}