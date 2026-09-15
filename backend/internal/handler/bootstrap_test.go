package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthEndpointReturnsOk(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if got := strings.TrimSpace(w.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("expected health payload %q, got %q", `{"status":"ok"}`, got)
	}
}

func TestWriteErrorUsesSharedResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	expected := `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`
	if got := strings.TrimSpace(w.Body.String()); got != expected {
		t.Fatalf("expected error payload %q, got %q", expected, got)
	}
}
