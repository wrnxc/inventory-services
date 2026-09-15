package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

func DashboardHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}
		result, err := service.GetDashboard(c.Request.Context(), db, service.AuthenticatedUser{ID: user.ID, Role: user.Role})
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
