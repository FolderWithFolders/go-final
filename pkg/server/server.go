package server

import (
	"log"
	"net/http"
	"os"

	"go1f/pkg/api"
)

func StartServer() {
	api.Init() // Инициализация API

	server := http.FileServer(http.Dir("./web"))
	http.Handle("/", server)

	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	log.Println("Listening on :" + port + "...")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
