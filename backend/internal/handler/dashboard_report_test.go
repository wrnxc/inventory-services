package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	dbm "github.com/wrnxc/inventory-service/internal/db"
	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestDashboardReportsAndActivityLogs(t *testing.T) {
	newTestEnv := func(tb *testing.T) (*sql.DB, http.Handler, map[string]*http.Cookie) {
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
			tb.Fatalf("upsert admin user: %v", err)
		}

		var notebookTypeID int
		if err := db.QueryRowContext(context.Background(), `
			SELECT id FROM equipment_types WHERE name = 'Notebook'
		`).Scan(&notebookTypeID); err != nil {
			tb.Fatalf("lookup notebook type: %v", err)
		}
		var thinClientTypeID int
		if err := db.QueryRowContext(context.Background(), `
			SELECT id FROM equipment_types WHERE name = 'Thin Client'
		`).Scan(&thinClientTypeID); err != nil {
			tb.Fatalf("lookup thin client type: %v", err)
		}
		var cardReaderTypeID int
		if err := db.QueryRowContext(context.Background(), `
			SELECT id FROM equipment_types WHERE name = 'Card reader'
		`).Scan(&cardReaderTypeID); err != nil {
			tb.Fatalf("lookup card reader type: %v", err)
		}

		for _, item := range []struct {
			name   string
			status string
		}{
			{name: "DASH-STOCK-1", status: "ในคลัง"},
			{name: "DASH-USE-1", status: "กำลังใช้งาน"},
			{name: "DASH-DAMAGED-1", status: "เสียหาย"},
			{name: "DASH-RETIRED-1", status: "เลิกใช้งาน"},
		} {
			if _, err := db.ExecContext(context.Background(), `
				INSERT INTO equipment (type_id, product_name, asset_name, status)
				VALUES ($1, 'Dashboard Test Equipment', $2, $3)
			`, notebookTypeID, item.name, item.status); err != nil {
				tb.Fatalf("insert equipment %s: %v", item.name, err)
			}
		}
		for index := 1; index <= 3; index++ {
			if _, err := db.ExecContext(context.Background(), `
				INSERT INTO equipment (type_id, product_name, asset_name, status)
				VALUES ($1, 'Thin Client Boundary Test', $2, 'ในคลัง')
			`, thinClientTypeID, "DASH-EQUAL-"+strconv.Itoa(index)); err != nil {
				tb.Fatalf("insert equal-boundary equipment: %v", err)
			}
		}
		for index := 1; index <= 4; index++ {
			if _, err := db.ExecContext(context.Background(), `
				INSERT INTO equipment (type_id, product_name, asset_name, status)
				VALUES ($1, 'Card Reader Boundary Test', $2, 'ในคลัง')
			`, cardReaderTypeID, "DASH-ABOVE-"+strconv.Itoa(index)); err != nil {
				tb.Fatalf("insert above-boundary equipment: %v", err)
			}
		}

		var userID, otherUserID, adminID int
		if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = 'active-user'`).Scan(&userID); err != nil {
			tb.Fatalf("lookup active user: %v", err)
		}
		if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = 'user'`).Scan(&otherUserID); err != nil {
			tb.Fatalf("lookup other user: %v", err)
		}
		if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = 'admin'`).Scan(&adminID); err != nil {
			tb.Fatalf("lookup admin: %v", err)
		}

		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, status)
			VALUES
				(1, $1, 'User Pending', 'เบิกถาวร', 'รออนุมัติ'),
				(2, $2, 'Other Approved', 'เบิกถาวร', 'อนุมัติ'),
				(3, $1, 'User Return', 'เบิกถาวร', 'รอตรวจรับคืน')
		`, userID, otherUserID); err != nil {
			tb.Fatalf("insert dashboard borrow records: %v", err)
		}

		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO repair_records (equipment_id, reporter_id, issue, remark, status)
			VALUES (3, $1, 'Dashboard repair', 'normal', 'ส่งซ่อม')
		`, userID); err != nil {
			tb.Fatalf("insert dashboard repair record: %v", err)
		}

		oldTime := time.Now().Add(-48 * time.Hour)
		newTime := time.Now().Add(-1 * time.Hour)
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO activity_logs (user_id, action, resource_type, resource_id, metadata, created_at)
			VALUES
				($1, 'old_action', 'equipment', 1, '{}'::jsonb, $2),
				($1, 'new_action', 'equipment', 2, '{}'::jsonb, $3)
		`, adminID, oldTime, newTime); err != nil {
			tb.Fatalf("insert activity logs: %v", err)
		}

		router := NewRouter(db)
		cookies := make(map[string]*http.Cookie)
		for username, password := range map[string]string{
			"admin":       "admin-password",
			"active-user": "correct-password",
			"user":        "user123",
		} {
			login := performJSONRequest(tb, router, http.MethodPost, "/api/v1/login", map[string]string{
				"username": username,
				"password": password,
			})
			if login.Code != http.StatusOK {
				tb.Fatalf("login %s failed: %d %s", username, login.Code, strings.TrimSpace(login.Body.String()))
			}
			cookies[username] = firstSessionCookie(tb, login)
		}

		return db, router, cookies
	}

	get := func(t *testing.T, router http.Handler, path string, cookie *http.Cookie) string {
		t.Helper()
		response := performCookieRequest(t, router, http.MethodGet, path, cookie)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s: expected status %d, got %d; body=%s", path, http.StatusOK, response.Code, strings.TrimSpace(response.Body.String()))
		}
		return response.Body.String()
	}

	assertJSONNumber := func(t *testing.T, body, field string, want float64) {
		t.Helper()
		var payload map[string]any
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			t.Fatalf("decode JSON response: %v; body=%s", err, body)
		}
		got, ok := payload[field].(float64)
		if !ok || got != want {
			t.Fatalf("expected %s=%v, got %v; body=%s", field, want, payload[field], body)
		}
	}

	t.Run("dashboard returns global admin and system admin KPIs including minimum boundaries", func(t *testing.T) {
		_, router, cookies := newTestEnv(t)

		for _, username := range []string{"admin", "systemadmin"} {
			body := get(t, router, "/api/v1/dashboard", cookies[username])
			assertJSONNumber(t, body, "total_equipment", 11)
			assertJSONNumber(t, body, "equipment_in_use", 1)
			assertJSONNumber(t, body, "equipment_under_repair", 1)
			assertJSONNumber(t, body, "equipment_below_minimum", 1)
			if username == "admin" {
				assertJSONNumber(t, body, "pending_borrow_requests", 1)
				assertJSONNumber(t, body, "pending_return_confirmations", 1)
				assertJSONNumber(t, body, "pending_repair_tickets", 1)
			}
		}
	})

	t.Run("user dashboard is scoped to the authenticated user", func(t *testing.T) {
		_, router, cookies := newTestEnv(t)
		body := get(t, router, "/api/v1/dashboard", cookies["active-user"])

		assertJSONNumber(t, body, "pending_borrow_requests", 1)
		assertJSONNumber(t, body, "successful_borrows", 0)
		assertJSONNumber(t, body, "under_repair", 1)
		assertJSONNumber(t, body, "in_progress_requests", 1)
	})

	t.Run("reports require admin role and apply created_at date filters", func(t *testing.T) {
		_, router, cookies := newTestEnv(t)

		wideBody := get(t, router, "/api/v1/reports?type=stock", cookies["admin"])
		if !strings.Contains(wideBody, "DASH-STOCK-1") {
			t.Fatalf("expected stock report to include current stock equipment; body=%s", wideBody)
		}

		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		filteredBody := get(t, router, "/api/v1/reports?type=stock&from="+yesterday+"&to="+yesterday, cookies["admin"])
		if strings.Contains(filteredBody, "DASH-STOCK-1") {
			t.Fatalf("expected date filter to exclude today's equipment when range is yesterday; body=%s", filteredBody)
		}

		response := performCookieRequest(t, router, http.MethodGet, "/api/v1/reports?type=stock", cookies["active-user"])
		if response.Code != http.StatusForbidden {
			t.Fatalf("expected user report status %d, got %d; body=%s", http.StatusForbidden, response.Code, strings.TrimSpace(response.Body.String()))
		}
	})

	t.Run("activity logs are returned newest first", func(t *testing.T) {
		_, router, cookies := newTestEnv(t)
		body := get(t, router, "/api/v1/activity-logs", cookies["active-user"])

		var logs []struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal([]byte(body), &logs); err != nil {
			t.Fatalf("decode activity logs: %v; body=%s", err, body)
		}
		if len(logs) < 2 {
			t.Fatalf("expected at least two activity logs, got %d", len(logs))
		}
		if logs[0].Action != "new_action" || logs[1].Action != "old_action" {
			t.Fatalf("expected newest-first activity logs, got %#v", logs[:2])
		}
	})
}
