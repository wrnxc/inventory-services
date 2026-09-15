package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// NewPostgres starts an ephemeral Postgres container for the duration of the
// test and returns a ready-to-use *sql.DB. The container is terminated
// automatically via t.Cleanup, and the connection is closed with it.
func NewPostgres(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	container, err := postgrescontainer.Run(ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("inventory_test"),
		postgrescontainer.WithUsername("inventory"),
		postgrescontainer.WithPassword("inventory"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get postgres connection string: %v", err)
	}

	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("open postgres connection: %v", err)
	}
	t.Cleanup(func() {
		dbConn.Close()
	})

	var pingErr error
	for i := 0; i < 30; i++ {
		pingErr = dbConn.PingContext(ctx)
		if pingErr == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if pingErr != nil {
		t.Fatalf("ping postgres after retries: %v", pingErr)
	}

	return dbConn
}