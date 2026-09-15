package db

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// SeedEquipmentTypes inserts the baseline equipment types used by the MVP.
func SeedEquipmentTypes(ctx context.Context, db *sql.DB) error {
	seedValues := []struct {
		name        string
		minQuantity int
	}{
		{name: "Thin Client", minQuantity: 3},
		{name: "Notebook", minQuantity: 3},
		{name: "Card reader", minQuantity: 3},
		{name: "IP Phone", minQuantity: 3},
		{name: "Monitor LCD", minQuantity: 3},
		{name: "Pocket Wifi", minQuantity: 3},
		{name: "Switch", minQuantity: 3},
		{name: "Telephone Earphone", minQuantity: 3},
	}

	for _, item := range seedValues {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO equipment_types (name, min_quantity)
			VALUES ($1, $2)
			ON CONFLICT (name) DO NOTHING
		`, item.name, item.minQuantity); err != nil {
			return fmt.Errorf(
				"seed equipment type %s: %w",
				item.name,
				err,
			)
		}
	}

	return nil
}

// SeedUsers inserts development users used for local login.
func SeedUsers(ctx context.Context, db *sql.DB) error {
	seedUsers := []struct {
		username string
		password string
		role     string
	}{
		{
			username: "systemadmin",
			password: "system123",
			role:     "system_admin",
		},
		{
			username: "admin",
			password: "admin123",
			role:     "admin",
		},
		{
			username: "user",
			password: "user123",
			role:     "user",
		},
	}

	for _, item := range seedUsers {
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(item.password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return fmt.Errorf(
				"hash password for user %s: %w",
				item.username,
				err,
			)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO users (
				username,
				password_hash,
				is_active,
				role
			)
			VALUES ($1, $2, TRUE, $3)
			ON CONFLICT (username) DO NOTHING
		`,
			item.username,
			string(passwordHash),
			item.role,
		)

		if err != nil {
			return fmt.Errorf(
				"seed user %s: %w",
				item.username,
				err,
			)
		}
	}

	return nil
}