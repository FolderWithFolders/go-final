package server

import (
	"go1f/pkg/api"
	"go1f/pkg/config"
	"go1f/pkg/db"
	"log"
	"net/http"
)

func StartServer(store *db.Store, cfg *config.Config) {
	api := api.NewAPI(store, cfg)
	api.Init()

	server := http.FileServer(http.Dir("./web"))
	http.Handle("/", server)

	log.Println("Сервер запущен на порту:", cfg.Port)
	err := http.ListenAndServe(":"+cfg.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
