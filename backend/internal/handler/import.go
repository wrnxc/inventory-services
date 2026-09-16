package handler

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wrnxc/inventory-service/internal/repo"
	"github.com/wrnxc/inventory-service/internal/service"
)

func ImportEquipmentHandler(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, ok := CurrentUser(c)
        if !ok { WriteError(c,http.StatusUnauthorized,"UNAUTHORIZED","missing session"); return }
        if user.Role != "admin" && user.Role != "system_admin" {
            WriteError(c,http.StatusForbidden,"FORBIDDEN","only admin or system admin can import equipment"); return
        }

        header, err := c.FormFile("file")
        if err != nil { WriteError(c,http.StatusUnprocessableEntity,"VALIDATION_ERROR","file is required"); return }

        ext := strings.ToLower(filepath.Ext(header.Filename))
        if ext != ".csv" && ext != ".xlsx" {
            WriteError(c,http.StatusUnprocessableEntity,"INVALID_FILE_TYPE","รองรับเฉพาะไฟล์ .csv และ .xlsx"); return
        }

        file, err := header.Open()
        if err != nil { WriteError(c,http.StatusUnprocessableEntity,"VALIDATION_ERROR","unable to read import file"); return }
        defer file.Close()

        result, err := service.ImportEquipment(c.Request.Context(),db,user.ID,user.Role,header.Filename,file)
        if err != nil { writeAppError(c,err); return }
        c.JSON(http.StatusCreated,result)
    }
}

func ListImportHistoryHandler(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, ok := CurrentUser(c)
        if !ok { WriteError(c,http.StatusUnauthorized,"UNAUTHORIZED","missing session"); return }
        if user.Role != "admin" && user.Role != "system_admin" {
            WriteError(c,http.StatusForbidden,"FORBIDDEN","only admin or system admin can view import history"); return
        }
        items, err := repo.ListImportHistory(c.Request.Context(),db)
        if err != nil { WriteError(c,http.StatusInternalServerError,"INTERNAL_ERROR","unable to load import history"); return }
        c.JSON(http.StatusOK,items)
    }
}

func PreviewImportEquipmentHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)

		if !ok {
			WriteError(
				c,
				http.StatusUnauthorized,
				"UNAUTHORIZED",
				"missing session",
			)
			return
		}

		if user.Role != "admin" &&
			user.Role != "system_admin" {

			WriteError(
				c,
				http.StatusForbidden,
				"FORBIDDEN",
				"only admin or system admin can preview import",
			)
			return
		}

		header, err := c.FormFile("file")

		if err != nil {
			WriteError(
				c,
				http.StatusUnprocessableEntity,
				"VALIDATION_ERROR",
				"file is required",
			)
			return
		}

		ext := strings.ToLower(
			filepath.Ext(header.Filename),
		)

		if ext != ".csv" &&
			ext != ".xlsx" {

			WriteError(
				c,
				http.StatusUnprocessableEntity,
				"INVALID_FILE_TYPE",
				"รองรับเฉพาะไฟล์ .csv และ .xlsx",
			)
			return
		}

		file, err := header.Open()

		if err != nil {
			WriteError(
				c,
				http.StatusUnprocessableEntity,
				"VALIDATION_ERROR",
				"unable to read import file",
			)
			return
		}

		defer file.Close()

		result, err :=
			service.PreviewEquipmentImport(
				c.Request.Context(),
				db,
				header.Filename,
				file,
			)

		if err != nil {
			writeAppError(c, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}