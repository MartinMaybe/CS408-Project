package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "stress" || os.Args[1] == "stress-test") {
		if err := runStressTest(os.Args[2:]); err != nil {
			log.Fatal("Stress test error:", err)
		}
		return
	}

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
