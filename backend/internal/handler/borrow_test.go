package handler

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/wrnxc/inventory-service/internal/testutil"
    dbm "github.com/wrnxc/inventory-service/internal/db"
)

// Tests for T-12: create/list/detail + duplicate protection and validations.
// This is a test-only task; it must not implement feature code.
func TestBorrowRequestCreateListDetail(t *testing.T) {
    gin.SetMode(gin.TestMode)
    // helper to create fresh DB + router per subtest for deterministic isolation
    newTestEnv := func(tb *testing.T) (*sql.DB, *gin.Engine) {
        tb.Helper()
        db := testutil.NewPostgres(tb)

        if err := dbm.Migrate(context.Background(), db); err != nil {
            tb.Fatalf("run migrations: %v", err)
        }

        // seed base users from seed.go
        seedAuthUsers(tb, db)

        // upsert admin with known password
        adminHash := mustHashPassword(tb, "admin-password")
        if _, err := db.ExecContext(context.Background(), `
            INSERT INTO users (username, password_hash, role, is_active)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash, role = EXCLUDED.role, is_active = EXCLUDED.is_active
        `, "admin", adminHash, "admin", true); err != nil {
            tb.Fatalf("upsert admin user: %v", err)
        }

        // ensure equipment exists
        var typeID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM equipment_types WHERE name = $1 LIMIT 1`, "Notebook").Scan(&typeID); err != nil {
            tb.Fatalf("lookup equipment type: %v", err)
        }
        if _, err := db.ExecContext(context.Background(), `
            INSERT INTO equipment (type_id, product_name, asset_name, status)
            VALUES ($1, 'HP Test', 'NB-TEST-1', 'ในคลัง')
        `, typeID); err != nil {
            tb.Fatalf("insert equipment: %v", err)
        }

        router := NewRouter(db)
        return db, router
    }

    t.Run("user can create borrow request (expected to fail until implementation)", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db
        // login as seeded active-user
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        if login.Code != http.StatusOK {
            t.Fatalf("login failed: %d", login.Code)
        }
        cookie := firstSessionCookie(t, login)

        payload := map[string]any{
            "equipment_id": 1,
            "borrow_type":  "เบิกชั่วคราว",
            "return_date":  time.Now().Add(24 * time.Hour).Format("2006-01-02"),
            "borrower_name": "สมชาย",
        }

        var body bytes.Buffer
        if err := json.NewEncoder(&body).Encode(payload); err != nil {
            t.Fatalf("encode payload: %v", err)
        }

        req := httptest.NewRequest(http.MethodPost, "/api/v1/borrow-requests", &body)
        req.AddCookie(cookie)
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        // Expect 201 when implemented; currently test will be red.
        if w.Code != http.StatusCreated {
            t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, w.Code, strings.TrimSpace(w.Body.String()))
        }
    })

    t.Run("borrower_name empty returns 422", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        cookie := firstSessionCookie(t, login)

        payload := map[string]any{
            "equipment_id": 1,
            "borrow_type":  "เบิกชั่วคราว",
            "return_date":  time.Now().Add(24 * time.Hour).Format("2006-01-02"),
            "borrower_name": "",
        }

        var body bytes.Buffer
        if err := json.NewEncoder(&body).Encode(payload); err != nil {
            t.Fatalf("encode payload: %v", err)
        }

        req := httptest.NewRequest(http.MethodPost, "/api/v1/borrow-requests", &body)
        req.AddCookie(cookie)
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusUnprocessableEntity {
            t.Fatalf("expected status %d for empty borrower_name, got %d; body=%s", http.StatusUnprocessableEntity, w.Code, strings.TrimSpace(w.Body.String()))
        }
    })

    t.Run("admin is forbidden to create borrow requests (403)", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db
        // login as admin
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "admin", "password": "admin-password"})
        if login.Code != http.StatusOK {
            t.Fatalf("admin login failed: %d", login.Code)
        }
        cookie := firstSessionCookie(t, login)

        payload := map[string]any{
            "equipment_id": 1,
            "borrow_type":  "เบิกถาวร",
            "borrower_name": "AdminTry",
        }

        var body bytes.Buffer
        if err := json.NewEncoder(&body).Encode(payload); err != nil {
            t.Fatalf("encode payload: %v", err)
        }

        req := httptest.NewRequest(http.MethodPost, "/api/v1/borrow-requests", &body)
        req.AddCookie(cookie)
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusForbidden {
            t.Fatalf("expected status %d for admin create, got %d; body=%s", http.StatusForbidden, w.Code, strings.TrimSpace(w.Body.String()))
        }
    })

    t.Run("equipment not found returns 404", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        cookie := firstSessionCookie(t, login)

        payload := map[string]any{
            "equipment_id": 9999,
            "borrow_type":  "เบิกถาวร",
            "borrower_name": "Someone",
        }

        var body bytes.Buffer
        if err := json.NewEncoder(&body).Encode(payload); err != nil {
            t.Fatalf("encode payload: %v", err)
        }

        req := httptest.NewRequest(http.MethodPost, "/api/v1/borrow-requests", &body)
        req.AddCookie(cookie)
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusNotFound {
            t.Fatalf("expected status %d for missing equipment, got %d; body=%s", http.StatusNotFound, w.Code, strings.TrimSpace(w.Body.String()))
        }
    })

    t.Run("invalid borrow_type returns 422", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        cookie := firstSessionCookie(t, login)

        payload := map[string]any{
            "equipment_id": 1,
            "borrow_type":  "INVALID",
            "borrower_name": "Someone",
        }

        var body bytes.Buffer
        if err := json.NewEncoder(&body).Encode(payload); err != nil {
            t.Fatalf("encode payload: %v", err)
        }

        req := httptest.NewRequest(http.MethodPost, "/api/v1/borrow-requests", &body)
        req.AddCookie(cookie)
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusUnprocessableEntity {
            t.Fatalf("expected status %d for invalid borrow_type, got %d; body=%s", http.StatusUnprocessableEntity, w.Code, strings.TrimSpace(w.Body.String()))
        }
    })

    t.Run("concurrent duplicate creation is prevented by DB unique index", func(t *testing.T) {
        // This test exercises the DB-level unique index directly.
        // Insert two conflicting rows concurrently and expect one to fail with duplicate key.
        var wg sync.WaitGroup
        errs := make([]error, 2)
        db, _ := newTestEnv(t)
        
        insert := func(i int) {
            defer wg.Done()
            _, err := db.ExecContext(context.Background(), `
                INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, status)
                VALUES ($1, $2, $3, $4, $5)
            `, 1, 1, "Concurrent", "เบิกถาวร", "รออนุมัติ")
            errs[i] = err
        }

        wg.Add(2)
        go insert(0)
        go insert(1)
        wg.Wait()

        // At least one insert should succeed and at least one should return an error.
        var successCount, errCount int
        for _, e := range errs {
            if e == nil {
                successCount++
            } else {
                errCount++
                if !strings.Contains(e.Error(), "duplicate key") && !strings.Contains(e.Error(), "unique") {
                    t.Fatalf("expected duplicate key error, got: %v", e)
                }
            }
        }

        if successCount != 1 || errCount != 1 {
            t.Fatalf("expected one success and one duplicate error, got success=%d err=%d; errs=%v", successCount, errCount, errs)
        }
    })

    t.Run("list filter parameters (status/equipment_id/borrower_name)", func(t *testing.T) {
        // This endpoint is not yet implemented; the test asserts expected behavior when implemented.
        db, router := newTestEnv(t)
        _ = db
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        cookie := firstSessionCookie(t, login)

        // Query with filters
        req := httptest.NewRequest(http.MethodGet, "/api/v1/borrow-requests?status=รออนุมัติ&equipment_id=1&borrower_name=Concurrent", nil)
        req.AddCookie(cookie)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Fatalf("expected status %d for list with filters, got %d; body=%s", http.StatusOK, w.Code, strings.TrimSpace(w.Body.String()))
        }
    })

    t.Run("activity_logs written on create (AC-9)", func(t *testing.T) {
        // After creating a borrow request, an activity log should be recorded.
        // This test performs a create then inspects activity_logs table.
        db, router := newTestEnv(t)
        _ = db
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        cookie := firstSessionCookie(t, login)

        payload := map[string]any{
            "equipment_id": 1,
            "borrow_type":  "เบิกถาวร",
            "borrower_name": "AC9",
        }

        var body bytes.Buffer
        if err := json.NewEncoder(&body).Encode(payload); err != nil {
            t.Fatalf("encode payload: %v", err)
        }

        req := httptest.NewRequest(http.MethodPost, "/api/v1/borrow-requests", &body)
        req.AddCookie(cookie)
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusCreated {
            t.Fatalf("expected create to return %d, got %d; body=%s", http.StatusCreated, w.Code, strings.TrimSpace(w.Body.String()))
        }

        var count int
        if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM activity_logs WHERE resource_type = $1`, "borrow").Scan(&count); err != nil {
            if err == sql.ErrNoRows {
                t.Fatalf("activity_logs query returned no rows: %v", err)
            }
            t.Fatalf("query activity_logs: %v", err)
        }

        if count == 0 {
            t.Fatalf("expected activity_logs to contain an entry for borrow create")
        }
    })
}
