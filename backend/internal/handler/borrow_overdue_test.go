package handler

import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/wrnxc/inventory-service/internal/testutil"
    dbm "github.com/wrnxc/inventory-service/internal/db"
)

// T-18: overdue detection test (AC-6c)
func TestBorrowRequestOverdueDetection(t *testing.T) {
    gin.SetMode(gin.TestMode)

    newTestEnv := func(tb *testing.T) (*sql.DB, *gin.Engine) {
        tb.Helper()
        db := testutil.NewPostgres(tb)

        if err := dbm.Migrate(context.Background(), db); err != nil {
            tb.Fatalf("run migrations: %v", err)
        }

        seedAuthUsers(tb, db)

        // ensure equipment exists
        var typeID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM equipment_types WHERE name = $1 LIMIT 1`, "Notebook").Scan(&typeID); err != nil {
            tb.Fatalf("lookup equipment type: %v", err)
        }
        if _, err := db.ExecContext(context.Background(), `
            INSERT INTO equipment (type_id, product_name, asset_name, status)
            VALUES ($1, 'HP Test', 'NB-OD-1', 'ในคลัง')
        `, typeID); err != nil {
            tb.Fatalf("insert equipment: %v", err)
        }

        router := NewRouter(db)
        return db, router
    }

    t.Run("temporary approved borrow past return_date appears as overdue in query", func(t *testing.T) {
        db, router := newTestEnv(t)

        // create approved temporary borrow with past return_date
        var borrowID int
        var userID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = $1`, "active-user").Scan(&userID); err != nil {
            t.Fatalf("lookup active user id: %v", err)
        }

        if err := db.QueryRowContext(context.Background(), `
            INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, return_date, status, approved_by, approved_at)
            VALUES ($1, $2, $3, $4, $5, 'อนุมัติ', $2, now()) RETURNING id
        `, 1, userID, "OverdueUser", "เบิกชั่วคราว", "2000-01-01").Scan(&borrowID); err != nil {
            t.Fatalf("insert approved borrow: %v", err)
        }

        // set equipment to in use
        if _, err := db.ExecContext(context.Background(), `UPDATE equipment SET status = 'กำลังใช้งาน' WHERE id = $1`, 1); err != nil {
            t.Fatalf("update equipment status: %v", err)
        }

        // login as any user (list is available to all roles)
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        if login.Code != http.StatusOK {
            t.Fatalf("login failed: %d", login.Code)
        }
        cookie := firstSessionCookie(t, login)

        // query for overdue status
        req := httptest.NewRequest(http.MethodGet, "/api/v1/borrow-requests?status=เกินกำหนดคืน", nil)
        req.AddCookie(cookie)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Fatalf("expected 200 from list, got %d; body=%s", w.Code, w.Body.String())
        }

        var items []map[string]interface{}
        if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
            t.Fatalf("decode response: %v", err)
        }

        if len(items) == 0 {
            t.Fatalf("expected at least one overdue borrow, got 0")
        }

        // verify returned item is our borrow
        found := false
        for _, it := range items {
            if idf, ok := it["id"]; ok {
                switch v := idf.(type) {
                case float64:
                    if int(v) == borrowID {
                        found = true
                    }
                case int:
                    if v == borrowID {
                        found = true
                    }
                }
            }
        }
        if !found {
            t.Fatalf("expected borrow id %d in overdue list", borrowID)
        }
    })
}
