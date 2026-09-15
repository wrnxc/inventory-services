package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

type repairTicketCreateRequest struct {
	EquipmentID      int    `json:"equipment_id"`
	IssueDescription string `json:"issue_description"`
	Urgency          string `json:"urgency"`
}

type repairTicketStatusUpdateRequest struct {
	Status string `json:"status"`
}

func ListRepairTicketsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var equipmentID *int
		if raw := c.Query("equipment_id"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil {
				WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "equipment_id must be a whole number")
				return
			}
			equipmentID = &value
		}

		items, err := service.ListRepairTickets(c.Request.Context(), db, c.Query("status"), equipmentID)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	}
}

func GetRepairTicketHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "REPAIR_TICKET_NOT_FOUND", "repair ticket not found")
			return
		}

		item, err := service.GetRepairTicket(c.Request.Context(), db, id)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func CreateRepairTicketHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		var req repairTicketCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid request body")
			return
		}

		item, err := service.CreateRepairTicket(c.Request.Context(), db, user.ID, user.Role, service.RepairTicketInput{
			EquipmentID:      req.EquipmentID,
			IssueDescription: req.IssueDescription,
			Urgency:          req.Urgency,
		})
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

func UpdateRepairTicketStatusHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}
		if user.Role != "admin" && user.Role != "system_admin" {
			WriteError(c, http.StatusForbidden, "FORBIDDEN", "only admin or system admin can update repair ticket status")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "REPAIR_TICKET_NOT_FOUND", "repair ticket not found")
			return
		}

		var req repairTicketStatusUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid request body")
			return
		}

		item, err := service.UpdateRepairTicketStatus(c.Request.Context(), db, user.ID, user.Role, id, req.Status)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func ListActivityLogsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.ListActivityLogs(c.Request.Context(), db)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	}
}
