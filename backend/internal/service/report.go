package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/wrnxc/inventory-service/internal/repo"
)

var validReportTypes = map[string]bool{
	"stock":  true,
	"borrow": true,
	"return": true,
	"repair": true,
}

func GetReport(ctx context.Context, db *sql.DB, role, reportType, fromValue, toValue string) (repo.ReportResult, error) {
	if role != "admin" && role != "system_admin" {
		return repo.ReportResult{}, NewAppError(403, "FORBIDDEN", "only admin or system admin can view reports")
	}
	if !validReportTypes[reportType] {
		return repo.ReportResult{}, NewAppError(422, "VALIDATION_ERROR", "invalid report type")
	}

	from, err := parseReportDate(fromValue)
	if err != nil {
		return repo.ReportResult{}, NewAppError(422, "VALIDATION_ERROR", "invalid from date")
	}
	to, err := parseReportDate(toValue)
	if err != nil {
		return repo.ReportResult{}, NewAppError(422, "VALIDATION_ERROR", "invalid to date")
	}
	if from != nil && to != nil && to.Before(*from) {
		return repo.ReportResult{}, NewAppError(422, "VALIDATION_ERROR", "from date must not be after to date")
	}

	result, err := repo.GetReport(ctx, db, reportType, from, to)
	if err != nil {
		return repo.ReportResult{}, err
	}
	return result, nil
}

func parseReportDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
