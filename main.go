package main

import (
	"log"
	"net/http"
)

func main() {
	var err error
	appDB, err = initializeDatabase()

	if err != nil {
		log.Fatal("Database initialization error:", err)
	}

	defer appDB.Close()

	registerRoutes()

	log.Println("Starting server on http://localhost:8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
