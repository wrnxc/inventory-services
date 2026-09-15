package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/wrnxc/inventory-service/internal/db"
	"github.com/wrnxc/inventory-service/internal/handler"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=admin password=password1234 dbname=inventory_service sslmode=disable"
	}

	dbConn, err := db.Open(ctx, db.Config{DSN: dsn})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer dbConn.Close()

	if hours := os.Getenv("SESSION_TTL_HOURS"); hours != "" {
		if parsedHours, err := strconv.Atoi(hours); err == nil && parsedHours > 0 {
			handler.ConfigureSessionTTL(time.Duration(parsedHours) * time.Hour)
		}
	}

	router := handler.NewRouter(dbConn)
	log.Fatal(router.Run(":8080"))
}