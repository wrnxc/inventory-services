package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type RepairTicket struct {
	ID              int
	EquipmentID     int
	ReporterID      int
	Issue           string
	Urgency         string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var (
	ErrRepairTicketNotFound    = errors.New("repair ticket not found")
	ErrRepairTicketAlreadyClosed = errors.New("repair ticket already closed")
	ErrEquipmentDecommissioned = errors.New("equipment decommissioned")
)

func GetRepairTicketByID(ctx context.Context, db *sql.DB, id int) (*RepairTicket, error) {
	var item RepairTicket
	if err := db.QueryRowContext(ctx, `
		SELECT id, equipment_id, reporter_id, issue, remark, status, created_at, updated_at
		FROM repair_records
		WHERE id = $1
	`, id).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.ReporterID,
		&item.Issue,
		&item.Urgency,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRepairTicketNotFound
		}
		return nil, err
	}
	return &item, nil
}

func ListRepairTickets(ctx context.Context, db *sql.DB, status string, equipmentID *int) ([]RepairTicket, error) {
	query := `
		SELECT id, equipment_id, reporter_id, issue, remark, status, created_at, updated_at
		FROM repair_records
	`
	args := make([]any, 0, 4)
	conditions := make([]string, 0, 2)

	if strings.TrimSpace(status) != "" {
		conditions = append(conditions, "status = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, strings.TrimSpace(status))
	}
	if equipmentID != nil && *equipmentID > 0 {
		conditions = append(conditions, "equipment_id = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, *equipmentID)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RepairTicket, 0)
	for rows.Next() {
		var item RepairTicket
		if err := rows.Scan(
			&item.ID,
			&item.EquipmentID,
			&item.ReporterID,
			&item.Issue,
			&item.Urgency,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func InsertRepairTicket(ctx context.Context, db *sql.DB, equipmentID int, reporterID int, issueDescription string, urgency string) (*RepairTicket, error) {
	var item RepairTicket
	if err := db.QueryRowContext(ctx, `
		INSERT INTO repair_records (equipment_id, reporter_id, issue, remark, status)
		VALUES ($1, $2, $3, $4, 'ส่งซ่อม')
		RETURNING id, equipment_id, reporter_id, issue, remark, status, created_at, updated_at
	`, equipmentID, reporterID, strings.TrimSpace(issueDescription), strings.TrimSpace(urgency)).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.ReporterID,
		&item.Issue,
		&item.Urgency,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRepairTicketNotFound
		}
		return nil, err
	}
	return &item, nil
}

func UpdateRepairTicketStatus(ctx context.Context, db *sql.DB, id int, newStatus string) (*RepairTicket, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var item RepairTicket
	if err := tx.QueryRowContext(ctx, `
		SELECT id, equipment_id, reporter_id, issue, remark, status, created_at, updated_at
		FROM repair_records
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.ReporterID,
		&item.Issue,
		&item.Urgency,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRepairTicketNotFound
		}
		return nil, err
	}
	if item.Status == "ซ่อมเสร็จ" || item.Status == "ซ่อมไม่ได้" {
		return nil, ErrRepairTicketAlreadyClosed
	}

	if err := tx.QueryRowContext(ctx, `
		UPDATE repair_records
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, equipment_id, reporter_id, issue, remark, status, created_at, updated_at
	`, id, newStatus).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.ReporterID,
		&item.Issue,
		&item.Urgency,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	var equipmentStatus string
	switch newStatus {
	case "ซ่อมเสร็จ":
		equipmentStatus = "ในคลัง"
	case "ซ่อมไม่ได้":
		equipmentStatus = "เลิกใช้งาน"
	default:
		equipmentStatus = "เสียหาย"
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE equipment
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, item.EquipmentID, equipmentStatus); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &item, nil
}

type ActivityLog struct {
	ID          int
	UserID      int
	Action      string
	ResourceType string
	ResourceID  int
	Metadata    []byte
	CreatedAt   time.Time
}

func ListActivityLogs(ctx context.Context, db *sql.DB) ([]ActivityLog, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, metadata, created_at
		FROM activity_logs
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ActivityLog, 0)
	for rows.Next() {
		var item ActivityLog
		if err := rows.Scan(&item.ID, &item.UserID, &item.Action, &item.ResourceType, &item.ResourceID, &item.Metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
