package service

import (
	"context"
	"testing"
	"time"

	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestRecordActivityLogPersistsRequiredFields(t *testing.T) {
	ctx := context.Background()
	dbConn := testutil.NewPostgres(t)

	if _, err := dbConn.ExecContext(ctx, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := dbConn.ExecContext(ctx, `
		CREATE TABLE activity_logs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id INTEGER NOT NULL,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatalf("create activity_logs table: %v", err)
	}
	if _, err := dbConn.ExecContext(ctx, `INSERT INTO users (username, role) VALUES ($1, $2)`, "alice", "user"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := RecordActivityLog(ctx, dbConn, 1, "created", "equipment", 42, map[string]any{"source": "test"}); err != nil {
		t.Fatalf("record activity log: %v", err)
	}

	var (
		id           int
		userID       int
		action       string
		resourceType string
		resourceID   int
		metadata     string
		createdAt    time.Time
	)
	if err := dbConn.QueryRowContext(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, metadata::text, created_at
		FROM activity_logs
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&id, &userID, &action, &resourceType, &resourceID, &metadata, &createdAt); err != nil {
		t.Fatalf("read activity log row: %v", err)
	}

	if id == 0 || userID != 1 || action != "created" || resourceType != "equipment" || resourceID != 42 {
		t.Fatalf("unexpected persisted activity log fields: id=%d user_id=%d action=%q resource_type=%q resource_id=%d", id, userID, action, resourceType, resourceID)
	}
	if metadata == "" || metadata == "<nil>" {
		t.Fatalf("expected metadata to be persisted, got %q", metadata)
	}
	if createdAt.IsZero() {
		t.Fatalf("expected created_at to be populated")
	}
}