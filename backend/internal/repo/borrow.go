package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type BorrowRequest struct {
	ID              int
	EquipmentID     int
	CreatedByUserID int
	BorrowerName    string
	ApprovedBy      sql.NullInt64
	BorrowType      string
	ReturnDate      sql.NullTime
	Reason          sql.NullString
	Remark          sql.NullString
	Status          string
	RequestedAt     time.Time
	ApprovedAt      sql.NullTime
	ReturnedAt      sql.NullTime
}

var (
	ErrBorrowRequestNotFound = errors.New("borrow request not found")
	ErrBorrowRequestNotPending = errors.New("borrow request not pending")
	ErrEquipmentAlreadyRequested = errors.New("equipment already requested")
	ErrEquipmentNotAvailable = errors.New("equipment not available")
	ErrBorrowRequestNotReturnable = errors.New("borrow request not returnable")
	ErrBorrowRequestNotPendingReturn = errors.New("borrow request not pending return")
)

func mapBorrowPostgresError(err error) error {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return err
	}

	switch pqErr.Code {
	case uniqueViolationCode:
		if pqErr.Constraint == "one_active_borrow_per_equipment" {
			return ErrEquipmentAlreadyRequested
		}
	case foreignKeyViolationCode:
		if pqErr.Constraint == "borrow_records_equipment_id_fkey" || pqErr.Constraint == "borrow_records_created_by_user_id_fkey" {
			return ErrEquipmentNotFound
		}
	}

	return err
}

func GetBorrowRequestByID(ctx context.Context, db *sql.DB, id int) (*BorrowRequest, error) {
	var item BorrowRequest

	err := db.QueryRowContext(ctx, `
		SELECT
			id,
			equipment_id,
			created_by_user_id,
			borrower_name,
			approved_by,
			borrow_type,
			return_date,
			reason,
			remark,
			status,
			requested_at,
			approved_at,
			returned_at
		FROM borrow_records
		WHERE id = $1
	`, id).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.CreatedByUserID,
		&item.BorrowerName,
		&item.ApprovedBy,
		&item.BorrowType,
		&item.ReturnDate,
		&item.Reason,
		&item.Remark,
		&item.Status,
		&item.RequestedAt,
		&item.ApprovedAt,
		&item.ReturnedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBorrowRequestNotFound
		}
		return nil, err
	}

	// Derived overdue status: temporary borrow that is approved and past return_date
	if item.BorrowType == "เบิกชั่วคราว" && item.Status == "อนุมัติ" && item.ReturnDate.Valid {
		if item.ReturnDate.Time.Before(time.Now()) {
			item.Status = "เกินกำหนดคืน"
		}
	}

	return &item, nil
}

func ListBorrowRequests(ctx context.Context, db *sql.DB, status string, equipmentID *int, borrowerName string) ([]BorrowRequest, error) {
	query := `
		SELECT
			id,
			equipment_id,
			created_by_user_id,
			borrower_name,
			approved_by,
			borrow_type,
			return_date,
			reason,
			remark,
			status,
			requested_at,
			approved_at,
			returned_at
		FROM borrow_records
	`
	args := make([]any, 0, 3)
	conditions := make([]string, 0, 3)

	if strings.TrimSpace(status) != "" {
		s := strings.TrimSpace(status)
		// support derived 'เกินกำหนดคืน' status: either stored as such OR computed from temporary approved past return_date
		if s == "เกินกำหนดคืน" {
			conditions = append(conditions, "(status = 'เกินกำหนดคืน' OR (borrow_type = 'เบิกชั่วคราว' AND status = 'อนุมัติ' AND return_date < CURRENT_DATE))")
		} else {
			conditions = append(conditions, "status = $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, s)
		}
	}
	if equipmentID != nil && *equipmentID > 0 {
		conditions = append(conditions, "equipment_id = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, *equipmentID)
	}
	if strings.TrimSpace(borrowerName) != "" {
		conditions = append(conditions, "borrower_name ILIKE $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, "%"+strings.TrimSpace(borrowerName)+"%")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY requested_at DESC, id DESC"

	// debug: log query and args when troubleshooting overdue detection
	// fmt.Printf("ListBorrowRequests QUERY: %s ARGS: %#v\n", query, args)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BorrowRequest, 0)
	for rows.Next() {
		var item BorrowRequest
		if err := rows.Scan(
			&item.ID,
			&item.EquipmentID,
			&item.CreatedByUserID,
			&item.BorrowerName,
			&item.ApprovedBy,
			&item.BorrowType,
			&item.ReturnDate,
			&item.Reason,
			&item.Remark,
			&item.Status,
			&item.RequestedAt,
			&item.ApprovedAt,
			&item.ReturnedAt,
		); err != nil {
			return nil, err
		}
		// apply derived overdue status when reading rows
		if item.BorrowType == "เบิกชั่วคราว" && item.Status == "อนุมัติ" && item.ReturnDate.Valid {
			if item.ReturnDate.Time.Before(time.Now()) {
				item.Status = "เกินกำหนดคืน"
			}
		}
		items = append(items, item)
	}
	// debug: report number of rows returned
	// fmt.Printf("ListBorrowRequests returned %d rows\n", len(items))

	return items, rows.Err()
}

func InsertBorrowRequest(ctx context.Context, db *sql.DB, equipmentID, createdByUserID int, borrowerName, borrowType, returnDate string) (*BorrowRequest, error) {
	item := BorrowRequest{}

	var returnDateValue interface{}
	if strings.TrimSpace(returnDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(returnDate))
		if err != nil {
			return nil, fmt.Errorf("parse return date: %w", err)
		}
		returnDateValue = parsed
	} else {
		returnDateValue = nil
	}

	err := db.QueryRowContext(ctx, `
		INSERT INTO borrow_records (
			equipment_id,
			created_by_user_id,
			borrower_name,
			borrow_type,
			return_date,
			status
		)
		VALUES ($1, $2, $3, $4, $5, 'รออนุมัติ')
		RETURNING
			id,
			equipment_id,
			created_by_user_id,
			borrower_name,
			approved_by,
			borrow_type,
			return_date,
			reason,
			remark,
			status,
			requested_at,
			approved_at,
			returned_at
	`, equipmentID, createdByUserID, strings.TrimSpace(borrowerName), borrowType, returnDateValue).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.CreatedByUserID,
		&item.BorrowerName,
		&item.ApprovedBy,
		&item.BorrowType,
		&item.ReturnDate,
		&item.Reason,
		&item.Remark,
		&item.Status,
		&item.RequestedAt,
		&item.ApprovedAt,
		&item.ReturnedAt,
	)
	if err != nil {
		return nil, mapBorrowPostgresError(err)
	}

	return &item, nil
}

func ApproveBorrowRequest(ctx context.Context, db *sql.DB, id int, approverID int) (*BorrowRequest, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Update borrow_records only when status is pending ('รออนุมัติ')
	var item BorrowRequest
	err = tx.QueryRowContext(ctx, `
		UPDATE borrow_records
		SET status = 'อนุมัติ', approved_by = $2, approved_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'รออนุมัติ'
		RETURNING id, equipment_id, created_by_user_id, borrower_name, approved_by, borrow_type, return_date, reason, remark, status, requested_at, approved_at, returned_at
	`, id, approverID).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.CreatedByUserID,
		&item.BorrowerName,
		&item.ApprovedBy,
		&item.BorrowType,
		&item.ReturnDate,
		&item.Reason,
		&item.Remark,
		&item.Status,
		&item.RequestedAt,
		&item.ApprovedAt,
		&item.ReturnedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// determine whether not found or not pending
			var curStatus string
			if err2 := tx.QueryRowContext(ctx, `SELECT status FROM borrow_records WHERE id = $1`, id).Scan(&curStatus); err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					tx.Rollback()
					return nil, ErrBorrowRequestNotFound
				}
				tx.Rollback()
				return nil, err2
			}
			tx.Rollback()
			return nil, ErrBorrowRequestNotPending
		}
		tx.Rollback()
		return nil, err
	}

	// update equipment status to 'กำลังใช้งาน'
	if _, err := tx.ExecContext(ctx, `UPDATE equipment SET status = 'กำลังใช้งาน' WHERE id = $1`, item.EquipmentID); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &item, nil
}

func RejectBorrowRequest(ctx context.Context, db *sql.DB, id int, approverID int) (*BorrowRequest, error) {
	// reject does not change equipment status
	var item BorrowRequest
	err := db.QueryRowContext(ctx, `
		UPDATE borrow_records
		SET status = 'ไม่อนุมัติ', approved_by = $2, approved_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'รออนุมัติ'
		RETURNING id, equipment_id, created_by_user_id, borrower_name, approved_by, borrow_type, return_date, reason, remark, status, requested_at, approved_at, returned_at
	`, id, approverID).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.CreatedByUserID,
		&item.BorrowerName,
		&item.ApprovedBy,
		&item.BorrowType,
		&item.ReturnDate,
		&item.Reason,
		&item.Remark,
		&item.Status,
		&item.RequestedAt,
		&item.ApprovedAt,
		&item.ReturnedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var curStatus string
			if err2 := db.QueryRowContext(ctx, `SELECT status FROM borrow_records WHERE id = $1`, id).Scan(&curStatus); err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					return nil, ErrBorrowRequestNotFound
				}
				return nil, err2
			}
			return nil, ErrBorrowRequestNotPending
		}
		return nil, err
	}

	return &item, nil
}

func RequestReturnBorrow(ctx context.Context, db *sql.DB, id int) (*BorrowRequest, error) {
	var item BorrowRequest
	err := db.QueryRowContext(ctx, `
		UPDATE borrow_records
		SET status = 'รอตรวจรับคืน'
		WHERE id = $1 AND status IN ('อนุมัติ', 'เกินกำหนดคืน')
		RETURNING id, equipment_id, created_by_user_id, borrower_name, approved_by, borrow_type, return_date, reason, remark, status, requested_at, approved_at, returned_at
	`, id).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.CreatedByUserID,
		&item.BorrowerName,
		&item.ApprovedBy,
		&item.BorrowType,
		&item.ReturnDate,
		&item.Reason,
		&item.Remark,
		&item.Status,
		&item.RequestedAt,
		&item.ApprovedAt,
		&item.ReturnedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var curStatus string
			if err2 := db.QueryRowContext(ctx, `SELECT status FROM borrow_records WHERE id = $1`, id).Scan(&curStatus); err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					return nil, ErrBorrowRequestNotFound
				}
				return nil, err2
			}
			return nil, ErrBorrowRequestNotReturnable
		}
		return nil, err
	}

	return &item, nil
}

func ConfirmReturnBorrowRequest(ctx context.Context, db *sql.DB, id int, approverID int) (*BorrowRequest, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var item BorrowRequest
	err = tx.QueryRowContext(ctx, `
		UPDATE borrow_records
		SET status = 'คืนแล้ว', returned_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'รอตรวจรับคืน'
		RETURNING id, equipment_id, created_by_user_id, borrower_name, approved_by, borrow_type, return_date, reason, remark, status, requested_at, approved_at, returned_at
	`, id).Scan(
		&item.ID,
		&item.EquipmentID,
		&item.CreatedByUserID,
		&item.BorrowerName,
		&item.ApprovedBy,
		&item.BorrowType,
		&item.ReturnDate,
		&item.Reason,
		&item.Remark,
		&item.Status,
		&item.RequestedAt,
		&item.ApprovedAt,
		&item.ReturnedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var curStatus string
			if err2 := tx.QueryRowContext(ctx, `SELECT status FROM borrow_records WHERE id = $1`, id).Scan(&curStatus); err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					tx.Rollback()
					return nil, ErrBorrowRequestNotFound
				}
				tx.Rollback()
				return nil, err2
			}
			tx.Rollback()
			return nil, ErrBorrowRequestNotPendingReturn
		}
		tx.Rollback()
		return nil, err
	}

	// set equipment back to 'ในคลัง'
	if _, err := tx.ExecContext(ctx, `UPDATE equipment SET status = 'ในคลัง' WHERE id = $1`, item.EquipmentID); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &item, nil
}
