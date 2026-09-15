package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type DashboardCounts struct {
	TotalEquipment             int
	EquipmentInUse             int
	EquipmentUnderRepair       int
	EquipmentBelowMinimum      int
	PendingBorrowRequests      int
	PendingReturnConfirmations int
	PendingRepairTickets       int
	SuccessfulBorrows          int
	InProgressRequests         int
}

func GetDashboardCounts(ctx context.Context, db *sql.DB, userID *int) (DashboardCounts, error) {
	var counts DashboardCounts
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM equipment
	`).Scan(&counts.TotalEquipment); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM equipment WHERE status = 'กำลังใช้งาน'
	`).Scan(&counts.EquipmentInUse); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM equipment WHERE status = 'เสียหาย'
	`).Scan(&counts.EquipmentUnderRepair); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT et.id
			FROM equipment_types et
			LEFT JOIN equipment e ON e.type_id = et.id AND e.status = 'ในคลัง'
			GROUP BY et.id, et.min_quantity
			HAVING COUNT(DISTINCT e.id) < et.min_quantity
		) below_minimum
	`).Scan(&counts.EquipmentBelowMinimum); err != nil {
		return counts, err
	}

	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM borrow_records WHERE status = 'รออนุมัติ'
	`).Scan(&counts.PendingBorrowRequests); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM borrow_records WHERE status = 'รอตรวจรับคืน'
	`).Scan(&counts.PendingReturnConfirmations); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM repair_records WHERE status IN ('ส่งซ่อม', 'กำลังซ่อม')
	`).Scan(&counts.PendingRepairTickets); err != nil {
		return counts, err
	}

	if userID == nil {
		return counts, nil
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM borrow_records
		WHERE created_by_user_id = $1 AND status = 'อนุมัติ'
	`, *userID).Scan(&counts.SuccessfulBorrows); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM repair_records
		WHERE reporter_id = $1 AND status IN ('ส่งซ่อม', 'กำลังซ่อม')
	`, *userID).Scan(&counts.EquipmentUnderRepair); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM borrow_records
		WHERE created_by_user_id = $1 AND status IN ('อนุมัติ', 'รอตรวจรับคืน', 'เกินกำหนดคืน')
	`, *userID).Scan(&counts.InProgressRequests); err != nil {
		return counts, err
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM borrow_records
		WHERE created_by_user_id = $1 AND status = 'รออนุมัติ'
	`, *userID).Scan(&counts.PendingBorrowRequests); err != nil {
		return counts, err
	}
	return counts, nil
}

type ReportResult struct {
	Type  string           `json:"type"`
	Items []map[string]any `json:"items"`
}

func GetReport(ctx context.Context, db *sql.DB, reportType string, from, to *time.Time) (ReportResult, error) {
	result := ReportResult{Type: reportType, Items: make([]map[string]any, 0)}
	dateColumn := map[string]string{
		"stock":  "e.created_at",
		"borrow": "b.requested_at",
		"return": "b.returned_at",
		"repair": "r.created_at",
	}[reportType]
	if dateColumn == "" {
		return result, fmt.Errorf("unsupported report type")
	}

	query := ""
	args := make([]any, 0, 2)
	switch reportType {
	case "stock":
		query = `SELECT e.id, e.asset_name, e.status, e.type_id, e.created_at FROM equipment e WHERE 1=1`
	case "borrow":
		query = `SELECT b.id, b.equipment_id, b.created_by_user_id, b.borrower_name, b.status, b.requested_at FROM borrow_records b WHERE 1=1`
	case "return":
		query = `SELECT b.id, b.equipment_id, b.borrower_name, b.status, b.returned_at FROM borrow_records b WHERE b.returned_at IS NOT NULL`
	case "repair":
		query = `SELECT r.id, r.equipment_id, r.reporter_id, r.issue, r.remark, r.status, r.created_at, r.updated_at FROM repair_records r WHERE 1=1`
	}
	if from != nil {
		args = append(args, *from)
		query += " AND " + dateColumn + " >= $" + fmt.Sprintf("%d", len(args))
	}
	if to != nil {
		args = append(args, to.Add(24*time.Hour))
		query += " AND " + dateColumn + " < $" + fmt.Sprintf("%d", len(args))
	}
	query += " ORDER BY " + dateColumn + " DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return result, err
	}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return result, err
		}
		item := make(map[string]any, len(columns))
		for index, column := range columns {
			switch value := values[index].(type) {
			case time.Time:
				item[column] = value
			case []byte:
				item[column] = string(value)
			default:
				item[column] = value
			}
		}
		result.Items = append(result.Items, item)
	}
	return result, rows.Err()
}
