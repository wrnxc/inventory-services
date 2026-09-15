package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestDatabaseSchemaContract(t *testing.T) {
	dbConn := testutil.NewPostgres(t)

	if err := Migrate(context.Background(), dbConn); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	checkRequiredTable(t, dbConn, "equipment_types")
	checkRequiredTable(t, dbConn, "equipment")
	checkRequiredTable(t, dbConn, "borrow_records")
	checkRequiredTable(t, dbConn, "repair_records")
	checkRequiredTable(t, dbConn, "activity_logs")

	assertColumnExists(t, dbConn, "equipment", "status")
	assertColumnExists(t, dbConn, "borrow_records", "created_by_user_id")
	assertColumnExists(t, dbConn, "borrow_records", "borrower_name")
	assertColumnExists(t, dbConn, "activity_logs", "resource_type")
	assertColumnExists(t, dbConn, "activity_logs", "resource_id")
	assertColumnExists(t, dbConn, "activity_logs", "metadata")

	assertCheckConstraintContains(t, dbConn, "equipment", "status", "กำลังใช้งาน")
	assertCheckConstraintContains(t, dbConn, "equipment", "status", "เลิกใช้งาน")
	assertCheckConstraintContains(t, dbConn, "borrow_records", "status", "คืนแล้ว")
	assertCheckConstraintContains(t, dbConn, "borrow_records", "status", "เกินกำหนดคืน")

	assertColumnNotNullable(t, dbConn, "borrow_records", "borrower_name")
	assertForeignKeyExists(t, dbConn, "borrow_records", "created_by_user_id", "users")
	assertIndexExists(t, dbConn, "one_active_borrow_per_equipment")

	assertSeededEquipmentTypes(t, dbConn)
}

func checkRequiredTable(t *testing.T, dbConn *sql.DB, tableName string) {
	t.Helper()
	var exists bool
	err := dbConn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("check table %s existence: %v", tableName, err)
	}
	if !exists {
		t.Fatalf("expected table %s to exist", tableName)
	}
}

func assertColumnExists(t *testing.T, dbConn *sql.DB, tableName, columnName string) {
	t.Helper()
	var exists bool
	err := dbConn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2)`, tableName, columnName).Scan(&exists)
	if err != nil {
		t.Fatalf("check column %s.%s: %v", tableName, columnName, err)
	}
	if !exists {
		t.Fatalf("expected column %s.%s to exist", tableName, columnName)
	}
}

func assertColumnNotNullable(t *testing.T, dbConn *sql.DB, tableName, columnName string) {
	t.Helper()
	var isNullable string
	err := dbConn.QueryRow(`SELECT is_nullable FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`, tableName, columnName).Scan(&isNullable)
	if err != nil {
		t.Fatalf("check nullability for %s.%s: %v", tableName, columnName, err)
	}
	if isNullable != "NO" {
		t.Fatalf("expected column %s.%s to be NOT NULL, got %s", tableName, columnName, isNullable)
	}
}

func assertForeignKeyExists(t *testing.T, dbConn *sql.DB, tableName, columnName, refTable string) {
	t.Helper()
	var exists bool
	err := dbConn.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint c
			JOIN pg_class rel ON rel.oid = c.conrelid
			JOIN pg_namespace relns ON relns.oid = rel.relnamespace
			JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
			JOIN pg_class refrel ON refrel.oid = c.confrelid
			JOIN pg_namespace refns ON refns.oid = refrel.relnamespace
			WHERE relns.nspname = 'public'
			AND rel.relname = $1
			AND a.attname = $2
			AND refns.nspname = 'public'
			AND refrel.relname = $3
			AND c.contype = 'f'
		)
	`, tableName, columnName, refTable).Scan(&exists)
	if err != nil {
		t.Fatalf("check foreign key for %s.%s: %v", tableName, columnName, err)
	}
	if !exists {
		t.Fatalf("expected foreign key from %s.%s to %s", tableName, columnName, refTable)
	}
}

func assertCheckConstraintContains(t *testing.T, dbConn *sql.DB, tableName, columnName, expectedValue string) {
	t.Helper()
	var def string
	err := dbConn.QueryRow(`
		SELECT cc.check_clause
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.table_schema
		JOIN information_schema.check_constraints cc
			ON tc.constraint_name = cc.constraint_name
		WHERE tc.table_schema = 'public'
		AND tc.table_name = $1
		AND ccu.column_name = $2
		AND tc.constraint_type = 'CHECK'
	`, tableName, columnName).Scan(&def)
	if err != nil {
		t.Fatalf("read check constraint for %s.%s: %v", tableName, columnName, err)
	}
	if !strings.Contains(def, expectedValue) {
		t.Fatalf("expected check constraint for %s.%s to include %q, got %q", tableName, columnName, expectedValue, def)
	}
}

func assertIndexExists(t *testing.T, dbConn *sql.DB, indexName string) {
	t.Helper()
	var exists bool
	err := dbConn.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1)`, indexName).Scan(&exists)
	if err != nil {
		t.Fatalf("check index %s: %v", indexName, err)
	}
	if !exists {
		t.Fatalf("expected index %s to exist", indexName)
	}
}

func assertSeededEquipmentTypes(t *testing.T, dbConn *sql.DB) {
	t.Helper()
	var count int
	err := dbConn.QueryRow(`SELECT COUNT(*) FROM equipment_types`).Scan(&count)
	if err != nil {
		t.Fatalf("count seeded equipment types: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected equipment_types to be seeded")
	}
}
