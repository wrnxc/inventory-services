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

// T-14: approve/reject tests (AC-5, AC-5b, AC-9)
func TestBorrowRequestApproveReject(t *testing.T) {
    gin.SetMode(gin.TestMode)

    newTestEnv := func(tb *testing.T) (*sql.DB, *gin.Engine) {
        tb.Helper()
        db := testutil.NewPostgres(tb)

        if err := dbm.Migrate(context.Background(), db); err != nil {
            tb.Fatalf("run migrations: %v", err)
        }

        seedAuthUsers(tb, db)

        // upsert admin
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
            VALUES ($1, 'HP Test', 'NB-APP-1', 'ในคลัง')
        `, typeID); err != nil {
            tb.Fatalf("insert equipment: %v", err)
        }

        router := NewRouter(db)
        return db, router
    }

    t.Run("admin can approve and activity_log written; duplicate approve -> 409; user forbidden", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db

        // create a borrow request directly in DB (pending)
        var createdID int
        var userID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = $1`, "active-user").Scan(&userID); err != nil {
            t.Fatalf("lookup active user id: %v", err)
        }
        if err := db.QueryRowContext(context.Background(), `
            INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, status)
            VALUES ($1, $2, $3, $4, $5) RETURNING id
        `, 1, userID, "Worker", "เบิกถาวร", "รออนุมัติ").Scan(&createdID); err != nil {
            t.Fatalf("insert borrow record: %v", err)
        }

        // login as admin
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "admin", "password": "admin-password"})
        if login.Code != http.StatusOK {
            t.Fatalf("admin login failed: %d", login.Code)
        }
        adminCookie := firstSessionCookie(t, login)

        // approve
        approve := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(createdID)+"/approve", adminCookie)
        if approve.Code != http.StatusOK {
            t.Fatalf("expected approve 200, got %d; body=%s", approve.Code, approve.Body.String())
        }

        // second approve should return 409
        approve2 := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(createdID)+"/approve", adminCookie)
        if approve2.Code != http.StatusConflict {
            t.Fatalf("expected second approve 409, got %d; body=%s", approve2.Code, approve2.Body.String())
        }

        // verify activity_logs contains entry for this borrow
        var count int
        if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM activity_logs WHERE resource_type = $1 AND resource_id = $2`, "borrow", createdID).Scan(&count); err != nil {
            t.Fatalf("query activity logs: %v", err)
        }
        if count == 0 {
            t.Fatalf("expected activity log entry for approve action")
        }

        // login as regular user and try approve -> 403
        userLogin := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        if userLogin.Code != http.StatusOK {
            t.Fatalf("user login failed: %d", userLogin.Code)
        }
        userCookie := firstSessionCookie(t, userLogin)

        userApprove := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(createdID)+"/approve", userCookie)
        if userApprove.Code != http.StatusForbidden {
            t.Fatalf("expected user approve 403, got %d; body=%s", userApprove.Code, userApprove.Body.String())
        }
    })

    t.Run("admin can reject pending; rejecting non-pending -> 409; user forbidden", func(t *testing.T) {
        db, router := newTestEnv(t)
        _ = db

        // create pending borrow
        var createdID int
        var userID int
        if err := db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = $1`, "active-user").Scan(&userID); err != nil {
            t.Fatalf("lookup active user id: %v", err)
        }
        if err := db.QueryRowContext(context.Background(), `
            INSERT INTO borrow_records (equipment_id, created_by_user_id, borrower_name, borrow_type, status)
            VALUES ($1, $2, $3, $4, $5) RETURNING id
        `, 1, userID, "RejectMe", "เบิกถาวร", "รออนุมัติ").Scan(&createdID); err != nil {
            t.Fatalf("insert borrow record: %v", err)
        }

        // admin login
        login := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "admin", "password": "admin-password"})
        if login.Code != http.StatusOK {
            t.Fatalf("admin login failed: %d", login.Code)
        }
        adminCookie := firstSessionCookie(t, login)

        // reject
        reject := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(createdID)+"/reject", adminCookie)
        if reject.Code != http.StatusOK {
            t.Fatalf("expected reject 200, got %d; body=%s", reject.Code, reject.Body.String())
        }

        // second reject should return 409
        reject2 := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(createdID)+"/reject", adminCookie)
        if reject2.Code != http.StatusConflict {
            t.Fatalf("expected second reject 409, got %d; body=%s", reject2.Code, reject2.Body.String())
        }

        // verify activity log
        var count int
        if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM activity_logs WHERE resource_type = $1 AND resource_id = $2`, "borrow", createdID).Scan(&count); err != nil {
            t.Fatalf("query activity logs: %v", err)
        }
        if count == 0 {
            t.Fatalf("expected activity log entry for reject action")
        }

        // user forbidden to reject
        userLogin := performJSONRequest(t, router, http.MethodPost, "/api/v1/login", map[string]string{"username": "active-user", "password": "correct-password"})
        if userLogin.Code != http.StatusOK {
            t.Fatalf("user login failed: %d", userLogin.Code)
        }
        userCookie := firstSessionCookie(t, userLogin)

        userReject := performCookieRequest(t, router, http.MethodPut, "/api/v1/borrow-requests/"+strconv.Itoa(createdID)+"/reject", userCookie)
        if userReject.Code != http.StatusForbidden {
            t.Fatalf("expected user reject 403, got %d; body=%s", userReject.Code, userReject.Body.String())
        }
    })
}
