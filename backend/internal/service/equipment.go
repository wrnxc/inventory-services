package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/wrnxc/inventory-service/internal/repo"
)

type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

type EquipmentInput struct {
	TypeID               int
	ProductName          string
	AssetName            string
	AssetTag             string
	AssetSerialNo        string
	BarCode              string
	VendorName           string
	Location             string
	AssignedToDepartment string
	Site                 string
	AssetNo              string
	Budget               string
	Remark               string
	Username             string
	ReqNo                string
	Status               string
}

type Equipment struct {
	ID                   int       `json:"id"`
	TypeID               int       `json:"type_id"`
	TypeName             string    `json:"type_name"`
	ProductName          string    `json:"product_name"`
	AssetName            string    `json:"asset_name"`
	AssetTag             string    `json:"asset_tag"`
	AssetSerialNo        string    `json:"asset_serial_no"`
	BarCode              string    `json:"bar_code"`
	VendorName           string    `json:"vendor_name"`
	Location             string    `json:"location"`
	AssignedToDepartment string    `json:"assigned_to_department"`
	Site                 string    `json:"site"`
	AssetNo              string    `json:"asset_no"`
	Budget               string    `json:"budget"`
	Remark               string    `json:"remark"`
	Username             string    `json:"username"`
	ReqNo                string    `json:"req_no"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

var validEquipmentStatuses = map[string]bool{
	"กำลังใช้งาน": true,
	"ในคลัง":      true,
	"เสียหาย":     true,
	"เลิกใช้งาน":  true,
}

func validateEquipmentInput(input EquipmentInput, isCreate bool) error {
	if input.TypeID <= 0 {
		return NewAppError(422, "VALIDATION_ERROR", "type_id is required")
	}
	if strings.TrimSpace(input.ProductName) == "" {
		return NewAppError(422, "VALIDATION_ERROR", "product_name is required")
	}
	if strings.TrimSpace(input.AssetName) == "" {
		return NewAppError(422, "VALIDATION_ERROR", "asset_name is required")
	}
	if len(strings.TrimSpace(input.Budget)) > 10 {
		return NewAppError(422, "VALIDATION_ERROR", "budget must not exceed 10 characters")
	}
	if !isCreate && !validEquipmentStatuses[input.Status] {
		return NewAppError(422, "VALIDATION_ERROR", "invalid equipment status")
	}
	return nil
}

func toRepoInput(input EquipmentInput) repo.EquipmentInput {
	return repo.EquipmentInput{
		TypeID:               input.TypeID,
		ProductName:          strings.TrimSpace(input.ProductName),
		AssetName:            strings.TrimSpace(input.AssetName),
		AssetTag:             strings.TrimSpace(input.AssetTag),
		AssetSerialNo:        strings.TrimSpace(input.AssetSerialNo),
		BarCode:              strings.TrimSpace(input.BarCode),
		VendorName:           strings.TrimSpace(input.VendorName),
		Location:             strings.TrimSpace(input.Location),
		AssignedToDepartment: strings.TrimSpace(input.AssignedToDepartment),
		Site:                 strings.TrimSpace(input.Site),
		AssetNo:              strings.TrimSpace(input.AssetNo),
		Budget:               strings.TrimSpace(input.Budget),
		Remark:               strings.TrimSpace(input.Remark),
		Username:             strings.TrimSpace(input.Username),
		ReqNo:                strings.TrimSpace(input.ReqNo),
		Status:               input.Status,
	}
}

func fromRepoEquipment(r repo.Equipment) Equipment {
	return Equipment{
		ID:                   r.ID,
		TypeID:               r.TypeID,
		TypeName:             r.TypeName,
		ProductName:          r.ProductName.String,
		AssetName:            r.AssetName,
		AssetTag:             r.AssetTag.String,
		AssetSerialNo:        r.AssetSerialNo.String,
		BarCode:              r.BarCode.String,
		VendorName:           r.VendorName.String,
		Location:             r.Location.String,
		AssignedToDepartment: r.AssignedToDepartment.String,
		Site:                 r.Site.String,
		AssetNo:              r.AssetNo.String,
		Budget:               r.Budget.String,
		Remark:               r.Remark.String,
		Username:             r.Username.String,
		ReqNo:                r.ReqNo.String,
		Status:               r.Status,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func mapEquipmentRepoError(err error) error {
	switch {
	case errors.Is(err, repo.ErrEquipmentNotFound):
		return NewAppError(404, "EQUIPMENT_NOT_FOUND", "equipment not found")
	case errors.Is(err, repo.ErrAssetNameDuplicated):
		return NewAppError(409, "ASSET_NAME_DUPLICATED", "asset name already exists")
	case errors.Is(err, repo.ErrAssetSerialDuplicated):
		return NewAppError(409, "ASSET_SERIAL_DUPLICATED", "asset serial number already exists")
	case errors.Is(err, repo.ErrEquipmentTypeNotFound):
		return NewAppError(422, "VALIDATION_ERROR", "equipment type does not exist")
	default:
		return err
	}
}

func CreateEquipment(ctx context.Context, db *sql.DB, actorUserID int, role string, input EquipmentInput) (*Equipment, error) {
	if role != "admin" && role != "system_admin" {
		return nil, NewAppError(403, "FORBIDDEN", "only admin or system admin can create equipment")
	}
	if err := validateEquipmentInput(input, true); err != nil {
		return nil, err
	}

	created, err := repo.InsertEquipment(ctx, db, toRepoInput(input))
	if err != nil {
		return nil, mapEquipmentRepoError(err)
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "created", "equipment", created.ID, map[string]any{
		"asset_name":      created.AssetName,
		"asset_serial_no": created.AssetSerialNo.String,
		"type_id":         created.TypeID,
		"username":        created.Username.String,
	}); err != nil {
		return nil, err
	}

	result := fromRepoEquipment(*created)
	return &result, nil
}

func ListEquipment(ctx context.Context, db *sql.DB) ([]Equipment, error) {
	rows, err := repo.ListEquipment(ctx, db)
	if err != nil {
		return nil, err
	}

	items := make([]Equipment, 0, len(rows))
	for _, row := range rows {
		items = append(items, fromRepoEquipment(row))
	}
	return items, nil
}

func UpdateEquipment(ctx context.Context, db *sql.DB, actorUserID int, role string, equipmentID int, input EquipmentInput) error {
	if role != "admin" && role != "system_admin" {
		return NewAppError(403, "FORBIDDEN", "only admin or system admin can update equipment")
	}
	if err := validateEquipmentInput(input, false); err != nil {
		return err
	}

	if err := repo.UpdateEquipment(ctx, db, equipmentID, toRepoInput(input)); err != nil {
		return mapEquipmentRepoError(err)
	}

	return RecordActivityLog(ctx, db, actorUserID, "updated", "equipment", equipmentID, map[string]any{
		"asset_name":      input.AssetName,
		"asset_serial_no": input.AssetSerialNo,
		"type_id":         input.TypeID,
		"username":        input.Username,
		"status":          input.Status,
	})
}

func DeleteEquipment(ctx context.Context, db *sql.DB, actorUserID int, role string, equipmentID int) error {
	if role != "admin" && role != "system_admin" {
		return NewAppError(403, "FORBIDDEN", "only admin or system admin can delete equipment")
	}
	if err := repo.DeleteEquipment(ctx, db, equipmentID); err != nil {
		return mapEquipmentRepoError(err)
	}
	return RecordActivityLog(ctx, db, actorUserID, "deleted", "equipment", equipmentID, map[string]any{"deleted": true})
}
