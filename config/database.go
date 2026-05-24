package config

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {
	// load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env")
	}

	// ambil env
	dsn := os.Getenv("MYSQL_DSN")

	if dsn == "" {
		log.Fatal("MYSQL_DSN is empty")
	}

	var errDB error
	DB, errDB = sql.Open("mysql", dsn)
	if errDB != nil {
		log.Fatal("Error opening database: ", errDB)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	log.Println("Database connected successfully")
}