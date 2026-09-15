package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

func ReportsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}
		result, err := service.GetReport(
			c.Request.Context(),
			db,
			user.Role,
			c.Query("type"),
			c.Query("from"),
			c.Query("to"),
		)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
