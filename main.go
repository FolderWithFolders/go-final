package main

import (
	"go1f/pkg/db"
	"go1f/pkg/server"
	"log"
)

func main() {
	log.Println("Initializing database...")
	err := db.Init("scheduler.db")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting server...")
	server.StartServer()
}
