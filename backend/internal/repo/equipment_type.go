package repo

import (
	"context"
	"database/sql"
)

type EquipmentType struct {
	ID          int
	Name        string
	MinQuantity int
}

func ListEquipmentTypes(
	ctx context.Context,
	db *sql.DB,
) ([]EquipmentType, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, min_quantity
		FROM equipment_types
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []EquipmentType

	for rows.Next() {
		var item EquipmentType

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.MinQuantity,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}
