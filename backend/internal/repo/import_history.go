package repo

import (
    "context"
    "database/sql"
    "time"
)

type ImportHistory struct {
    ID int `json:"id"`
    FileName string `json:"file_name"`
    Total int `json:"total"`
    Success int `json:"success"`
    Duplicate int `json:"duplicate"`
    Failed int `json:"failed"`
    ImportedBy int `json:"imported_by"`
    Username string `json:"username"`
    ImportedAt time.Time `json:"imported_at"`
}

func InsertImportHistory(ctx context.Context, db *sql.DB, fileName string, total, success, duplicate, failed, importedBy int) error {
    _, err := db.ExecContext(ctx, `
        INSERT INTO import_history (file_name,total,success,duplicate,failed,imported_by)
        VALUES ($1,$2,$3,$4,$5,$6)
    `, fileName,total,success,duplicate,failed,importedBy)
    return err
}

func ListImportHistory(ctx context.Context, db *sql.DB) ([]ImportHistory, error) {
    rows, err := db.QueryContext(ctx, `
        SELECT ih.id,ih.file_name,ih.total,ih.success,ih.duplicate,ih.failed,
               ih.imported_by,u.username,ih.imported_at
        FROM import_history ih
        JOIN users u ON u.id = ih.imported_by
        ORDER BY ih.imported_at DESC, ih.id DESC
    `)
    if err != nil { return nil, err }
    defer rows.Close()

    items := make([]ImportHistory, 0)
    for rows.Next() {
        var x ImportHistory
        if err := rows.Scan(&x.ID,&x.FileName,&x.Total,&x.Success,&x.Duplicate,&x.Failed,&x.ImportedBy,&x.Username,&x.ImportedAt); err != nil {
            return nil, err
        }
        items = append(items, x)
    }
    return items, rows.Err()
}
