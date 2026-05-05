package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	host := getenv("DB_HOST", "localhost")
	user := getenv("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := getenv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s", host, user, password, dbname, sslmode)

	var err error

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 30; i++ {
		err = DB.Ping()
		if err == nil {
			fmt.Println("Database connected")
			return
		}
		fmt.Printf("Database connection attempt %d/30 failed: %v. Retrying...\n", i+1, err)
		time.Sleep(1 * time.Second)
	}

	panic(fmt.Sprintf("Failed to connect to database after 30 attempts: %v", err))
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
