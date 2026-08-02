package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMigrateAppliesSchemaAndSeeds(t *testing.T) {
	postgresCtx := context.Background()
	postgresContainer, err := postgrescontainer.Run(postgresCtx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("inventory_test"),
		postgrescontainer.WithUsername("inventory"),
		postgrescontainer.WithPassword("inventory"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer func() {
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	}()

	connStr, err := postgresContainer.ConnectionString(postgresCtx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get postgres connection string: %v", err)
	}

	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("open postgres connection: %v", err)
	}
	defer dbConn.Close()

	for i := 0; i < 15; i++ {
		err = dbConn.PingContext(postgresCtx)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		t.Fatalf("ping postgres after retries: %v", err)
	}

	if err := Migrate(postgresCtx, dbConn); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	assertMigrationTableExists(t, dbConn, "users")
	assertMigrationTableExists(t, dbConn, "equipment_types")
	assertMigrationTableExists(t, dbConn, "equipment")
	assertMigrationTableExists(t, dbConn, "borrow_records")
	assertMigrationTableExists(t, dbConn, "repair_records")
	assertMigrationTableExists(t, dbConn, "activity_logs")
	assertMigrationIndexExists(t, dbConn, "one_active_borrow_per_equipment")
	assertMigrationSeededEquipmentTypes(t, dbConn)
}

func assertMigrationTableExists(t *testing.T, dbConn *sql.DB, tableName string) {
	t.Helper()
	var exists bool
	err := dbConn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("check table %s: %v", tableName, err)
	}
	if !exists {
		t.Fatalf("expected table %s to exist", tableName)
	}
}

func assertMigrationIndexExists(t *testing.T, dbConn *sql.DB, indexName string) {
	t.Helper()
	var exists bool
	err := dbConn.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1)`, indexName).Scan(&exists)
	if err != nil {
		t.Fatalf("check index %s: %v", indexName, err)
	}
	if !exists {
		t.Fatalf("expected index %s to exist", indexName)
	}
}

func assertMigrationSeededEquipmentTypes(t *testing.T, dbConn *sql.DB) {
	t.Helper()
	var count int
	err := dbConn.QueryRow(`SELECT COUNT(*) FROM equipment_types`).Scan(&count)
	if err != nil {
		t.Fatalf("count seed rows: %v", err)
	}
	if count < 5 {
		t.Fatalf("expected at least 5 seeded equipment types, got %d", count)
	}
}
