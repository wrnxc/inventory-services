package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

type borrowCreateRequest struct {
	EquipmentID  int    `json:"equipment_id"`
	BorrowType   string `json:"borrow_type"`
	ReturnDate   string `json:"return_date"`
	BorrowerName string `json:"borrower_name"`
}

func ListBorrowRequestsHandler(db *sql.DB) gin.HandlerFunc {
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

		// debug: incoming status query available via c.Query("status")
		items, err := service.ListBorrowRequests(c.Request.Context(), db, c.Query("status"), equipmentID, c.Query("borrower_name"))
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	}
}

func GetBorrowRequestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
			return
		}

		item, err := service.GetBorrowRequest(c.Request.Context(), db, id)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func CreateBorrowRequestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		var req borrowCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid request body")
			return
		}

		item, err := service.CreateBorrowRequest(c.Request.Context(), db, user.ID, user.Role, service.BorrowRequestInput{
			EquipmentID:  req.EquipmentID,
			BorrowType:   req.BorrowType,
			ReturnDate:   req.ReturnDate,
			BorrowerName: req.BorrowerName,
		})
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusCreated, item)
	}
}

func ApproveBorrowRequestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
			return
		}

		item, err := service.ApproveBorrowRequest(c.Request.Context(), db, user.ID, user.Role, id)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func RejectBorrowRequestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
			return
		}

		item, err := service.RejectBorrowRequest(c.Request.Context(), db, user.ID, user.Role, id)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func ReturnBorrowRequestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
			return
		}

		item, err := service.RequestReturnBorrowRequest(c.Request.Context(), db, user.ID, user.Role, id)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func ConfirmReturnBorrowRequestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "BORROW_REQUEST_NOT_FOUND", "borrow request not found")
			return
		}

		item, err := service.ConfirmReturnBorrowRequest(c.Request.Context(), db, user.ID, user.Role, id)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}
