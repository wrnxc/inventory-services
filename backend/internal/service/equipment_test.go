package service

import (
	"context"
	"testing"

	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestEquipmentCreateListAndPermissionRules(t *testing.T) {
	ctx := context.Background()
	dbConn := testutil.NewPostgres(t)

	for _, statement := range []string{
		`CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL
		)`,

		`CREATE TABLE equipment_types (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			min_quantity INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE equipment (
			id SERIAL PRIMARY KEY,

			type_id INTEGER NOT NULL
				REFERENCES equipment_types(id),

			product_name VARCHAR(255) NOT NULL
				CHECK (length(trim(product_name)) > 0),

			asset_name VARCHAR(255) UNIQUE NOT NULL
				CHECK (length(trim(asset_name)) > 0),

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
				CHECK (
					status IN (
						'กำลังใช้งาน',
						'ในคลัง',
						'เสียหาย',
						'เลิกใช้งาน'
					)
				),

			created_at TIMESTAMPTZ
				DEFAULT CURRENT_TIMESTAMP,

			updated_at TIMESTAMPTZ
				DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE activity_logs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id INTEGER NOT NULL,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	} {
		if _, err := dbConn.ExecContext(ctx, statement); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	// Seed admin
	if _, err := dbConn.ExecContext(
		ctx,
		`
			INSERT INTO users (username, role)
			VALUES ($1, $2)
		`,
		"admin",
		"admin",
	); err != nil {
		t.Fatalf("seed admin user: %v", err)
	}

	// Seed equipment type
	if _, err := dbConn.ExecContext(
		ctx,
		`
			INSERT INTO equipment_types (
				name,
				min_quantity
			)
			VALUES ($1, $2)
		`,
		"Notebook",
		0,
	); err != nil {
		t.Fatalf("seed equipment type: %v", err)
	}

	// --------------------------------------------------
	// Create
	// --------------------------------------------------

	created, err := CreateEquipment(
		ctx,
		dbConn,
		1,
		"admin",
		EquipmentInput{
			TypeID:        1,
			ProductName:   "HP ProBook",
			AssetName:     "NB-001",
			AssetSerialNo: "SN-001",

			AssetTag:             "HP ProBook 440 G7",
			BarCode:              "BAR-001",
			VendorName:           "HP",
			Location:             "Bangkok",
			AssignedToDepartment: "Service",
			Site:                 "HQ",
			AssetNo:              "ASSET-001",
			Budget:               "IT",
			Remark:               "test equipment",
			ReqNo:                "REQ-001",
		},
	)
	if err != nil {
		t.Fatalf("create equipment: %v", err)
	}

	if created == nil {
		t.Fatalf("expected created equipment")
	}

	if created.ProductName != "HP ProBook" {
		t.Fatalf(
			"expected product name %q, got %q",
			"HP ProBook",
			created.ProductName,
		)
	}

	if created.AssetName != "NB-001" {
		t.Fatalf(
			"expected asset name %q, got %q",
			"NB-001",
			created.AssetName,
		)
	}

	if created.Status != "ในคลัง" {
		t.Fatalf(
			"expected default status %q, got %q",
			"ในคลัง",
			created.Status,
		)
	}

	// --------------------------------------------------
	// List
	// --------------------------------------------------

	items, err := ListEquipment(ctx, dbConn)
	if err != nil {
		t.Fatalf("list equipment: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf(
			"expected one equipment row, got %d",
			len(items),
		)
	}

	if items[0].TypeName != "Notebook" {
		t.Fatalf(
			"expected type name %q, got %q",
			"Notebook",
			items[0].TypeName,
		)
	}

	// --------------------------------------------------
	// Update
	// --------------------------------------------------

	err = UpdateEquipment(
		ctx,
		dbConn,
		1,
		"admin",
		created.ID,
		EquipmentInput{
			TypeID:        1,
			ProductName:   "HP ProBook Updated",
			AssetName:     "NB-002",
			AssetSerialNo: "SN-002",

			AssetTag:             "HP ProBook 440 G8",
			BarCode:              "BAR-002",
			VendorName:           "HP",
			Location:             "Bangkok",
			AssignedToDepartment: "Service",
			Site:                 "HQ",
			AssetNo:              "ASSET-002",
			Budget:               "IT",
			Remark:               "updated equipment",
			ReqNo:                "REQ-002",
			Username:             "somchai.p",

			Status: "ในคลัง",
		},
	)
	if err != nil {
		t.Fatalf("update equipment: %v", err)
	}

	items, err = ListEquipment(ctx, dbConn)
	if err != nil {
		t.Fatalf("list equipment after update: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf(
			"expected one equipment after update, got %d",
			len(items),
		)
	}

	if items[0].ProductName != "HP ProBook Updated" {
		t.Fatalf(
			"expected updated product name, got %q",
			items[0].ProductName,
		)
	}

	if items[0].AssetName != "NB-002" {
		t.Fatalf(
			"expected updated asset name, got %q",
			items[0].AssetName,
		)
	}

	// --------------------------------------------------
	// Delete
	// --------------------------------------------------

	if err := DeleteEquipment(
		ctx,
		dbConn,
		1,
		"admin",
		created.ID,
	); err != nil {
		t.Fatalf("delete equipment: %v", err)
	}

	items, err = ListEquipment(ctx, dbConn)
	if err != nil {
		t.Fatalf("list equipment after delete: %v", err)
	}

	if len(items) != 0 {
		t.Fatalf(
			"expected no equipment after delete, got %d",
			len(items),
		)
	}

	// --------------------------------------------------
	// Activity log
	// --------------------------------------------------

	var count int

	if err := dbConn.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM activity_logs`,
	).Scan(&count); err != nil {
		t.Fatalf("count activity logs: %v", err)
	}

	if count == 0 {
		t.Fatalf(
			"expected activity logs to be written for mutations",
		)
	}
}
