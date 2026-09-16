package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(db *sql.DB) *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// X-Session-User ไม่ใช้แล้ว - session มากับ httpOnly cookie อัตโนมัติ
		// AllowCredentials: true คือตัวที่สำคัญ ทำให้ browser แนบ cookie ข้าม origin ได้
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// login/logout อยู่นอก group ที่ต้อง auth (ยังไม่มี session ตอนเรียก)
	router.POST("/api/v1/login", LoginHandler(db))
	router.POST("/api/v1/logout", LogoutHandler(db))

	api := router.Group("/api/v1")
	api.Use(RequireAuth(db))
	{
		api.GET("/me", MeHandler())
		api.GET("/equipment-types", ListEquipmentTypesHandler(db))
		api.GET("/equipment", ListEquipmentHandler(db))
		api.POST("/equipment", CreateEquipmentHandler(db))
		api.POST("/import/preview", PreviewImportEquipmentHandler(db))
		api.POST("/import", ImportEquipmentHandler(db))
		api.GET("/import/history", ListImportHistoryHandler(db))
		api.GET("/dashboard", DashboardHandler(db))
		api.GET("/reports", ReportsHandler(db))
		api.PUT("/equipment/:id", UpdateEquipmentHandler(db))
		api.DELETE("/equipment/:id", DeleteEquipmentHandler(db))
		api.GET("/borrow-requests", ListBorrowRequestsHandler(db))
		api.GET("/borrow-requests/:id", GetBorrowRequestHandler(db))
		api.POST("/borrow-requests", CreateBorrowRequestHandler(db))
		api.PUT("/borrow-requests/:id/approve", ApproveBorrowRequestHandler(db))
		api.PUT("/borrow-requests/:id/reject", RejectBorrowRequestHandler(db))
		api.POST("/borrow-requests/:id/return", ReturnBorrowRequestHandler(db))
		api.PUT("/borrow-requests/:id/confirm-return", ConfirmReturnBorrowRequestHandler(db))
		api.GET("/repair-tickets", ListRepairTicketsHandler(db))
		api.POST("/repair-tickets", CreateRepairTicketHandler(db))
		api.GET("/repair-tickets/:id", GetRepairTicketHandler(db))
		api.PUT("/repair-tickets/:id/status", UpdateRepairTicketStatusHandler(db))
		api.GET("/activity-logs", ListActivityLogsHandler(db))
	}

	return router
}

func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{"code": code, "message": message},
	})
}
