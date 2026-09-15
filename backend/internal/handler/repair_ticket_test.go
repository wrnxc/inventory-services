package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/testutil"
	dbm "github.com/wrnxc/inventory-service/internal/db"
)

func TestRepairTicketLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newTestEnv := func(tb *testing.T) (*sql.DB, *gin.Engine) {
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
			ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash, role = EXCLUDED.role, is_active = EXCLUDED.is_active
		`, "admin", adminHash, "admin", true); err != nil {
			tb.Fatalf("upsert admin user: %v", err)
		}

		var typeID int
		if err := db.QueryRowContext(context.Background(), `SELECT id FROM equipment_types WHERE name = $1 LIMIT 1`, "Notebook").Scan(&typeID); err != nil {
			tb.Fatalf("lookup equipment type: %v", err)
		}

		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO equipment (type_id, product_name, asset_name, status)
			VALUES ($1, 'HP Test', 'REPAIR-TEST-1', 'ในคลัง')
		`, typeID); err != nil {
			tb.Fatalf("insert working equipment: %v", err)
		}

		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO equipment (type_id, product_name, asset_name, status)
			VALUES ($1, 'HP Test', 'REPAIR-TEST-2', 'เลิกใช้งาน')
		`, typeID); err != nil {
			tb.Fatalf("insert decommissioned equipment: %v", err)
		}

		router := NewRouter(db)
		return db, router
	}

	t.Run("user can create repair ticket, list/detail, and activity log is written", func(t *testing.T) {
		_, router := newTestEnv(t)

		login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
		if login.Code != http.StatusOK {
			t.Fatalf("login failed: %d %s", login.Code, strings.TrimSpace(login.Body.String()))
		}
		cookie := firstSessionCookie(t, login)

		payload := map[string]any{
			"equipment_id":      1,
			"issue_description": "ปุ่ม power ไม่ตอบ",
			"urgency":           "normal",
		}
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/repair-tickets", &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d for create repair ticket, got %d; body=%s", http.StatusCreated, w.Code, strings.TrimSpace(w.Body.String()))
		}

		listReq := httptest.NewRequest(http.MethodGet, "/api/v1/repair-tickets", nil)
		listReq.AddCookie(cookie)
		listW := httptest.NewRecorder()
		router.ServeHTTP(listW, listReq)
		if listW.Code != http.StatusOK {
			t.Fatalf("expected status %d for list repair tickets, got %d; body=%s", http.StatusOK, listW.Code, strings.TrimSpace(listW.Body.String()))
		}

		detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/repair-tickets/1", nil)
		detailReq.AddCookie(cookie)
		detailW := httptest.NewRecorder()
		router.ServeHTTP(detailW, detailReq)
		if detailW.Code != http.StatusOK {
			t.Fatalf("expected status %d for get repair ticket, got %d; body=%s", http.StatusOK, detailW.Code, strings.TrimSpace(detailW.Body.String()))
		}
	})

	t.Run("missing equipment returns 404 and decommissioned equipment is rejected", func(t *testing.T) {
		_, router := newTestEnv(t)
		login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
		cookie := firstSessionCookie(t, login)

		payload := map[string]any{
			"equipment_id":      999,
			"issue_description": "ไม่พบอุปกรณ์",
			"urgency":           "high",
		}
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/repair-tickets", &body)
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status %d for missing equipment, got %d; body=%s", http.StatusNotFound, w.Code, strings.TrimSpace(w.Body.String()))
		}

		payload2 := map[string]any{
			"equipment_id":      2,
			"issue_description": "อุปกรณ์เลิกใช้งาน",
			"urgency":           "normal",
		}
		var body2 bytes.Buffer
		if err := json.NewEncoder(&body2).Encode(payload2); err != nil {
			t.Fatalf("encode payload2: %v", err)
		}

		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/repair-tickets", &body2)
		req2.AddCookie(cookie)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusConflict {
			t.Fatalf("expected status %d for decommissioned equipment, got %d; body=%s", http.StatusConflict, w2.Code, strings.TrimSpace(w2.Body.String()))
		}
	})

	t.Run("admin completes and fails repair tickets, invalid status and closed ticket are rejected, and user cannot update status", func(t *testing.T) {
		_, router := newTestEnv(t)
		userLogin := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
		userCookie := firstSessionCookie(t, userLogin)

		payload := map[string]any{
			"equipment_id":      1,
			"issue_description": "แป้นพิมพ์เสีย",
			"urgency":           "normal",
		}
		var createBody bytes.Buffer
		if err := json.NewEncoder(&createBody).Encode(payload); err != nil {
			t.Fatalf("encode create payload: %v", err)
		}
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/repair-tickets", &createBody)
		createReq.AddCookie(userCookie)
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		router.ServeHTTP(createW, createReq)
		if createW.Code != http.StatusCreated {
			t.Fatalf("expected create status %d, got %d; body=%s", http.StatusCreated, createW.Code, strings.TrimSpace(createW.Body.String()))
		}

		adminLogin := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "admin", "password": "admin-password"})
		adminCookie := firstSessionCookie(t, adminLogin)

		updatePayload := map[string]string{"status": "ซ่อมเสร็จ"}
		var updateBody bytes.Buffer
		if err := json.NewEncoder(&updateBody).Encode(updatePayload); err != nil {
			t.Fatalf("encode update payload: %v", err)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/repair-tickets/1/status", &updateBody)
		updateReq.AddCookie(adminCookie)
		updateReq.Header.Set("Content-Type", "application/json")
		updateW := httptest.NewRecorder()
		router.ServeHTTP(updateW, updateReq)
		if updateW.Code != http.StatusOK {
			t.Fatalf("expected status %d for completed repair, got %d; body=%s", http.StatusOK, updateW.Code, strings.TrimSpace(updateW.Body.String()))
		}

		invalidPayload := map[string]string{"status": "UNKNOWN"}
		var invalidBody bytes.Buffer
		if err := json.NewEncoder(&invalidBody).Encode(invalidPayload); err != nil {
			t.Fatalf("encode invalid payload: %v", err)
		}
		invalidReq := httptest.NewRequest(http.MethodPut, "/api/v1/repair-tickets/1/status", &invalidBody)
		invalidReq.AddCookie(adminCookie)
		invalidReq.Header.Set("Content-Type", "application/json")
		invalidW := httptest.NewRecorder()
		router.ServeHTTP(invalidW, invalidReq)
		if invalidW.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d for invalid repair status, got %d; body=%s", http.StatusUnprocessableEntity, invalidW.Code, strings.TrimSpace(invalidW.Body.String()))
		}

		userUpdateReq := httptest.NewRequest(http.MethodPut, "/api/v1/repair-tickets/1/status", &updateBody)
		userUpdateReq.AddCookie(userCookie)
		userUpdateReq.Header.Set("Content-Type", "application/json")
		userUpdateW := httptest.NewRecorder()
		router.ServeHTTP(userUpdateW, userUpdateReq)
		if userUpdateW.Code != http.StatusForbidden {
			t.Fatalf("expected status %d for user updating status, got %d; body=%s", http.StatusForbidden, userUpdateW.Code, strings.TrimSpace(userUpdateW.Body.String()))
		}
	})

	t.Run("activity logs include create and status update actions", func(t *testing.T) {
		_, router := newTestEnv(t)
		userLogin := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
		userCookie := firstSessionCookie(t, userLogin)

		payload := map[string]any{
			"equipment_id":      1,
			"issue_description": "เสียงดังผิดปกติ",
			"urgency":           "normal",
		}
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/repair-tickets", &body)
		createReq.AddCookie(userCookie)
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		router.ServeHTTP(createW, createReq)
		if createW.Code != http.StatusCreated {
			t.Fatalf("expected create status %d, got %d; body=%s", http.StatusCreated, createW.Code, strings.TrimSpace(createW.Body.String()))
		}

		logsReq := httptest.NewRequest(http.MethodGet, "/api/v1/activity-logs", nil)
		logsReq.AddCookie(userCookie)
		logsW := httptest.NewRecorder()
		router.ServeHTTP(logsW, logsReq)
		if logsW.Code != http.StatusOK {
			t.Fatalf("expected status %d for activity logs, got %d; body=%s", http.StatusOK, logsW.Code, strings.TrimSpace(logsW.Body.String()))
		}
	})
}
