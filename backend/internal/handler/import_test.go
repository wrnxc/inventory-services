package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	dbm "github.com/wrnxc/inventory-service/internal/db"
	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestImportEquipment(t *testing.T) {
	newTestEnv := func(tb *testing.T) (*sql.DB, http.Handler) {
		tb.Helper()

		db := testutil.NewPostgres(tb)
		if err := dbm.Migrate(context.Background(), db); err != nil {
			tb.Fatalf("run migrations: %v", err)
		}

		seedAuthUsers(tb, db)
		adminHash := mustHashPassword(tb, "admin-password")
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO users (username, password_hash, role, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (username) DO UPDATE SET
				password_hash = EXCLUDED.password_hash,
				role = EXCLUDED.role,
				is_active = EXCLUDED.is_active
		`, "admin", adminHash, "admin", true); err != nil {
			tb.Fatalf("insert admin user: %v", err)
		}

		return db, NewRouter(db)
	}

	loginAsAdmin := func(t *testing.T, router http.Handler) *http.Cookie {
		t.Helper()
		login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{
			"username": "admin",
			"password": "admin-password",
		})
		if login.Code != http.StatusOK {
			t.Fatalf("login failed: %d %s", login.Code, strings.TrimSpace(login.Body.String()))
		}
		return firstSessionCookie(t, login)
	}

	buildCSV := func(t *testing.T, rows [][]string) (*bytes.Buffer, string) {
		t.Helper()

		var content bytes.Buffer
		writer := csv.NewWriter(&content)
		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				t.Fatalf("write CSV row: %v", err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			t.Fatalf("flush CSV: %v", err)
		}
		return &content, "equipment.csv"
	}

	postImport := func(t *testing.T, router http.Handler, cookie *http.Cookie, content *bytes.Buffer, filename string) *httptest.ResponseRecorder {
		t.Helper()

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create multipart file: %v", err)
		}
		if _, err := part.Write(content.Bytes()); err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/import", &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	t.Run("imports valid rows, reports validation and duplicate failures, and writes an activity log", func(t *testing.T) {
		db, router := newTestEnv(t)
		cookie := loginAsAdmin(t, router)

		content, filename := buildCSV(t, [][]string{
			{"product_name", "asset_name", "asset_serial_no", "type_id"},
			{"Notebook", "IMPORT-001", "IMPORT-SN-001", "1"},
			{"Notebook", "IMPORT-002", "IMPORT-SN-002", "1"},
			{"", "IMPORT-003", "IMPORT-SN-003", "1"},
			{"Notebook", "IMPORT-001", "IMPORT-SN-004", "1"},
		})

		response := postImport(t, router, cookie, content, filename)
		if response.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, response.Code, strings.TrimSpace(response.Body.String()))
		}

		var result struct {
			Total   int `json:"total"`
			Success int `json:"success"`
			Failed  int `json:"failed"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode import result: %v", err)
		}
		if result.Total != 4 || result.Success != 2 || result.Failed != 2 {
			t.Fatalf("unexpected import result: %+v", result)
		}

		var importedCount int
		if err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM equipment WHERE asset_name LIKE 'IMPORT-%'
		`).Scan(&importedCount); err != nil {
			t.Fatalf("count imported equipment: %v", err)
		}
		if importedCount != 2 {
			t.Fatalf("expected 2 imported equipment rows, got %d", importedCount)
		}

		var activityCount int
		if err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*)
			FROM activity_logs
			WHERE action = 'import_equipment' AND resource_type = 'equipment'
		`).Scan(&activityCount); err != nil {
			t.Fatalf("count import activity logs: %v", err)
		}
		if activityCount != 1 {
			t.Fatalf("expected one import activity log, got %d", activityCount)
		}
	})

	t.Run("requires the CSV header row", func(t *testing.T) {
		_, router := newTestEnv(t)
		cookie := loginAsAdmin(t, router)
		content, filename := buildCSV(t, [][]string{
			{"Notebook", "NO-HEADER", "NO-HEADER-SN", "1"},
		})

		response := postImport(t, router, cookie, content, filename)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d for missing header, got %d; body=%s", http.StatusUnprocessableEntity, response.Code, strings.TrimSpace(response.Body.String()))
		}
	})

	t.Run("rejects invalid file input", func(t *testing.T) {
		_, router := newTestEnv(t)
		cookie := loginAsAdmin(t, router)

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("file", "not-a-file"); err != nil {
			t.Fatalf("write invalid form field: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/import", &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)

		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d for missing uploaded file, got %d; body=%s", http.StatusUnprocessableEntity, response.Code, strings.TrimSpace(response.Body.String()))
		}
	})

	t.Run("allows only admin roles to import", func(t *testing.T) {
		_, router := newTestEnv(t)
		login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{
			"username": "active-user",
			"password": "correct-password",
		})
		if login.Code != http.StatusOK {
			t.Fatalf("login failed: %d %s", login.Code, strings.TrimSpace(login.Body.String()))
		}

		content, filename := buildCSV(t, [][]string{
			{"product_name", "asset_name", "asset_serial_no", "type_id"},
			{"Notebook", "USER-IMPORT-001", "USER-IMPORT-SN-001", strconv.Itoa(1)},
		})
		response := postImport(t, router, firstSessionCookie(t, login), content, filename)
		if response.Code != http.StatusForbidden {
			t.Fatalf("expected status %d for user import, got %d; body=%s", http.StatusForbidden, response.Code, strings.TrimSpace(response.Body.String()))
		}
	})
}
