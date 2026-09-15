package handler

import (
    "context"
    "database/sql"
    "net/http"
    "strconv"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/wrnxc/inventory-service/internal/testutil"
    dbm "github.com/wrnxc/inventory-service/internal/db"
)

// T-16: return/confirm-return tests (AC-6, AC-6b, AC-9)
func TestBorrowRequestReturnConfirm(t *testing.T) {
    gin.SetMode(gin.TestMode)

    newTestEnv := func(tb *testing.T) (*sql.DB, *gin.Engine) {
        tb.Helper()
        db := testutil.NewPostgres(tb)

        if err := dbm.Migrate(context.Background(), db); err != nil {
            tb.Fatalf("run migrations: %v", err)
        }

        seedAuthUsers(tb, db)

        // upsert admin with known password for tests
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
            VALUES ($1, 'HP Test', 'NB-RET-1', 'ในคลัง')
        `, typeID); err != nil {
            tb.Fatalf("insert equipment: %v", err)
        }

        router := NewRouter(db)
        return db, router
    }

    t.Run("user can request return for approved borrow; user forbidden for others; confirm by admin", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db

        // create approved borrow (simulate via DB updates): insert and set status=อนุมัติ
        var borrowID int
        var userID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = $1`, "active-user").Scan(&userID); err != nil {
            t.Fatalf("lookup active user id: %v", err)
        }

        if err := db.QueryRowContext(context.Background(), `
            INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, return_date, status, approved_by, approved_at)
            VALUES ($1, $2, $3, $4, $5, 'อนุมัติ', $2, now()) RETURNING id
        `, 1, userID, "ReturnMe", "เบิกชั่วคราว", "2099-01-01").Scan(&borrowID); err != nil {
            t.Fatalf("insert approved borrow: %v", err)
        }

        // Set equipment status to 'กำลังใช้งาน'
        if _, err := db.ExecContext(context.Background(), `UPDATE equipment SET status = 'กำลังใช้งาน' WHERE id = $1`, 1); err != nil {
            t.Fatalf("update equipment status: %v", err)
        }

        // login as the same user
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        if login.Code != http.StatusOK {
            t.Fatalf("user login failed: %d", login.Code)
        }
        userCookie := firstSessionCookie(t, login)

        // user posts return
        ret := performCookieRequest(t, router, http.MethodPost, "/api/v1/borrow-requests/"+strconv.Itoa(borrowID)+"/return", userCookie)
        if ret.Code != http.StatusOK {
            t.Fatalf("expected return 200, got %d; body=%s", ret.Code, ret.Body.String())
        }

        // user tries to confirm -> should be forbidden
        userConfirm := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(borrowID)+"/confirm-return", userCookie)
        if userConfirm.Code != http.StatusForbidden {
            t.Fatalf("expected user confirm 403, got %d; body=%s", userConfirm.Code, userConfirm.Body.String())
        }

        // login as admin and confirm
        adminLogin := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "admin", "password": "admin-password"})
        if adminLogin.Code != http.StatusOK {
            t.Fatalf("admin login failed: %d", adminLogin.Code)
        }
        adminCookie := firstSessionCookie(t, adminLogin)

        confirm := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(borrowID)+"/confirm-return", adminCookie)
        if confirm.Code != http.StatusOK {
            t.Fatalf("expected confirm 200, got %d; body=%s", confirm.Code, confirm.Body.String())
        }

        // verify borrow status is 'คืนแล้ว' and equipment back to 'ในคลัง'
        var status string
        if err := db.QueryRowContext(context.Background(), `SELECT status FROM borrow_records WHERE id = $1`, borrowID).Scan(&status); err != nil {
            t.Fatalf("query borrow status: %v", err)
        }
        if status != "คืนแล้ว" {
            t.Fatalf("expected borrow status 'คืนแล้ว', got %s", status)
        }

        var eqStatus string
        if err := db.QueryRowContext(context.Background(), `SELECT status FROM equipment WHERE id = $1`, 1).Scan(&eqStatus); err != nil {
            t.Fatalf("query equipment status: %v", err)
        }
        if eqStatus != "ในคลัง" {
            t.Fatalf("expected equipment status 'ในคลัง', got %s", eqStatus)
        }
    })

    t.Run("user cannot return other user's borrow (403) and invalid state returns 409", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db

        // create approved borrow for active-user
        var borrowID int
        var userID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = $1`, "active-user").Scan(&userID); err != nil {
            t.Fatalf("lookup active user id: %v", err)
        }
        if err := db.QueryRowContext(context.Background(), `
            INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, status, approved_by, approved_at)
            VALUES ($1, $2, $3, $4, 'อนุมัติ', $2, now()) RETURNING id
        `, 1, userID, "OtherBorrow", "เบิกถาวร").Scan(&borrowID); err != nil {
            t.Fatalf("insert approved borrow: %v", err)
        }
        if _, err := db.ExecContext(context.Background(), `UPDATE equipment SET status = 'กำลังใช้งาน' WHERE id = $1`, 1); err != nil {
            t.Fatalf("update equipment status: %v", err)
        }

        // create a different user and login
        // insert user directly
        if _, err := db.ExecContext(context.Background(), `INSERT INTO users (username, password_hash, role, is_active) VALUES ($1, $2, $3, $4)`, "other-user", mustHashPassword(t, "p"), "user", true); err != nil {
            t.Fatalf("insert other user: %v", err)
        }
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "other-user", "password": "p"})
        if login.Code != http.StatusOK {
            t.Fatalf("other-user login failed: %d", login.Code)
        }
        cookie := firstSessionCookie(t, login)

        // attempt to return someone else's borrow
        ret := performCookieRequest(t, router, http.MethodPost, "/api/v1/borrow-requests/"+strconv.Itoa(borrowID)+"/return", cookie)
        if ret.Code != http.StatusForbidden {
            t.Fatalf("expected forbidden when returning other's borrow, got %d; body=%s", ret.Code, ret.Body.String())
        }

        // attempt to return non-returnable state (e.g., already returned)
        // mark borrow as 'คืนแล้ว'
        if _, err := db.ExecContext(context.Background(), `UPDATE borrow_records SET status='คืนแล้ว' WHERE id = $1`, borrowID); err != nil {
            t.Fatalf("mark returned: %v", err)
        }
        // original creator tries to return again
        login2 := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        if login2.Code != http.StatusOK {
            t.Fatalf("active-user re-login failed: %d", login2.Code)
        }
        cookie2 := firstSessionCookie(t, login2)
        ret2 := performCookieRequest(t, router, http.MethodPost, "/api/v1/borrow-requests/"+strconv.Itoa(borrowID)+"/return", cookie2)
        if ret2.Code != http.StatusConflict {
            t.Fatalf("expected 409 when returning non-returnable, got %d; body=%s", ret2.Code, ret2.Body.String())
        }
    })
}
