package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/wrnxc/inventory-service/internal/repo"
)

type BorrowRequestInput struct {
	EquipmentID  int
	BorrowType   string
	ReturnDate   string
	BorrowerName string
}

type BorrowRequest struct {
	ID              int        `json:"id"`
	EquipmentID     int        `json:"equipment_id"`
	CreatedByUserID int        `json:"created_by_user_id"`
	BorrowerName    string     `json:"borrower_name"`
	BorrowType      string     `json:"borrow_type"`
	ReturnDate      string     `json:"return_date,omitempty"`
	Status          string     `json:"status"`
	ApprovedBy      *int       `json:"approved_by,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`
}

func validateBorrowRequestInput(input BorrowRequestInput) error {
	if input.EquipmentID <= 0 {
		return NewAppError(422, "VALIDATION_ERROR", "equipment_id is required")
	}
	if strings.TrimSpace(input.BorrowerName) == "" {
		return NewAppError(422, "VALIDATION_ERROR", "borrower_name is required")
	}
	if input.BorrowType != "เบิกถาวร" && input.BorrowType != "เบิกชั่วคราว" {
		return NewAppError(422, "INVALID_BORROW_TYPE", "invalid borrow type")
	}
	if input.BorrowType == "เบิกชั่วคราว" && strings.TrimSpace(input.ReturnDate) == "" {
		return NewAppError(422, "VALIDATION_ERROR", "return_date is required for temporary borrow")
	}
	return nil
}

func formatReturnDate(date sql.NullTime) string {
	if !date.Valid {
		return ""
	}
	return date.Time.Format("2006-01-02")
}

func toBorrowRequestSummary(item *repo.BorrowRequest) *BorrowRequest {
	if item == nil {
		return nil
	}

	result := &BorrowRequest{
		ID:              item.ID,
		EquipmentID:     item.EquipmentID,
		CreatedByUserID: item.CreatedByUserID,
		BorrowerName:    item.BorrowerName,
		BorrowType:      item.BorrowType,
		ReturnDate:      formatReturnDate(item.ReturnDate),
		Status:          item.Status,
	}

	if item.ApprovedBy.Valid {
		v := int(item.ApprovedBy.Int64)
		result.ApprovedBy = &v
	}
	if item.ReturnedAt.Valid {
		v := item.ReturnedAt.Time
		result.ReturnedAt = &v
	}

	return result
}

func CreateBorrowRequest(ctx context.Context, db *sql.DB, actorUserID int, role string, input BorrowRequestInput) (*BorrowRequest, error) {
	if role != "user" {
		return nil, NewAppError(403, "FORBIDDEN", "only user can create borrow requests")
	}
	if err := validateBorrowRequestInput(input); err != nil {
		return nil, err
	}

	var equipmentStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM equipment WHERE id = $1`, input.EquipmentID).Scan(&equipmentStatus); err != nil {
		if err == sql.ErrNoRows {
			return nil, NewAppError(404, "EQUIPMENT_NOT_FOUND", "equipment not found")
		}
		return nil, err
	}
	if equipmentStatus != "ในคลัง" {
		return nil, NewAppError(409, "EQUIPMENT_NOT_AVAILABLE", "equipment is not available")
	}

	created, err := repo.InsertBorrowRequest(ctx, db, input.EquipmentID, actorUserID, input.BorrowerName, input.BorrowType, input.ReturnDate)
	if err != nil {
		switch err {
		case repo.ErrEquipmentAlreadyRequested:
			return nil, NewAppError(409, "EQUIPMENT_ALREADY_REQUESTED", "equipment already has an active borrow request")
		case repo.ErrEquipmentNotFound:
			return nil, NewAppError(404, "EQUIPMENT_NOT_FOUND", "equipment not found")
		default:
			return nil, err
		}
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "created", "borrow", created.ID, map[string]any{
		"equipment_id": created.EquipmentID,
		"borrow_type": created.BorrowType,
		"borrower_name": created.BorrowerName,
	}); err != nil {
		return nil, err
	}

	return toBorrowRequestSummary(created), nil
}

func GetBorrowRequest(ctx context.Context, db *sql.DB, id int) (*BorrowRequest, error) {
	item, err := repo.GetBorrowRequestByID(ctx, db, id)
	if err != nil {
		if err == repo.ErrBorrowRequestNotFound {
			return nil, NewAppError(404, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
		}
		return nil, err
	}
	return toBorrowRequestSummary(item), nil
}

func ListBorrowRequests(ctx context.Context, db *sql.DB, status string, equipmentID *int, borrowerName string) ([]BorrowRequest, error) {
	rows, err := repo.ListBorrowRequests(ctx, db, status, equipmentID, borrowerName)
	if err != nil {
		return nil, err
	}
	items := make([]BorrowRequest, 0, len(rows))
	for _, row := range rows {
		items = append(items, *toBorrowRequestSummary(&row))
	}
	return items, nil
}

func ApproveBorrowRequest(ctx context.Context, db *sql.DB, actorUserID int, role string, id int) (*BorrowRequest, error) {
	if role != "admin" && role != "system_admin" {
		return nil, NewAppError(403, "FORBIDDEN", "only admin or system_admin can approve borrow requests")
	}

	item, err := repo.ApproveBorrowRequest(ctx, db, id, actorUserID)
	if err != nil {
		switch err {
		case repo.ErrBorrowRequestNotFound:
			return nil, NewAppError(404, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
		case repo.ErrBorrowRequestNotPending:
			return nil, NewAppError(409, "BORROW_REQUEST_ALREADY_APPROVED", "borrow request already approved or not pending")
		default:
			return nil, err
		}
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "approved", "borrow", item.ID, map[string]any{"equipment_id": item.EquipmentID}); err != nil {
		return nil, err
	}

	return toBorrowRequestSummary(item), nil
}

func RejectBorrowRequest(ctx context.Context, db *sql.DB, actorUserID int, role string, id int) (*BorrowRequest, error) {
	if role != "admin" && role != "system_admin" {
		return nil, NewAppError(403, "FORBIDDEN", "only admin or system_admin can reject borrow requests")
	}

	item, err := repo.RejectBorrowRequest(ctx, db, id, actorUserID)
	if err != nil {
		switch err {
		case repo.ErrBorrowRequestNotFound:
			return nil, NewAppError(404, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
		case repo.ErrBorrowRequestNotPending:
			return nil, NewAppError(409, "BORROW_REQUEST_NOT_PENDING", "borrow request is not pending and cannot be rejected")
		default:
			return nil, err
		}
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "rejected", "borrow", item.ID, map[string]any{"equipment_id": item.EquipmentID}); err != nil {
		return nil, err
	}

	return toBorrowRequestSummary(item), nil
}

func RequestReturnBorrowRequest(ctx context.Context, db *sql.DB, actorUserID int, role string, id int) (*BorrowRequest, error) {
	if role != "user" {
		return nil, NewAppError(403, "FORBIDDEN", "only user can request return of borrow requests")
	}

	// ensure the actor is the creator of the borrow request
	item, err := repo.GetBorrowRequestByID(ctx, db, id)
	if err != nil {
		if err == repo.ErrBorrowRequestNotFound {
			return nil, NewAppError(404, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
		}
		return nil, err
	}
	if item.CreatedByUserID != actorUserID {
		return nil, NewAppError(403, "FORBIDDEN", "only owner can request return")
	}

	updated, err := repo.RequestReturnBorrow(ctx, db, id)
	if err != nil {
		switch err {
		case repo.ErrBorrowRequestNotFound:
			return nil, NewAppError(404, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
		case repo.ErrBorrowRequestNotReturnable:
			return nil, NewAppError(409, "BORROW_REQUEST_NOT_RETURNABLE", "borrow request is not returnable")
		default:
			return nil, err
		}
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "returned", "borrow", updated.ID, map[string]any{"equipment_id": updated.EquipmentID}); err != nil {
		return nil, err
	}

	return toBorrowRequestSummary(updated), nil
}

func ConfirmReturnBorrowRequest(ctx context.Context, db *sql.DB, actorUserID int, role string, id int) (*BorrowRequest, error) {
	if role != "admin" && role != "system_admin" {
		return nil, NewAppError(403, "FORBIDDEN", "only admin or system_admin can confirm return")
	}

	item, err := repo.ConfirmReturnBorrowRequest(ctx, db, id, actorUserID)
	if err != nil {
		switch err {
		case repo.ErrBorrowRequestNotFound:
			return nil, NewAppError(404, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
		case repo.ErrBorrowRequestNotPendingReturn:
			return nil, NewAppError(409, "BORROW_REQUEST_NOT_PENDING_RETURN", "borrow request is not pending return")
		default:
			return nil, err
		}
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "confirmed_return", "borrow", item.ID, map[string]any{"equipment_id": item.EquipmentID}); err != nil {
		return nil, err
	}

	return toBorrowRequestSummary(item), nil
}
