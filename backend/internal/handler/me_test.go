package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/testutil"
)

func TestMeEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutil.NewPostgres(t)
	seedAuthTables(t, db)
	seedAuthUsers(t, db)

	router := NewRouter(db)

	// login จริงเพื่อรับ session cookie
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

	if login.Code != http.StatusOK {
		t.Fatalf("login failed: %d", login.Code)
	}

	cookie := firstSessionCookie(t, login)

	t.Run("valid session returns current user", func(t *testing.T) {
		w := performCookieRequest(
			t,
			router,
			http.MethodGet,
			"/api/v1/me",
			cookie,
		)

		if w.Code != http.StatusOK {
			t.Fatalf(
				"expected 200 for /me with valid session, got %d",
				w.Code,
			)
		}

		var response struct {
			User struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				Role     string `json:"role"`
			} `json:"user"`
		}

		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode /me response: %v", err)
		}

		if response.User.ID == 0 {
			t.Fatal("expected user id to be populated")
		}

		if response.User.Username != "active-user" {
			t.Fatalf(
				"expected username active-user, got %q",
				response.User.Username,
			)
		}

		if response.User.Role != "user" {
			t.Fatalf(
				"expected role user, got %q",
				response.User.Role,
			)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		w := performRequest(
			t,
			router,
			http.MethodGet,
			"/api/v1/me",
			nil,
			nil,
		)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf(
				"expected 401 for /me without session, got %d",
				w.Code,
			)
		}
	})

	t.Run("invalid session returns 401", func(t *testing.T) {
		fake := &http.Cookie{
			Name:  sessionCookieName,
			Value: "invalid-token",
		}

		w := performCookieRequest(
			t,
			router,
			http.MethodGet,
			"/api/v1/me",
			fake,
		)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf(
				"expected 401 for /me with invalid session, got %d",
				w.Code,
			)
		}
	})
}