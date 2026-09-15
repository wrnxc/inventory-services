package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/service"
)

type equipmentCreateRequest struct {
	TypeID               int    `json:"type_id"`
	ProductName          string `json:"product_name"`
	AssetName            string `json:"asset_name"`
	AssetTag             string `json:"asset_tag"`
	AssetSerialNo        string `json:"asset_serial_no"`
	BarCode              string `json:"bar_code"`
	VendorName           string `json:"vendor_name"`
	Location             string `json:"location"`
	AssignedToDepartment string `json:"assigned_to_department"`
	Site                 string `json:"site"`
	AssetNo              string `json:"asset_no"`
	Budget               string `json:"budget"`
	Remark               string `json:"remark"`
	Username             string `json:"username"`
	ReqNo                string `json:"req_no"`
}

type equipmentUpdateRequest struct {
	TypeID               int    `json:"type_id"`
	ProductName          string `json:"product_name"`
	AssetName            string `json:"asset_name"`
	AssetTag             string `json:"asset_tag"`
	AssetSerialNo        string `json:"asset_serial_no"`
	BarCode              string `json:"bar_code"`
	VendorName           string `json:"vendor_name"`
	Location             string `json:"location"`
	AssignedToDepartment string `json:"assigned_to_department"`
	Site                 string `json:"site"`
	AssetNo              string `json:"asset_no"`
	Budget               string `json:"budget"`
	Remark               string `json:"remark"`
	Username             string `json:"username"`
	ReqNo                string `json:"req_no"`
	Status               string `json:"status"`
}

func writeAppError(c *gin.Context, err error) {
	var appErr *service.AppError
	if errors.As(err, &appErr) {
		WriteError(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error")
}

func ListEquipmentHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := service.ListEquipment(c.Request.Context(), db)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	}
}

func CreateEquipmentHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		var req equipmentCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid request body")
			return
		}

		input := service.EquipmentInput{
			TypeID:               req.TypeID,
			ProductName:          req.ProductName,
			AssetName:            req.AssetName,
			AssetTag:             req.AssetTag,
			AssetSerialNo:        req.AssetSerialNo,
			BarCode:              req.BarCode,
			VendorName:           req.VendorName,
			Location:             req.Location,
			AssignedToDepartment: req.AssignedToDepartment,
			Site:                 req.Site,
			AssetNo:              req.AssetNo,
			Budget:               req.Budget,
			Remark:               req.Remark,
			Username:             req.Username,
			ReqNo:                req.ReqNo,
		}

		created, err := service.CreateEquipment(c.Request.Context(), db, user.ID, user.Role, input)
		if err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

func UpdateEquipmentHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "equipment not found")
			return
		}

		var req equipmentUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid request body")
			return
		}

		input := service.EquipmentInput{
			TypeID:               req.TypeID,
			ProductName:          req.ProductName,
			AssetName:            req.AssetName,
			AssetTag:             req.AssetTag,
			AssetSerialNo:        req.AssetSerialNo,
			BarCode:              req.BarCode,
			VendorName:           req.VendorName,
			Location:             req.Location,
			AssignedToDepartment: req.AssignedToDepartment,
			Site:                 req.Site,
			AssetNo:              req.AssetNo,
			Budget:               req.Budget,
			Remark:               req.Remark,
			Username:             req.Username,
			ReqNo:                req.ReqNo,
			Status:               req.Status,
		}

		if err := service.UpdateEquipment(c.Request.Context(), db, user.ID, user.Role, id, input); err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func DeleteEquipmentHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			WriteError(c, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "equipment not found")
			return
		}

		if err := service.DeleteEquipment(c.Request.Context(), db, user.ID, user.Role, id); err != nil {
			writeAppError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
