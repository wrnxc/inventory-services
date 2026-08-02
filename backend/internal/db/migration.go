package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config captures the database settings used by the migration layer.
type Config struct {
	DSN string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLifetime time.Duration
}

// Open opens a PostgreSQL connection and applies the schema migration.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres database: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres database: %w", err)
	}

	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// Migrate applies the schema and seed data required by the inventory service MVP.
func Migrate(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`SET statement_timeout = '5000ms'`,
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE CHECK (length(trim(username)) > 0),
			role TEXT NOT NULL CHECK (role IN ('system_admin', 'admin', 'user'))
		)`,
		`CREATE TABLE IF NOT EXISTS equipment_types (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
			min_quantity INTEGER NOT NULL DEFAULT 0 CHECK (min_quantity >= 0)
		)`,
		`CREATE TABLE IF NOT EXISTS equipment (
			id SERIAL PRIMARY KEY,
			type_id INTEGER NOT NULL REFERENCES equipment_types(id),
			product_name VARCHAR(255),
			asset_name VARCHAR(255) UNIQUE NOT NULL CHECK (length(trim(asset_name)) > 0),
			asset_tag VARCHAR(255),
			asset_serial_no VARCHAR(100) UNIQUE,
			bar_code VARCHAR(255),
			vendor_name VARCHAR(255),
			location VARCHAR(255),
			assigned_to_department VARCHAR(255),
			site VARCHAR(255),
			asset_no VARCHAR(255),
			budget VARCHAR(10),
			remark TEXT,
			req_no TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'ในคลัง'
				CHECK (status IN ('กำลังใช้งาน', 'ในคลัง', 'เสียหาย', 'เลิกใช้งาน')),
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS borrow_records (
			id SERIAL PRIMARY KEY,
			equipment_id INTEGER NOT NULL REFERENCES equipment(id),
			created_by_user_id INTEGER NOT NULL REFERENCES users(id),
			borrower_name VARCHAR(255) NOT NULL CHECK (length(trim(borrower_name)) > 0),
			approved_by INTEGER REFERENCES users(id),
			borrow_type VARCHAR(20) NOT NULL
				CHECK (borrow_type IN ('เบิกถาวร', 'เบิกชั่วคราว')),
			return_date DATE,
			reason TEXT,
			remark VARCHAR(255),
			status VARCHAR(30) NOT NULL DEFAULT 'รออนุมัติ'
				CHECK (status IN ('รออนุมัติ', 'อนุมัติ', 'ไม่อนุมัติ', 'รอตรวจรับคืน', 'คืนแล้ว', 'เกินกำหนดคืน')),
			requested_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			approved_at TIMESTAMPTZ,
			returned_at TIMESTAMPTZ,
			CONSTRAINT chk_return_date_matches_type CHECK (
				(borrow_type = 'เบิกชั่วคราว' AND return_date IS NOT NULL)
				OR (borrow_type = 'เบิกถาวร' AND return_date IS NULL)
			)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS one_active_borrow_per_equipment
			ON borrow_records (equipment_id)
			WHERE status IN ('รออนุมัติ', 'อนุมัติ', 'รอตรวจรับคืน', 'เกินกำหนดคืน')`,
		`CREATE TABLE IF NOT EXISTS repair_records (
			id SERIAL PRIMARY KEY,
			equipment_id INTEGER NOT NULL REFERENCES equipment(id),
			reporter_id INTEGER NOT NULL REFERENCES users(id),
			issue TEXT NOT NULL,
			remark TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'ส่งซ่อม'
				CHECK (status IN ('ส่งซ่อม', 'กำลังซ่อม', 'ซ่อมเสร็จ', 'ซ่อมไม่ได้')),
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS activity_logs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id INTEGER NOT NULL,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("execute migration statement: %w", err)
		}
	}

	return SeedEquipmentTypes(ctx, db)
}

// SeedEquipmentTypes inserts the baseline equipment types used by the MVP.
func SeedEquipmentTypes(ctx context.Context, db *sql.DB) error {
	seedValues := []struct {
		name         string
		minQuantity  int
	}{
		{name: "Laptop", minQuantity: 5},
		{name: "Monitor", minQuantity: 4},
		{name: "Printer", minQuantity: 2},
		{name: "Router", minQuantity: 3},
		{name: "Mobile", minQuantity: 6},
	}

	for _, item := range seedValues {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO equipment_types (name, min_quantity)
			VALUES ($1, $2)
			ON CONFLICT (name) DO NOTHING
		`, item.name, item.minQuantity); err != nil {
			return fmt.Errorf("seed equipment type %s: %w", item.name, err)
		}
	}

	return nil
}
