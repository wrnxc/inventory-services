package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

func ListEquipmentTypesHandler(
	db *sql.DB,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.ListEquipmentTypes(
			c.Request.Context(),
			db,
		)

		if err != nil {
			writeAppError(c, err)
			return
		}

		c.JSON(http.StatusOK, items)
	}
}
