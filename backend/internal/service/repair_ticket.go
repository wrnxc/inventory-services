package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/wrnxc/inventory-service/internal/repo"
)

type RepairTicketInput struct {
	EquipmentID      int
	IssueDescription string
	Urgency          string
}

type RepairTicket struct {
	ID               int       `json:"id"`
	EquipmentID      int       `json:"equipment_id"`
	ReporterID       int       `json:"reporter_id"`
	IssueDescription string    `json:"issue_description"`
	Urgency          string    `json:"urgency"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

var validRepairTicketStatuses = map[string]bool{
	"ส่งซ่อม":     true,
	"กำลังซ่อม":   true,
	"ซ่อมเสร็จ":   true,
	"ซ่อมไม่ได้":   true,
}

func validateRepairTicketInput(input RepairTicketInput) error {
	if input.EquipmentID <= 0 {
		return NewAppError(422, "VALIDATION_ERROR", "equipment_id is required")
	}
	if strings.TrimSpace(input.IssueDescription) == "" {
		return NewAppError(422, "VALIDATION_ERROR", "issue_description is required")
	}
	if strings.TrimSpace(input.Urgency) == "" {
		return NewAppError(422, "VALIDATION_ERROR", "urgency is required")
	}
	return nil
}

func fromRepoRepairTicket(item *repo.RepairTicket) *RepairTicket {
	if item == nil {
		return nil
	}
	return &RepairTicket{
		ID:               item.ID,
		EquipmentID:      item.EquipmentID,
		ReporterID:       item.ReporterID,
		IssueDescription: item.Issue,
		Urgency:          item.Urgency,
		Status:           item.Status,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func CreateRepairTicket(ctx context.Context, db *sql.DB, actorUserID int, role string, input RepairTicketInput) (*RepairTicket, error) {
	if role != "user" {
		return nil, NewAppError(403, "FORBIDDEN", "only user can create repair tickets")
	}
	if err := validateRepairTicketInput(input); err != nil {
		return nil, err
	}

	var equipmentStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM equipment WHERE id = $1`, input.EquipmentID).Scan(&equipmentStatus); err != nil {
		if err == sql.ErrNoRows {
			return nil, NewAppError(404, "EQUIPMENT_NOT_FOUND", "equipment not found")
		}
		return nil, err
	}
	if equipmentStatus == "เลิกใช้งาน" {
		return nil, NewAppError(409, "EQUIPMENT_DECOMMISSIONED", "equipment is decommissioned")
	}

	created, err := repo.InsertRepairTicket(ctx, db, input.EquipmentID, actorUserID, input.IssueDescription, input.Urgency)
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, `UPDATE equipment SET status = 'เสียหาย', updated_at = NOW() WHERE id = $1`, input.EquipmentID); err != nil {
		return nil, err
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "created", "repair", created.ID, map[string]any{
		"equipment_id":      created.EquipmentID,
		"issue_description": created.Issue,
		"urgency":           created.Urgency,
	}); err != nil {
		return nil, err
	}

	return fromRepoRepairTicket(created), nil
}

func ListRepairTickets(ctx context.Context, db *sql.DB, status string, equipmentID *int) ([]RepairTicket, error) {
	rows, err := repo.ListRepairTickets(ctx, db, status, equipmentID)
	if err != nil {
		return nil, err
	}
	items := make([]RepairTicket, 0, len(rows))
	for _, row := range rows {
		items = append(items, *fromRepoRepairTicket(&row))
	}
	return items, nil
}

func GetRepairTicket(ctx context.Context, db *sql.DB, id int) (*RepairTicket, error) {
	item, err := repo.GetRepairTicketByID(ctx, db, id)
	if err != nil {
		if err == repo.ErrRepairTicketNotFound {
			return nil, NewAppError(404, "REPAIR_TICKET_NOT_FOUND", "repair ticket not found")
		}
		return nil, err
	}
	return fromRepoRepairTicket(item), nil
}

func UpdateRepairTicketStatus(ctx context.Context, db *sql.DB, actorUserID int, role string, id int, newStatus string) (*RepairTicket, error) {
	if role != "admin" && role != "system_admin" {
		return nil, NewAppError(403, "FORBIDDEN", "only admin or system admin can update repair ticket status")
	}
	if !validRepairTicketStatuses[newStatus] {
		return nil, NewAppError(422, "INVALID_STATUS", "invalid repair ticket status")
	}

	item, err := repo.UpdateRepairTicketStatus(ctx, db, id, newStatus)
	if err != nil {
		switch err {
		case repo.ErrRepairTicketNotFound:
			return nil, NewAppError(404, "REPAIR_TICKET_NOT_FOUND", "repair ticket not found")
		case repo.ErrRepairTicketAlreadyClosed:
			return nil, NewAppError(409, "REPAIR_TICKET_ALREADY_CLOSED", "repair ticket is already closed")
		default:
			return nil, err
		}
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "status_updated", "repair", item.ID, map[string]any{
		"equipment_id": item.EquipmentID,
		"status":       item.Status,
	}); err != nil {
		return nil, err
	}

	return fromRepoRepairTicket(item), nil
}

func ListActivityLogs(ctx context.Context, db *sql.DB) ([]ActivityLog, error) {
	rows, err := repo.ListActivityLogs(ctx, db)
	if err != nil {
		return nil, err
	}
	items := make([]ActivityLog, 0, len(rows))
	for _, row := range rows {
		meta := map[string]any{}
		if len(row.Metadata) > 0 && string(row.Metadata) != "null" {
			if err := json.Unmarshal(row.Metadata, &meta); err != nil {
				return nil, err
			}
		}
		items = append(items, ActivityLog{
			ID:           row.ID,
			UserID:       row.UserID,
			Action:       row.Action,
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			Metadata:     meta,
			CreatedAt:    row.CreatedAt,
		})
	}
	return items, nil
}

type ActivityLog struct {
	ID           int            `json:"id"`
	UserID       int            `json:"user_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   int            `json:"resource_id"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}
