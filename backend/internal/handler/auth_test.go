package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestLoginLogoutSessionBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutil.NewPostgres(t)

	seedAuthTables(t, db)
	seedAuthUsers(t, db)

	router := NewRouter(db)

	t.Run(
		"forged X-Session-User header without a session is rejected",
		func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/equipment",
				nil,
			)

			req.Header.Set(
				"X-Session-User",
				"admin",
			)

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusUnauthorized,
					w.Code,
				)
			}
		},
	)

	t.Run(
		"login success returns session cookie with strict flags",
		func(t *testing.T) {
			w := performJSONRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/login",
				map[string]string{
					"username": "active-user",
					"password": "correct-password",
				},
			)

			if w.Code != http.StatusOK {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusOK,
					w.Code,
				)
			}

			cookies := w.Result().Cookies()

			if len(cookies) == 0 {
				t.Fatalf(
					"expected login response to set a session cookie",
				)
			}

			cookie := cookies[0]

			if cookie.Name != sessionCookieName {
				t.Fatalf(
					"expected cookie %q, got %q",
					sessionCookieName,
					cookie.Name,
				)
			}

			if !cookie.HttpOnly {
				t.Fatalf(
					"expected session cookie to be HttpOnly",
				)
			}

			if !cookie.Secure {
				t.Fatalf(
					"expected session cookie to be Secure",
				)
			}

			if cookie.SameSite != http.SameSiteStrictMode {
				t.Fatalf(
					"expected SameSite=Strict, got %v",
					cookie.SameSite,
				)
			}

			if cookie.MaxAge <= 0 {
				t.Fatalf(
					"expected cookie to have a positive TTL, got %d",
					cookie.MaxAge,
				)
			}

			var expiresAt time.Time

			if err := db.QueryRow(
				`
					SELECT expires_at
					FROM sessions
					WHERE token = $1
				`,
				cookie.Value,
			).Scan(&expiresAt); err != nil {
				t.Fatalf(
					"read session expiry: %v",
					err,
				)
			}

			if !expiresAt.After(time.Now()) {
				t.Fatalf(
					"expected session expiry to be in the future, got %v",
					expiresAt,
				)
			}

			var body struct {
				User struct {
					ID       int    `json:"id"`
					Username string `json:"username"`
					Role     string `json:"role"`
				} `json:"user"`
			}

			if err := json.Unmarshal(
				w.Body.Bytes(),
				&body,
			); err != nil {
				t.Fatalf(
					"unmarshal login body: %v",
					err,
				)
			}

			if body.User.Username != "active-user" ||
				body.User.Role != "user" {
				t.Fatalf(
					"unexpected login user payload: %+v",
					body.User,
				)
			}

			// Protected endpoint must succeed
			// when the valid session cookie is supplied.
			protected := performCookieRequest(
				t,
				router,
				http.MethodGet,
				"/api/v1/equipment",
				cookie,
			)

			if protected.Code != http.StatusOK {
				t.Fatalf(
					"expected authenticated equipment request to succeed with %d, got %d; body=%s",
					http.StatusOK,
					protected.Code,
					protected.Body.String(),
				)
			}
		},
	)

	t.Run(
		"wrong password and missing user share the same 401 response",
		func(t *testing.T) {
			wrongPassword := performJSONRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/login",
				map[string]string{
					"username": "active-user",
					"password": "wrong-password",
				},
			)

			missingUser := performJSONRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/login",
				map[string]string{
					"username": "missing-user",
					"password": "wrong-password",
				},
			)

			if wrongPassword.Code != http.StatusUnauthorized {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusUnauthorized,
					wrongPassword.Code,
				)
			}

			if missingUser.Code != http.StatusUnauthorized {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusUnauthorized,
					missingUser.Code,
				)
			}

			expected :=
				`{"error":{"code":"INVALID_CREDENTIALS","message":"invalid username or password"}}`

			if got := strings.TrimSpace(
				wrongPassword.Body.String(),
			); got != expected {
				t.Fatalf(
					"expected wrong-password payload %q, got %q",
					expected,
					got,
				)
			}

			if got := strings.TrimSpace(
				missingUser.Body.String(),
			); got != expected {
				t.Fatalf(
					"expected missing-user payload %q, got %q",
					expected,
					got,
				)
			}

			if strings.TrimSpace(
				wrongPassword.Body.String(),
			) != strings.TrimSpace(
				missingUser.Body.String(),
			) {
				t.Fatalf(
					"expected identical 401 responses for wrong password and missing user",
				)
			}
		},
	)

	t.Run(
		"inactive account is forbidden",
		func(t *testing.T) {
			w := performJSONRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/login",
				map[string]string{
					"username": "inactive-user",
					"password": "correct-password",
				},
			)

			if w.Code != http.StatusForbidden {
				t.Fatalf(
					"expected status %d for inactive user, got %d",
					http.StatusForbidden,
					w.Code,
				)
			}
		},
	)

	t.Run(
		"logout invalidates the existing token",
		func(t *testing.T) {
			login := performJSONRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/login",
				map[string]string{
					"username": "active-user",
					"password": "correct-password",
				},
			)

			cookie := firstSessionCookie(
				t,
				login,
			)

			var countBefore int

			if err := db.QueryRow(
				`
					SELECT COUNT(*)
					FROM sessions
					WHERE token = $1
				`,
				cookie.Value,
			).Scan(&countBefore); err != nil {
				t.Fatalf(
					"count session before logout: %v",
					err,
				)
			}

			if countBefore != 1 {
				t.Fatalf(
					"expected one server-side session before logout, got %d",
					countBefore,
				)
			}

			logout := performCookieRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/logout",
				cookie,
			)

			if logout.Code != http.StatusOK {
				t.Fatalf(
					"expected logout status %d, got %d",
					http.StatusOK,
					logout.Code,
				)
			}

			var countAfter int

			if err := db.QueryRow(
				`
					SELECT COUNT(*)
					FROM sessions
					WHERE token = $1
				`,
				cookie.Value,
			).Scan(&countAfter); err != nil {
				t.Fatalf(
					"count session after logout: %v",
					err,
				)
			}

			if countAfter != 0 {
				t.Fatalf(
					"expected server-side session to be deleted after logout, got %d",
					countAfter,
				)
			}

			protected := performCookieRequest(
				t,
				router,
				http.MethodGet,
				"/api/v1/equipment",
				cookie,
			)

			if protected.Code != http.StatusUnauthorized {
				t.Fatalf(
					"expected protected route to reject invalidated token with %d, got %d",
					http.StatusUnauthorized,
					protected.Code,
				)
			}
		},
	)

	t.Run(
		"logout is idempotent without an active session",
		func(t *testing.T) {
			w := performRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/logout",
				nil,
				nil,
			)

			if w.Code != http.StatusOK {
				t.Fatalf(
					"expected idempotent logout status %d, got %d",
					http.StatusOK,
					w.Code,
				)
			}

			w = performRequest(
				t,
				router,
				http.MethodPost,
				"/api/v1/logout",
				nil,
				nil,
			)

			if w.Code != http.StatusOK {
				t.Fatalf(
					"expected repeated logout status %d, got %d",
					http.StatusOK,
					w.Code,
				)
			}
		},
	)
}

func seedAuthTables(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	statements := []string{
		`CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE
		)`,

		`CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL
				REFERENCES users(id),
			expires_at TIMESTAMPTZ NOT NULL
		)`,

		`CREATE TABLE equipment_types (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			min_quantity INTEGER NOT NULL DEFAULT 0
		)`,

		// Must follow the current equipment schema because
		// GET /equipment is used as the protected endpoint
		// in the authentication test.
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

			req_no TEXT,

			username VARCHAR(255),

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
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf(
				"create table: %v",
				err,
			)
		}
	}
}

func seedAuthUsers(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	activeHash := mustHashPassword(
		t,
		"correct-password",
	)

	inactiveHash := mustHashPassword(
		t,
		"correct-password",
	)

	if _, err := db.Exec(
		`
			INSERT INTO users (
				username,
				password_hash,
				role,
				is_active
			)
			VALUES ($1, $2, $3, $4)
		`,
		"active-user",
		activeHash,
		"user",
		true,
	); err != nil {
		t.Fatalf(
			"insert active user: %v",
			err,
		)
	}

	if _, err := db.Exec(
		`
			INSERT INTO users (
				username,
				password_hash,
				role,
				is_active
			)
			VALUES ($1, $2, $3, $4)
		`,
		"inactive-user",
		inactiveHash,
		"user",
		false,
	); err != nil {
		t.Fatalf(
			"insert inactive user: %v",
			err,
		)
	}
}

func mustHashPassword(
	t *testing.T,
	password string,
) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	return string(hash)
}

func performJSONRequest(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	payload any,
) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer

	if payload != nil {
		if err := json.NewEncoder(
			&body,
		).Encode(payload); err != nil {
			t.Fatalf(
				"encode request body: %v",
				err,
			)
		}
	}

	return performRequest(
		t,
		router,
		method,
		path,
		&body,
		nil,
	)
}

func performCookieRequest(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	cookie *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	return performRequest(
		t,
		router,
		method,
		path,
		nil,
		cookie,
	)
}

func performRequest(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	body *bytes.Buffer,
	cookie *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader

	if body != nil {
		reader = bytes.NewReader(
			body.Bytes(),
		)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(
		method,
		path,
		reader,
	)

	if cookie != nil {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()

	router.ServeHTTP(
		w,
		req,
	)

	return w
}

func firstSessionCookie(
	t *testing.T,
	w *httptest.ResponseRecorder,
) *http.Cookie {
	t.Helper()

	cookies := w.Result().Cookies()

	if len(cookies) == 0 {
		t.Fatalf(
			"expected a session cookie in response",
		)
	}

	return cookies[0]
}
