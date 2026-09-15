package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

func ImportEquipmentHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}
		if user.Role != "admin" && user.Role != "system_admin" {
			WriteError(c, http.StatusForbidden, "FORBIDDEN", "only admin or system admin can import equipment")
			return
		}

		header, err := c.FormFile("file")
		if err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "CSV file is required")
			return
		}

		file, err := header.Open()
		if err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "unable to read CSV file")
			return
		}
		defer file.Close()

		result, err := service.ImportEquipment(c.Request.Context(), db, user.ID, user.Role, file)
		if err != nil {
			writeAppError(c, err)
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}
