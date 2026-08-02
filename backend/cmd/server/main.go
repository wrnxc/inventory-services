package main

import (
	"context"
	"log"

	"github.com/wrnxc/inventory-service/internal/db"
	"github.com/wrnxc/inventory-service/internal/handler"
)

func main() {
	ctx := context.Background()

	dbConn, err := db.Open(ctx, db.Config{
		DSN: "host=localhost port=5432 user=admin password=password1234 dbname=inventory_service sslmode=disable",
	})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer dbConn.Close()

	router := handler.NewRouter()
	log.Fatal(router.Run(":8080"))
}