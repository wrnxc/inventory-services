package service

import (
	"context"
	"database/sql"

	"github.com/wrnxc/inventory-service/internal/repo"
)

type EquipmentType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	MinQuantity int    `json:"min_quantity"`
}

func ListEquipmentTypes(
	ctx context.Context,
	db *sql.DB,
) ([]EquipmentType, error) {
	rows, err := repo.ListEquipmentTypes(ctx, db)
	if err != nil {
		return nil, err
	}

	items := make([]EquipmentType, 0, len(rows))

	for _, row := range rows {
		items = append(items, EquipmentType{
			ID:          row.ID,
			Name:        row.Name,
			MinQuantity: row.MinQuantity,
		})
	}

	return items, nil
}
