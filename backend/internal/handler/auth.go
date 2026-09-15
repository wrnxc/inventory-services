package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"strconv"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthUser struct {
	ID       int
	Username string
	Role     string
}

const (
	authUserContextKey = "authUser"
	sessionCookieName  = "session_token"
)

var sessionTTL = 24 * time.Hour

func ConfigureSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		sessionTTL = ttl
	}
}

func SessionTTLFromEnv() time.Duration {
	value := os.Getenv("SESSION_TTL_HOURS")
	if value == "" {
		return sessionTTL
	}
	hours, err := strconv.Atoi(value)
	if err != nil || hours <= 0 {
		return sessionTTL
	}
	return time.Duration(hours) * time.Hour
}

// ---------- Login / Logout ----------

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginHandler ตรวจ username+password แล้วออก session token เก็บใน httpOnly cookie
func LoginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "username and password are required")
			return
		}

		var userID int
		var passwordHash string
		var role string
		var isActive bool
		err := db.QueryRowContext(c.Request.Context(), `
			SELECT id, password_hash, role, is_active FROM users WHERE username = $1
		`, req.Username).Scan(&userID, &passwordHash, &role, &isActive)

		// สำคัญ: username ไม่พบ กับ password ผิด ต้องคืน error message เดียวกัน
		// เพื่อไม่เปิดช่องให้เดา (enumerate) ว่า username ไหนมีอยู่จริงในระบบ
		if err != nil {
			WriteError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
			return
		}

		if !isActive {
			WriteError(c, http.StatusForbidden, "ACCOUNT_INACTIVE", "account is inactive")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			WriteError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
			return
		}

		token, err := generateSessionToken()
		if err != nil {
			WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to start session")
			return
		}

		expiresAt := time.Now().Add(sessionTTL)
		if _, err := db.ExecContext(c.Request.Context(), `
			INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)
		`, token, userID, expiresAt); err != nil {
			WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to start session")
			return
		}

		setSessionCookie(c, token, sessionTTL) //Cookie ถูกตั้งเป็น HttpOnly
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{"id": userID, "username": req.Username, "role": role},
		})
	}
}

// LogoutHandler ลบ session ปัจจุบันออกจาก DB และล้าง cookie
func LogoutHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookieName)
		if err == nil && token != "" {
			// best-effort ลบทิ้ง - ไม่ error ให้ user แม้ session จะไม่มีอยู่แล้วก็ตาม
			_, _ = db.ExecContext(c.Request.Context(), `DELETE FROM sessions WHERE token = $1`, token)
		}
		clearSessionCookie(c)
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// ---------- Middleware ----------

func RequireAuth(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookieName)
		if err != nil || token == "" {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			c.Abort()
			return
		}

		var user AuthUser
		var expiresAt time.Time
		err = db.QueryRowContext(c.Request.Context(), `
			SELECT u.id, u.username, u.role, s.expires_at
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.token = $1
		`, token).Scan(&user.ID, &user.Username, &user.Role, &expiresAt)

		if err != nil || time.Now().After(expiresAt) {
			// session ไม่พบ หรือหมดอายุแล้ว -> เคลียร์ cookie เก่าทิ้งไปด้วย
			clearSessionCookie(c)
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired session")
			c.Abort()
			return
		}

		c.Set(authUserContextKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (AuthUser, bool) {
	value, exists := c.Get(authUserContextKey)
	if !exists {
		return AuthUser{}, false
	}
	user, ok := value.(AuthUser)
	return user, ok
}

// ---------- helpers ----------

func generateSessionToken() (string, error) {
	buf := make([]byte, 32) // 256 bit
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func setSessionCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		sessionCookieName,
		token,
		int(ttl.Seconds()),
		"/",
		"", // domain - ปล่อยว่างให้ browser ใช้ domain ปัจจุบัน (ปรับตอน deploy จริงถ้าต้องข้าม subdomain)
		true, // secure - บังคับตาม T-11b
		true,  // httpOnly - JS อ่านไม่ได้ ป้องกัน XSS ขโมย token
	)
}

func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", true, true)
}
