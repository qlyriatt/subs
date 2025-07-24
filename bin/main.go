package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"subs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	if s := os.Getenv("LOG_TO_FILE"); s != "" {
		i, _ := strconv.Atoi(s)
		if i == 1 {
			file, err := os.OpenFile("subs.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				log.Println("unable to set custom logger in subs.log")
			} else {
				subs.SetLogger(log.New(file, "[subs] ", log.LstdFlags))
			}
		}
	}

	conn_str := fmt.Sprintf("postgres://%v:%v@%v:5432/%v?sslmode=disable", os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_HOST"), os.Getenv("DB_DB"))

	db, err := subs.NewPGXDB(conn_str)
	if err != nil {
		log.Fatalf("postgres connection error: %v", err)
	}

	mig, err := migrate.New("file://", conn_str)
	if err != nil {
		log.Fatalf("migration error: %v", err)
	}

	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration error: %v", err)
	}

	subs.Start(db)
}
