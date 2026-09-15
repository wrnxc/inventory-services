package service

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wrnxc/inventory-service/internal/repo"
)

type ImportResult struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

var requiredImportHeaders = []string{"product_name", "asset_name", "asset_serial_no", "type_id"}

func ImportEquipment(ctx context.Context, db *sql.DB, actorUserID int, role string, input io.Reader) (ImportResult, error) {
	if role != "admin" && role != "system_admin" {
		return ImportResult{}, NewAppError(403, "FORBIDDEN", "only admin or system admin can import equipment")
	}

	reader := csv.NewReader(input)
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return ImportResult{}, NewAppError(422, "VALIDATION_ERROR", "CSV header is required")
	}
	if err != nil {
		return ImportResult{}, NewAppError(422, "VALIDATION_ERROR", "invalid CSV file")
	}

	headerIndexes, err := importHeaderIndexes(header)
	if err != nil {
		return ImportResult{}, err
	}

	result := ImportResult{}
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		result.Total++
		if readErr != nil || len(record) != len(header) {
			result.Failed++
			if readErr != nil {
				for {
					_, discardErr := reader.Read()
					if errors.Is(discardErr, io.EOF) {
						break
					}
				}
				break
			}
			continue
		}

		input, validationErr := parseImportRecord(record, headerIndexes)
		if validationErr != nil {
			result.Failed++
			continue
		}

		if _, insertErr := repo.InsertEquipment(ctx, db, input); insertErr != nil {
			if errors.Is(insertErr, repo.ErrAssetNameDuplicated) || errors.Is(insertErr, repo.ErrAssetSerialDuplicated) || errors.Is(insertErr, repo.ErrEquipmentTypeNotFound) {
				result.Failed++
				continue
			}
			return ImportResult{}, mapEquipmentRepoError(insertErr)
		}
		result.Success++
	}

	if err := RecordActivityLog(ctx, db, actorUserID, "import_equipment", "equipment", 0, map[string]any{
		"total":   result.Total,
		"success": result.Success,
		"failed":  result.Failed,
	}); err != nil {
		return ImportResult{}, err
	}

	return result, nil
}

func importHeaderIndexes(header []string) (map[string]int, error) {
	indexes := make(map[string]int, len(header))
	for index, value := range header {
		name := strings.TrimSpace(value)
		if name == "" {
			return nil, NewAppError(422, "VALIDATION_ERROR", "CSV header is invalid")
		}
		if _, exists := indexes[name]; exists {
			return nil, NewAppError(422, "VALIDATION_ERROR", "CSV header is invalid")
		}
		indexes[name] = index
	}
	for _, required := range requiredImportHeaders {
		if _, exists := indexes[required]; !exists {
			return nil, NewAppError(422, "VALIDATION_ERROR", "CSV header is invalid")
		}
	}
	return indexes, nil
}

func parseImportRecord(record []string, indexes map[string]int) (repo.EquipmentInput, error) {
	value := func(name string) string {
		return strings.TrimSpace(record[indexes[name]])
	}

	typeID, err := strconv.Atoi(value("type_id"))
	if err != nil || typeID <= 0 {
		return repo.EquipmentInput{}, fmt.Errorf("invalid type_id")
	}
	productName := value("product_name")
	assetName := value("asset_name")
	if productName == "" || assetName == "" {
		return repo.EquipmentInput{}, fmt.Errorf("product_name and asset_name are required")
	}

	return repo.EquipmentInput{
		TypeID:               typeID,
		ProductName:          productName,
		AssetName:            assetName,
		AssetTag:             importValue(record, indexes, "asset_tag"),
		AssetSerialNo:        value("asset_serial_no"),
		BarCode:              importValue(record, indexes, "bar_code"),
		VendorName:           importValue(record, indexes, "vendor_name"),
		Location:             importValue(record, indexes, "location"),
		AssignedToDepartment: importValue(record, indexes, "assigned_to_department"),
		Site:                 importValue(record, indexes, "site"),
		AssetNo:              importValue(record, indexes, "asset_no"),
		Budget:               importValue(record, indexes, "budget"),
		Remark:               importValue(record, indexes, "remark"),
		Username:             importValue(record, indexes, "username"),
		ReqNo:                importValue(record, indexes, "req_no"),
	}, nil
}

func importValue(record []string, indexes map[string]int, name string) string {
	index, exists := indexes[name]
	if !exists || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}
