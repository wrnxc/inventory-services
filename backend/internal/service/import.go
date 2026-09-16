package service

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/csv"
    "errors"
    "fmt"
    "io"
    "path/filepath"
    "strings"

    "github.com/wrnxc/inventory-service/internal/repo"
    "github.com/xuri/excelize/v2"
)

type ImportResult struct {
    Total int `json:"total"`
    Success int `json:"success"`
    Duplicate int `json:"duplicate"`
    Failed int `json:"failed"`
}

var requiredImportHeaders = []string{"product_type","product_name","asset_name","asset_serial_no"}

func ImportEquipment(ctx context.Context, db *sql.DB, actorUserID int, role, fileName string, input io.Reader) (ImportResult, error) {
    if role != "admin" && role != "system_admin" {
        return ImportResult{}, NewAppError(403,"FORBIDDEN","only admin or system admin can import equipment")
    }

    records, err := readImportRecords(fileName, input)
    if err != nil { return ImportResult{}, err }
    if len(records) == 0 {
        return ImportResult{}, NewAppError(422,"VALIDATION_ERROR","file header is required")
    }

    indexes, err := importHeaderIndexes(records[0])
    if err != nil { return ImportResult{}, err }

    result := ImportResult{}
    for _, record := range records[1:] {
        if importRecordIsEmpty(record) { continue }
        result.Total++

        equipmentInput, err := parseImportRecord(ctx, db, record, indexes)
        if err != nil {
            result.Failed++
            continue
        }

        if _, err := repo.InsertEquipment(ctx, db, equipmentInput); err != nil {
            if errors.Is(err, repo.ErrAssetNameDuplicated) || errors.Is(err, repo.ErrAssetSerialDuplicated) {
                result.Duplicate++
                continue
            }
            if errors.Is(err, repo.ErrEquipmentTypeNotFound) {
                result.Failed++
                continue
            }
            return ImportResult{}, mapEquipmentRepoError(err)
        }
        result.Success++
    }

    if result.Total == 0 {
        return ImportResult{}, NewAppError(422,"VALIDATION_ERROR","no equipment data found in file")
    }

    if err := repo.InsertImportHistory(ctx,db,fileName,result.Total,result.Success,result.Duplicate,result.Failed,actorUserID); err != nil {
        return ImportResult{}, err
    }

    if err := RecordActivityLog(ctx,db,actorUserID,"import_equipment","equipment",0,map[string]any{
        "file_name":fileName,"total":result.Total,"success":result.Success,
        "duplicate":result.Duplicate,"failed":result.Failed,
    }); err != nil {
        return ImportResult{}, err
    }

    if result.Success == 0 && result.Failed == 0 && result.Duplicate == result.Total {
        return result, NewAppError(409,"DUPLICATE_IMPORT","ไม่สามารถนำเข้ารายการซ้ำได้")
    }
    return result, nil
}

func readImportRecords(fileName string, input io.Reader) ([][]string, error) {
    switch strings.ToLower(filepath.Ext(fileName)) {
    case ".csv":
        r := csv.NewReader(input)
        r.FieldsPerRecord = -1
        var records [][]string
        for {
            row, err := r.Read()
            if errors.Is(err, io.EOF) { break }
            if err != nil { return nil, NewAppError(422,"VALIDATION_ERROR","invalid CSV file") }
            records = append(records, row)
        }
        return records, nil
    case ".xlsx":
        data, err := io.ReadAll(input)
        if err != nil { return nil, NewAppError(422,"VALIDATION_ERROR","unable to read XLSX file") }
        book, err := excelize.OpenReader(bytes.NewReader(data))
        if err != nil { return nil, NewAppError(422,"VALIDATION_ERROR","invalid XLSX file") }
        defer book.Close()
        sheets := book.GetSheetList()
        if len(sheets) == 0 { return nil, NewAppError(422,"VALIDATION_ERROR","XLSX worksheet is required") }
        rows, err := book.GetRows(sheets[0])
        if err != nil { return nil, NewAppError(422,"VALIDATION_ERROR","unable to read XLSX worksheet") }
        return rows, nil
    default:
        return nil, NewAppError(422,"INVALID_FILE_TYPE","รองรับเฉพาะไฟล์ .csv และ .xlsx")
    }
}

func normalizeImportHeader(value string) string {
    value = strings.TrimSpace(strings.TrimPrefix(value,"\uFEFF"))
    m := map[string]string{
        "Product Type *":"product_type","Product Name *":"product_name","Asset Name *":"asset_name",
        "Asset Tag":"asset_tag","Asset Serial No.":"asset_serial_no","Bar Code":"bar_code",
        "Vendor Name":"vendor_name","Location":"location","Assigned To Department":"assigned_to_department",
        "Site":"site","Asset No.":"asset_no","Budget":"budget","Remark":"remark","UserName":"username",
        "Req. No":"req_no","Asset State":"asset_state",
        "product_type":"product_type","product_name":"product_name","asset_name":"asset_name",
        "asset_tag":"asset_tag","asset_serial_no":"asset_serial_no","bar_code":"bar_code",
        "vendor_name":"vendor_name","location":"location","assigned_to_department":"assigned_to_department",
        "site":"site","asset_no":"asset_no","budget":"budget","remark":"remark","username":"username",
        "req_no":"req_no","asset_state":"asset_state",
    }
    if v, ok := m[value]; ok { return v }
    return value
}

func importHeaderIndexes(header []string) (map[string]int, error) {
    indexes := make(map[string]int)
    for i, raw := range header {
        name := normalizeImportHeader(raw)
        if name == "" { continue }
        if _, exists := indexes[name]; exists {
            return nil, NewAppError(422,"INVALID_FILE_HEADER",fmt.Sprintf("duplicate file header: %s",name))
        }
        indexes[name] = i
    }
    for _, required := range requiredImportHeaders {
        if _, exists := indexes[required]; !exists {
            return nil, NewAppError(422,"INVALID_FILE_HEADER",fmt.Sprintf("missing required file header: %s",required))
        }
    }
    return indexes, nil
}

func parseImportRecord(ctx context.Context, db *sql.DB, record []string, indexes map[string]int) (repo.EquipmentInput, error) {
    value := func(name string) string { return importValue(record,indexes,name) }
    productType, productName, assetName := value("product_type"), value("product_name"), value("asset_name")
    if productType == "" { return repo.EquipmentInput{}, fmt.Errorf("product_type is required") }
    if productName == "" { return repo.EquipmentInput{}, fmt.Errorf("product_name is required") }
    if assetName == "" { return repo.EquipmentInput{}, fmt.Errorf("asset_name is required") }

    typeID, err := repo.FindEquipmentTypeIDByName(ctx,db,productType)
    if err != nil { return repo.EquipmentInput{}, fmt.Errorf("equipment type %q not found: %w",productType,err) }

    return repo.EquipmentInput{
        TypeID:typeID, ProductName:productName, AssetName:assetName,
        AssetTag:value("asset_tag"), AssetSerialNo:value("asset_serial_no"), BarCode:value("bar_code"),
        VendorName:value("vendor_name"), Location:value("location"),
        AssignedToDepartment:value("assigned_to_department"), Site:value("site"), AssetNo:value("asset_no"),
        Budget:value("budget"), Remark:value("remark"), Username:value("username"), ReqNo:value("req_no"),
        Status:value("asset_state"),
    }, nil
}

func importValue(record []string, indexes map[string]int, name string) string {
    i, ok := indexes[name]
    if !ok || i >= len(record) { return "" }
    return strings.TrimSpace(record[i])
}

func importRecordIsEmpty(record []string) bool {
    for _, v := range record {
        if strings.TrimSpace(v) != "" { return false }
    }
    return true
}

type ImportPreviewItem struct {
	RowNumber   int    `json:"row_number"`
	ProductType string `json:"product_type"`
	ProductName string `json:"product_name"`
	AssetName   string `json:"asset_name"`
	SerialNo    string `json:"asset_serial_no"`
}

type ImportPreviewResult struct {
	FileName string              `json:"file_name"`
	Total    int                 `json:"total"`
	Preview  []ImportPreviewItem `json:"preview"`
}

func PreviewEquipmentImport(
	ctx context.Context,
	db *sql.DB,
	fileName string,
	input io.Reader,
) (ImportPreviewResult, error) {

	// ใช้ parser ตัวเดียวกับ Import จริง
	// รองรับทั้ง .csv และ .xlsx
	records, err := readImportRecords(fileName, input)
	if err != nil {
		return ImportPreviewResult{}, err
	}

	// ต้องมีอย่างน้อย header
	if len(records) == 0 {
		return ImportPreviewResult{}, NewAppError(
			422,
			"VALIDATION_ERROR",
			"file header is required",
		)
	}

	// ใช้การตรวจสอบ header ชุดเดียวกับ Import จริง
	indexes, err := importHeaderIndexes(records[0])
	if err != nil {
		return ImportPreviewResult{}, err
	}

	result := ImportPreviewResult{
		FileName: fileName,
		Preview:  make([]ImportPreviewItem, 0, 5),
	}

	for i, record := range records[1:] {

		// ข้ามแถวว่าง
		if importRecordIsEmpty(record) {
			continue
		}

		result.Total++

		// Preview แค่ 5 รายการแรก
		if len(result.Preview) >= 5 {
			continue
		}

		result.Preview = append(
			result.Preview,
			ImportPreviewItem{
				// +2 เพราะ
				// index 0 = data row แรก
				// แต่ในไฟล์ row 1 = header
				RowNumber: i + 2,

				ProductType: importValue(
					record,
					indexes,
					"product_type",
				),

				ProductName: importValue(
					record,
					indexes,
					"product_name",
				),

				AssetName: importValue(
					record,
					indexes,
					"asset_name",
				),

				SerialNo: importValue(
					record,
					indexes,
					"asset_serial_no",
				),
			},
		)
	}

	if result.Total == 0 {
		return ImportPreviewResult{}, NewAppError(
			422,
			"VALIDATION_ERROR",
			"no equipment data found in file",
		)
	}

	return result, nil
}
