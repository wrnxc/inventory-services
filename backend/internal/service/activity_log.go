package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// RecordActivityLog writes a single activity log row for downstream workflow steps.
func RecordActivityLog(ctx context.Context, db *sql.DB, userID int, action, resourceType string, resourceID int, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal activity log metadata: %w", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO activity_logs (user_id, action, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, action, resourceType, resourceID, string(payload))
	if err != nil {
		return fmt.Errorf("insert activity log: %w", err)
	}

	return nil
}
