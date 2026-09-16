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
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
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
			password_hash TEXT NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			role TEXT NOT NULL CHECK (role IN ('system_admin', 'admin', 'user'))
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE`,
		`CREATE TABLE IF NOT EXISTS equipment_types (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
			min_quantity INTEGER NOT NULL DEFAULT 0 CHECK (min_quantity >= 0)
		)`,
		`CREATE TABLE IF NOT EXISTS equipment (
			id SERIAL PRIMARY KEY,
			type_id INTEGER NOT NULL REFERENCES equipment_types(id),
			product_name VARCHAR(255) NOT NULL CHECK (length(trim(product_name)) > 0),
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
			username VARCHAR(255),
			req_no TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'ในคลัง'
				CHECK (status IN ('กำลังใช้งาน', 'ในคลัง', 'เสียหาย', 'เลิกใช้งาน')),
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		// CREATE TABLE IF NOT EXISTS does not modify an existing equipment table.
		// These statements bring an existing database in line with the current schema.
		`ALTER TABLE equipment ADD COLUMN IF NOT EXISTS product_name VARCHAR(255)`,
		`ALTER TABLE equipment
			ALTER COLUMN product_name SET NOT NULL`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'equipment_product_name_not_blank'
				  AND conrelid = 'equipment'::regclass
			) THEN
				ALTER TABLE equipment
				ADD CONSTRAINT equipment_product_name_not_blank
				CHECK (length(trim(product_name)) > 0);
			END IF;
		END
		$$`,
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
		`CREATE TABLE IF NOT EXISTS import_history (
			id SERIAL PRIMARY KEY,
			file_name VARCHAR(255) NOT NULL,
			total INTEGER NOT NULL DEFAULT 0 CHECK (total >= 0),
			success INTEGER NOT NULL DEFAULT 0 CHECK (success >= 0),
			failed INTEGER NOT NULL DEFAULT 0 CHECK (failed >= 0),
			imported_by INTEGER NOT NULL REFERENCES users(id),
			imported_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE import_history
    		ADD COLUMN IF NOT EXISTS duplicate INTEGER NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("execute migration statement: %w", err)
		}
	}

	if err := SeedEquipmentTypes(ctx, db); err != nil {
		return err
	}
	if err := SeedUsers(ctx, db); err != nil {
		return err
	}
	return nil
}
