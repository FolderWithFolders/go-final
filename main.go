package main

import (
	"go1f/pkg/config"
	"go1f/pkg/db"
	"go1f/pkg/server"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	log.Println("Инициализация базы данных...")
	store, err := db.NewStore("scheduler.db")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	log.Println("Запуск сервера...")
	server.StartServer(store, cfg)
}
