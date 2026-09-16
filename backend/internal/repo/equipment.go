package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

type Equipment struct {
	ID                   int
	TypeID               int
	TypeName             string
	ProductName          sql.NullString
	AssetName            string
	AssetTag             sql.NullString
	AssetSerialNo        sql.NullString
	BarCode              sql.NullString
	VendorName           sql.NullString
	Location             sql.NullString
	AssignedToDepartment sql.NullString
	Site                 sql.NullString
	AssetNo              sql.NullString
	Budget               sql.NullString
	Remark               sql.NullString
	Username             sql.NullString
	ReqNo                sql.NullString
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type EquipmentInput struct {
	TypeID               int
	ProductName          string
	AssetName            string
	AssetTag             string
	AssetSerialNo        string
	BarCode              string
	VendorName           string
	Location             string
	AssignedToDepartment string
	Site                 string
	AssetNo              string
	Budget               string
	Remark               string
	Username             string
	ReqNo                string
	Status               string
}

var (
	ErrAssetNameDuplicated   = errors.New("asset name duplicated")
	ErrAssetSerialDuplicated = errors.New("asset serial duplicated")
	ErrEquipmentNotFound     = errors.New("equipment not found")
	ErrEquipmentTypeNotFound = errors.New("equipment type not found")
)

const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
)

func mapPostgresError(err error) error {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return err
	}

	switch pqErr.Code {
	case uniqueViolationCode:
		switch pqErr.Constraint {
		case "equipment_asset_name_key":
			return ErrAssetNameDuplicated
		case "equipment_asset_serial_no_key":
			return ErrAssetSerialDuplicated
		}
	case foreignKeyViolationCode:
		if pqErr.Constraint == "equipment_type_id_fkey" {
			return ErrEquipmentTypeNotFound
		}
	}

	return err
}

// FindEquipmentTypeIDByName converts the Product Type name from an imported
// inventory file (for example "Notebook") to equipment_types.id.
func FindEquipmentTypeIDByName(ctx context.Context, db *sql.DB, name string) (int, error) {
	var id int

	err := db.QueryRowContext(ctx, `
		SELECT id
		FROM equipment_types
		WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
		LIMIT 1
	`, name).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrEquipmentTypeNotFound
	}
	if err != nil {
		return 0, err
	}

	return id, nil
}

func InsertEquipment(ctx context.Context, db *sql.DB, input EquipmentInput) (*Equipment, error) {
	var item Equipment

	err := db.QueryRowContext(ctx, `
		INSERT INTO equipment (
			type_id,
			product_name,
			asset_name,
			asset_tag,
			asset_serial_no,
			bar_code,
			vendor_name,
			location,
			assigned_to_department,
			site,
			asset_no,
			budget,
			remark,
			username,
			req_no,
			status
		)
		VALUES (
			$1,
			$2,
			$3,
			NULLIF($4, ''),
			NULLIF($5, ''),
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			NULLIF($9, ''),
			NULLIF($10, ''),
			NULLIF($11, ''),
			NULLIF($12, ''),
			NULLIF($13, ''),
			NULLIF($14, ''),
			NULLIF($15, ''),
			'ในคลัง'
		)
		RETURNING
			id,
			type_id,
			product_name,
			asset_name,
			asset_tag,
			asset_serial_no,
			bar_code,
			vendor_name,
			location,
			assigned_to_department,
			site,
			asset_no,
			budget,
			remark,
			username,
			req_no,
			status,
			created_at,
			updated_at
	`,
		input.TypeID,
		input.ProductName,
		input.AssetName,
		input.AssetTag,
		input.AssetSerialNo,
		input.BarCode,
		input.VendorName,
		input.Location,
		input.AssignedToDepartment,
		input.Site,
		input.AssetNo,
		input.Budget,
		input.Remark,
		input.Username,
		input.ReqNo,
	).Scan(
		&item.ID,
		&item.TypeID,
		&item.ProductName,
		&item.AssetName,
		&item.AssetTag,
		&item.AssetSerialNo,
		&item.BarCode,
		&item.VendorName,
		&item.Location,
		&item.AssignedToDepartment,
		&item.Site,
		&item.AssetNo,
		&item.Budget,
		&item.Remark,
		&item.Username,
		&item.ReqNo,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}

	if err := db.QueryRowContext(ctx, `
		SELECT name
		FROM equipment_types
		WHERE id = $1
	`, item.TypeID).Scan(&item.TypeName); err != nil {
		return nil, err
	}

	return &item, nil
}

func ListEquipment(ctx context.Context, db *sql.DB) ([]Equipment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			e.id,
			e.type_id,
			et.name,
			e.product_name,
			e.asset_name,
			e.asset_tag,
			e.asset_serial_no,
			e.bar_code,
			e.vendor_name,
			e.location,
			e.assigned_to_department,
			e.site,
			e.asset_no,
			e.budget,
			e.remark,
			e.username,
			e.req_no,
			e.status,
			e.created_at,
			e.updated_at
		FROM equipment e
		JOIN equipment_types et ON et.id = e.type_id
		ORDER BY e.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Equipment
	for rows.Next() {
		var item Equipment
		if err := rows.Scan(
			&item.ID,
			&item.TypeID,
			&item.TypeName,
			&item.ProductName,
			&item.AssetName,
			&item.AssetTag,
			&item.AssetSerialNo,
			&item.BarCode,
			&item.VendorName,
			&item.Location,
			&item.AssignedToDepartment,
			&item.Site,
			&item.AssetNo,
			&item.Budget,
			&item.Remark,
			&item.Username,
			&item.ReqNo,
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

func UpdateEquipment(ctx context.Context, db *sql.DB, id int, input EquipmentInput) error {
	result, err := db.ExecContext(ctx, `
		UPDATE equipment
		SET
			type_id = $1,
			product_name = $2,
			asset_name = $3,
			asset_tag = NULLIF($4, ''),
			asset_serial_no = NULLIF($5, ''),
			bar_code = NULLIF($6, ''),
			vendor_name = NULLIF($7, ''),
			location = NULLIF($8, ''),
			assigned_to_department = NULLIF($9, ''),
			site = NULLIF($10, ''),
			asset_no = NULLIF($11, ''),
			budget = NULLIF($12, ''),
			remark = NULLIF($13, ''),
			username = NULLIF($14, ''),
			req_no = NULLIF($15, ''),
			status = $16,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $17
	`,
		input.TypeID,
		input.ProductName,
		input.AssetName,
		input.AssetTag,
		input.AssetSerialNo,
		input.BarCode,
		input.VendorName,
		input.Location,
		input.AssignedToDepartment,
		input.Site,
		input.AssetNo,
		input.Budget,
		input.Remark,
		input.Username,
		input.ReqNo,
		input.Status,
		id,
	)
	if err != nil {
		return mapPostgresError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrEquipmentNotFound
	}

	return nil
}

func DeleteEquipment(ctx context.Context, db *sql.DB, id int) error {
	result, err := db.ExecContext(ctx, `DELETE FROM equipment WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrEquipmentNotFound
	}

	return nil
}
