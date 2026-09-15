package service

import (
	"context"
	"database/sql"

	"github.com/wrnxc/inventory-service/internal/repo"
)

type Dashboard struct {
	TotalEquipment             int `json:"total_equipment"`
	EquipmentInUse             int `json:"equipment_in_use"`
	EquipmentUnderRepair       int `json:"equipment_under_repair"`
	EquipmentBelowMinimum      int `json:"equipment_below_minimum"`
	PendingBorrowRequests      int `json:"pending_borrow_requests"`
	PendingReturnConfirmations int `json:"pending_return_confirmations"`
	PendingRepairTickets       int `json:"pending_repair_tickets"`
	SuccessfulBorrows          int `json:"successful_borrows"`
	UnderRepair                int `json:"under_repair"`
	InProgressRequests         int `json:"in_progress_requests"`
}

func GetDashboard(ctx context.Context, db *sql.DB, user AuthenticatedUser) (Dashboard, error) {
	var userID *int
	if user.Role == "user" {
		userID = &user.ID
	}
	counts, err := repo.GetDashboardCounts(ctx, db, userID)
	if err != nil {
		return Dashboard{}, err
	}

	result := Dashboard{
		TotalEquipment:             counts.TotalEquipment,
		EquipmentInUse:             counts.EquipmentInUse,
		EquipmentUnderRepair:       counts.EquipmentUnderRepair,
		EquipmentBelowMinimum:      counts.EquipmentBelowMinimum,
		PendingBorrowRequests:      counts.PendingBorrowRequests,
		PendingReturnConfirmations: counts.PendingReturnConfirmations,
		PendingRepairTickets:       counts.PendingRepairTickets,
		SuccessfulBorrows:          counts.SuccessfulBorrows,
		InProgressRequests:         counts.InProgressRequests,
	}
	if user.Role == "user" {
		result.TotalEquipment = 0
		result.EquipmentInUse = 0
		result.EquipmentUnderRepair = 0
		result.PendingBorrowRequests = counts.PendingBorrowRequests
		result.UnderRepair = counts.EquipmentUnderRepair
		result.PendingReturnConfirmations = 0
		result.PendingRepairTickets = 0
		result.EquipmentBelowMinimum = 0
	}
	return result, nil
}

type AuthenticatedUser struct {
	ID   int
	Role string
}
