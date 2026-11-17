package master

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xuri/excelize/v2"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/response"
	"wbs-golang/backend/internal/platform/security"
)

const (
	importMaxBytes = 5 << 20
	importMaxRows  = 1000
)

type importRow struct {
	Line         int               `json:"line"`
	Data         map[string]string `json:"data"`
	Errors       []string          `json:"errors"`
	Action       string            `json:"action,omitempty"`
	TargetID     string            `json:"target_id,omitempty"`
	ExpectedHash string            `json:"expected_hash"`
}
type importPreview struct {
	Rows []importRow `json:"rows"`
}
type importSummary struct {
	Inserted               int    `json:"inserted"`
	Updated                int    `json:"updated"`
	CreatedUsersInactive   int    `json:"created_users_inactive"`
	CredentialProvisioning string `json:"credential_provisioning"`
}

var importColumns = map[string][]string{
	"report-types": {"id", "parent_id", "parent_name", "name", "description", "is_active"},
	"admin-users":  {"id", "name", "email", "role_name", "is_active"},
	"faqs":         {"id", "parent_id", "parent_name", "question", "answer", "sort_order", "is_active"},
}

func parseBoolean(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
}

func parseWorkbook(data []byte, module string) (importPreview, error) {
	var result importPreview
	if len(data) < 4 || !bytes.Equal(data[:4], []byte("PK\x03\x04")) {
		return result, invalid("File harus merupakan XLSX yang valid.")
	}
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: importMaxBytes * 8, UnzipXMLSizeLimit: importMaxBytes * 4})
	if err != nil {
		return result, invalid("Workbook Excel tidak dapat dibaca.")
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "Import" {
		return result, invalid("Workbook harus memiliki satu sheet bernama Import.")
	}
	rows, err := f.GetRows("Import", excelize.Options{RawCellValue: true})
	if err != nil || len(rows) == 0 || len(rows)-1 > importMaxRows {
		return result, invalid("Jumlah baris atau struktur workbook tidak valid.")
	}
	expected, ok := importColumns[module]
	if !ok {
		return result, invalid("Jenis import tidak didukung.")
	}
	if len(rows[0]) != len(expected) {
		return result, invalid("Kolom workbook tidak sesuai template.")
	}
	for i, key := range expected {
		if strings.TrimSpace(rows[0][i]) != key {
			return result, invalid("Header workbook tidak sesuai template.")
		}
	}
	seen := map[string]int{}
	for index, cells := range rows[1:] {
		if len(cells) == 0 || (len(cells) == 1 && strings.TrimSpace(cells[0]) == "") {
			continue
		}
		line := index + 2
		row := importRow{Line: line, Data: map[string]string{}, Errors: []string{}}
		for i, key := range expected {
			if i < len(cells) {
				row.Data[key] = strings.TrimSpace(cells[i])
			}
			if len(row.Data[key]) > 20000 {
				row.Errors = append(row.Errors, key+" melebihi batas panjang.")
			}
		}
		// Formula cells are rejected; spreadsheet formulas are never evaluated.
		for i := range expected {
			cell, _ := excelize.CoordinatesToCellName(i+1, line)
			formula, formulaErr := f.GetCellFormula("Import", cell)
			if formulaErr == nil && formula != "" {
				row.Errors = append(row.Errors, "formula tidak diizinkan.")
				break
			}
		}
		validateImportRow(module, &row)
		key := importIdentity(module, row.Data)
		if key != "" {
			if first, duplicate := seen[key]; duplicate {
				row.Errors = append(row.Errors, fmt.Sprintf("baris duplikat dengan baris %d.", first))
			} else {
				seen[key] = line
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if len(result.Rows) == 0 {
		return result, invalid("Workbook tidak berisi baris data.")
	}
	return result, nil
}

func validateImportRow(module string, row *importRow) {
	v := row.Data
	if v["id"] != "" && !security.UUID.MatchString(v["id"]) {
		row.Errors = append(row.Errors, "ID harus UUID valid.")
	}
	if v["is_active"] != "true" && v["is_active"] != "false" && v["is_active"] != "1" && v["is_active"] != "0" {
		row.Errors = append(row.Errors, "is_active harus true atau false.")
	}
	switch module {
	case "report-types":
		if v["name"] == "" {
			row.Errors = append(row.Errors, "name wajib diisi.")
		}
		if v["parent_id"] != "" && !security.UUID.MatchString(v["parent_id"]) {
			row.Errors = append(row.Errors, "parent_id harus UUID valid.")
		}
		if v["parent_id"] != "" && v["parent_name"] == "" {
			row.Errors = append(row.Errors, "parent_name wajib untuk subkategori.")
		}
	case "admin-users":
		if v["name"] == "" || v["email"] == "" || v["role_name"] == "" {
			row.Errors = append(row.Errors, "name, email, dan role_name wajib diisi.")
		}
		if len(v["email"]) > 254 || !strings.Contains(v["email"], "@") || strings.ContainsAny(v["email"], "\r\n ") {
			row.Errors = append(row.Errors, "email tidak valid.")
		} else if address, err := mail.ParseAddress(v["email"]); err != nil || address.Address != v["email"] {
			row.Errors = append(row.Errors, "email tidak valid.")
		}
	case "faqs":
		if v["question"] == "" || v["answer"] == "" {
			row.Errors = append(row.Errors, "question dan answer wajib diisi.")
		}
		if v["parent_id"] != "" && !security.UUID.MatchString(v["parent_id"]) {
			row.Errors = append(row.Errors, "parent_id harus UUID valid.")
		}
		if v["parent_id"] != "" && v["parent_name"] == "" {
			row.Errors = append(row.Errors, "parent_name wajib untuk FAQ kategori.")
		}
		n, err := strconv.Atoi(v["sort_order"])
		if err != nil || n < 0 {
			row.Errors = append(row.Errors, "sort_order harus integer nonnegatif.")
		}
	}
}

func importIdentity(module string, row map[string]string) string {
	if row["id"] != "" {
		return "id:" + strings.ToLower(row["id"])
	}
	switch module {
	case "report-types":
		return "type:" + strings.ToLower(strings.TrimSpace(row["parent_name"])) + "/" + strings.ToLower(strings.TrimSpace(row["name"]))
	case "admin-users":
		return "email:" + strings.ToLower(strings.TrimSpace(row["email"]))
	case "faqs":
		return "faq:" + strings.ToLower(strings.TrimSpace(row["parent_name"])) + "/" + strings.ToLower(strings.TrimSpace(row["question"]))
	}
	return ""
}

func workbook(module string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	if err := f.SetSheetName(sheet, "Import"); err != nil {
		return nil, err
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		values := make([]any, len(row))
		for j := range row {
			values[j] = row[j]
		}
		if err := f.SetSheetRow("Import", cell, &values); err != nil {
			return nil, err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func setWorkbookHeaders(w http.ResponseWriter, name string) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.xlsx"`)
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func (h *Handler) export(c *gin.Context, e Entity, templateOnly bool) {
	columns, ok := importColumns[e.Path]
	if !ok {
		h.auth.Fail(c, invalid("Export tidak tersedia."))
		return
	}
	rows := [][]string{columns}
	if !templateOnly {
		ctx, cancel := context.WithTimeout(c.Request.Context(), h.s.timeout)
		defer cancel()
		var query string
		switch e.Path {
		case "report-types":
			query = `SELECT t.id::text,COALESCE(t.parent_id::text,''),COALESCE(p.name,''),t.name,t.description,t.is_active::text FROM report_types t LEFT JOIN report_types p ON p.id=t.parent_id ORDER BY t.parent_id NULLS FIRST,t.name,t.id`
		case "faqs":
			query = `SELECT f.id::text,COALESCE(f.report_type_id::text,''),COALESCE(rt.name,''),f.question,f.answer,f.sort_order::text,f.is_active::text FROM faqs f LEFT JOIN report_types rt ON rt.id=f.report_type_id ORDER BY f.sort_order,f.id`
		case "admin-users":
			query = `SELECT u.id::text,u.name,u.email,r.name,u.is_active::text FROM admin_users u JOIN roles r ON r.id=u.role_id ORDER BY lower(u.email)`
		}
		dbRows, err := h.s.repo.pool.Query(ctx, query+` LIMIT 10001`)
		if err != nil {
			h.auth.Fail(c, err)
			return
		}
		defer dbRows.Close()
		for dbRows.Next() {
			row := make([]string, len(columns))
			targets := make([]any, len(row))
			for i := range row {
				targets[i] = &row[i]
			}
			if err = dbRows.Scan(targets...); err != nil {
				h.auth.Fail(c, err)
				return
			}
			rows = append(rows, row)
		}
		if err = dbRows.Err(); err != nil {
			h.auth.Fail(c, err)
			return
		}
		if len(rows) > 10001 {
			h.auth.Fail(c, problem.New(413, "EXPORT_TOO_LARGE", "Export melebihi batas 10.000 baris."))
			return
		}
	}
	buffer, err := workbook(e.Path, rows)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	name := strings.ReplaceAll(e.Path, "-", "_")
	if templateOnly {
		name += "_template"
	} else {
		name += "_export"
	}
	setWorkbookHeaders(c.Writer, name)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}

func (h *Handler) previewImport(c *gin.Context, e Entity) {
	if c.Request.ContentLength > importMaxBytes {
		response.Error(c, 413, "IMPORT_TOO_LARGE", "Ukuran workbook melebihi 5 MiB.")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importMaxBytes)
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		h.auth.Fail(c, invalid("Multipart import tidak valid atau terlalu besar."))
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	files := c.Request.MultipartForm.File["file"]
	if len(files) != 1 || strings.ToLower(filepath.Ext(files[0].Filename)) != ".xlsx" {
		h.auth.Fail(c, invalid("Pilih tepat satu file .xlsx."))
		return
	}
	file, err := files[0].Open()
	if err != nil {
		h.auth.Fail(c, invalid("File import tidak dapat dibaca."))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, importMaxBytes+1))
	if err != nil || len(data) > importMaxBytes {
		response.Error(c, 413, "IMPORT_TOO_LARGE", "Ukuran workbook melebihi 5 MiB.")
		return
	}
	plan, err := parseWorkbook(data, e.Path)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.s.timeout)
	defer cancel()
	insertCandidates, updateCandidates := 0, 0
	workbookParents := map[string]bool{}
	if e.Path == "report-types" {
		for _, row := range plan.Rows {
			if len(row.Errors) == 0 && row.Data["parent_id"] == "" && row.Data["parent_name"] == "" {
				workbookParents[normalizeImportKey(row.Data["name"])] = true
			}
		}
	}
	for i := range plan.Rows {
		if len(plan.Rows[i].Errors) != 0 {
			continue
		}
		if refErr := validateImportReference(ctx, h.s.repo.pool, e.Path, plan.Rows[i], workbookParents); refErr != nil {
			plan.Rows[i].Errors = append(plan.Rows[i].Errors, refErr.Error())
			continue
		}
		action, targetID, expected, lookupErr := lookupImportAction(ctx, h.s.repo.pool, e.Path, plan.Rows[i], workbookParents)
		if lookupErr != nil {
			plan.Rows[i].Errors = append(plan.Rows[i].Errors, "identitas atau referensi tidak dapat divalidasi.")
			continue
		}
		plan.Rows[i].Action, plan.Rows[i].TargetID, plan.Rows[i].ExpectedHash = action, targetID, expected
		if action == "update" {
			updateCandidates++
		} else {
			insertCandidates++
		}
	}
	token := security.Token()
	payload, _ := json.Marshal(plan)
	if _, err = h.s.repo.pool.Exec(ctx, `DELETE FROM master_import_previews WHERE expires_at < now()-interval '1 day' AND (committed_at IS NULL OR committed_at < now()-interval '30 days')`); err != nil {
		h.auth.Fail(c, err)
		return
	}
	_, err = h.s.repo.pool.Exec(ctx, `INSERT INTO master_import_previews(admin_id,module,token_hash,payload,expires_at) VALUES($1,$2,$3,$4,now()+interval '15 minutes')`, auth.Current(c).ID, e.Path, security.HashToken(token), payload)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	valid, invalidCount := 0, 0
	for _, row := range plan.Rows {
		if len(row.Errors) == 0 {
			valid++
		} else {
			invalidCount++
		}
	}
	c.Header("Cache-Control", "no-store, private")
	publicRows := make([]gin.H, 0, len(plan.Rows))
	for _, row := range plan.Rows {
		publicRows = append(publicRows, gin.H{"line": row.Line, "data": row.Data, "errors": row.Errors, "action": row.Action})
	}
	response.Success(c, 200, gin.H{"preview_token": token, "expires_in_seconds": 900, "total_rows": len(plan.Rows), "valid_rows": valid, "invalid_rows": invalidCount, "insert_candidates": insertCandidates, "update_candidates": updateCandidates, "rows": publicRows, "credential_provisioning": "Akun baru dari import dibuat nonaktif; administrator perlu mengatur password dan aktivasi secara terpisah."})
}

func (h *Handler) commitImport(c *gin.Context, e Entity) {
	var req struct {
		Token string `json:"preview_token"`
	}
	if err := auth.Decode(c, &req); err != nil {
		h.auth.Fail(c, err)
		return
	}
	if len(req.Token) != 64 {
		h.auth.Fail(c, invalid("Preview token tidak valid."))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.s.timeout*2)
	defer cancel()
	tx, err := h.s.repo.pool.Begin(ctx)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auth.AdminLock); err != nil {
		h.auth.Fail(c, err)
		return
	}
	var owner, module string
	var raw []byte
	var resultRaw []byte
	var expires time.Time
	var committed *time.Time
	err = tx.QueryRow(ctx, `SELECT admin_id::text,module,payload,result,expires_at,committed_at FROM master_import_previews WHERE token_hash=$1 FOR UPDATE`, security.HashToken(req.Token)).Scan(&owner, &module, &raw, &resultRaw, &expires, &committed)
	if errors.Is(err, pgx.ErrNoRows) || owner != auth.Current(c).ID || module != e.Path {
		h.auth.Fail(c, problem.New(404, "IMPORT_PREVIEW_NOT_FOUND", "Preview import tidak ditemukan."))
		return
	}
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	if committed != nil {
		if err = tx.Commit(ctx); err != nil {
			h.auth.Fail(c, err)
			return
		}
		c.Header("Cache-Control", "no-store, private")
		response.Success(c, 200, json.RawMessage(resultRaw))
		return
	}
	if !expires.After(time.Now()) {
		h.auth.Fail(c, problem.New(409, "IMPORT_PREVIEW_EXPIRED", "Preview kedaluwarsa. Silakan unggah ulang."))
		return
	}
	fresh, err := auth.Resolve(ctx, tx, auth.Current(c).ID)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	if err = authorize(fresh, e); err != nil {
		h.auth.Fail(c, err)
		return
	}
	var plan importPreview
	if err = json.Unmarshal(raw, &plan); err != nil {
		h.auth.Fail(c, errors.New("invalid import preview payload"))
		return
	}
	for _, row := range plan.Rows {
		if len(row.Errors) > 0 {
			h.auth.Fail(c, problem.New(422, "IMPORT_VALIDATION_FAILED", fmt.Sprintf("Workbook memiliki error pada baris %d.", row.Line)))
			return
		}
		if row.Action == "update" {
			currentHash, hashErr := importRowHash(ctx, tx, e.Path, row.TargetID)
			if hashErr != nil || currentHash != row.ExpectedHash {
				h.auth.Fail(c, problem.New(409, "IMPORT_PREVIEW_STALE", fmt.Sprintf("Data pada baris %d berubah setelah preview. Buat preview baru.", row.Line)))
				return
			}
		} else if row.Action != "insert" {
			h.auth.Fail(c, problem.New(422, "IMPORT_VALIDATION_FAILED", fmt.Sprintf("Baris %d belum valid.", row.Line)))
			return
		}
	}
	plannedRootInserts := map[string]bool{}
	if e.Path == "report-types" {
		for _, row := range plan.Rows {
			if row.Action == "insert" && row.Data["parent_id"] == "" && row.Data["parent_name"] == "" {
				plannedRootInserts[normalizeImportKey(row.Data["name"])] = true
			}
		}
	}
	// Recheck natural-key inserts under the same administrative lock before applying
	// any rows. This prevents preview/commit races without mistaking parents created
	// earlier in this same batch for stale inserts.
	for _, row := range plan.Rows {
		if row.Action != "insert" {
			continue
		}
		if e.Path == "report-types" && row.Data["parent_name"] != "" && plannedRootInserts[normalizeImportKey(row.Data["parent_name"])] {
			continue
		}
		action, _, _, checkErr := lookupImportAction(ctx, tx, e.Path, row, nil)
		if checkErr != nil || action != "insert" {
			h.auth.Fail(c, problem.New(409, "IMPORT_PREVIEW_STALE", fmt.Sprintf("Identitas pada baris %d berubah setelah preview. Buat preview baru.", row.Line)))
			return
		}
	}
	if e.Path == "report-types" {
		sortReportTypeRows(plan.Rows)
	}
	summary := importSummary{CredentialProvisioning: "Akun baru dibuat nonaktif; administrator harus menetapkan password dan mengaktifkan akun."}
	for _, row := range plan.Rows {
		if len(row.Errors) > 0 {
			continue
		}
		if err = applyImportRow(ctx, tx, e.Path, row, fresh.ID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				h.auth.Fail(c, problem.New(409, "IMPORT_CONFLICT", "Baris import bertentangan dengan data saat ini."))
				return
			}
			h.auth.Fail(c, err)
			return
		}
		if row.Action == "update" {
			summary.Updated++
		} else {
			summary.Inserted++
		}
		if e.Path == "admin-users" && row.Action == "insert" {
			summary.CreatedUsersInactive++
		}
	}
	resultRaw, err = json.Marshal(summary)
	if err != nil {
		h.auth.Fail(c, err)
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE master_import_previews SET committed_at=now(),result=$2 WHERE token_hash=$1`, security.HashToken(req.Token), resultRaw); err != nil {
		h.auth.Fail(c, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		h.auth.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store, private")
	response.Success(c, 200, summary)
}

func applyImportRow(ctx context.Context, tx pgx.Tx, module string, row importRow, actorID string) error {
	v := row.Data
	switch module {
	case "report-types":
		var parentID any
		if v["parent_id"] != "" {
			parentID = v["parent_id"]
		} else if v["parent_name"] != "" {
			if err := tx.QueryRow(ctx, `SELECT id FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1))`, v["parent_name"]).Scan(&parentID); err != nil {
				return fmt.Errorf("unknown parent")
			}
		}
		active := parseBoolean(v["is_active"])
		if v["id"] != "" {
			var currentParent *string
			if err := tx.QueryRow(ctx, `SELECT parent_id::text FROM report_types WHERE id=$1 FOR UPDATE`, v["id"]).Scan(&currentParent); err != nil {
				return fmt.Errorf("report type id missing")
			}
			if !sameNullable(currentParent, parentID) {
				return fmt.Errorf("report type parent changes require explicit CRUD")
			}
			_, err := tx.Exec(ctx, `UPDATE report_types SET name=$2,description=$3,is_active=$4,updated_at=now() WHERE id=$1`, v["id"], v["name"], v["description"], active)
			return err
		}
		var existing string
		if parentID == nil {
			_ = tx.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1))`, v["name"]).Scan(&existing)
		} else {
			_ = tx.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id=$1 AND lower(btrim(name))=lower(btrim($2))`, parentID, v["name"]).Scan(&existing)
		}
		if existing != "" {
			_, err := tx.Exec(ctx, `UPDATE report_types SET description=$2,is_active=$3,updated_at=now() WHERE id=$1`, existing, v["description"], active)
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO report_types(parent_id,name,description,is_active) VALUES($1,$2,$3,$4)`, parentID, v["name"], v["description"], active)
		return err
	case "faqs":
		var parentID any
		if v["parent_id"] != "" {
			parentID = v["parent_id"]
		} else if v["parent_name"] != "" {
			if err := tx.QueryRow(ctx, `SELECT id FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1))`, v["parent_name"]).Scan(&parentID); err != nil {
				return fmt.Errorf("unknown FAQ parent")
			}
		}
		var order int
		if _, err := fmt.Sscan(v["sort_order"], &order); err != nil {
			return err
		}
		active := parseBoolean(v["is_active"])
		if v["id"] != "" {
			var currentParent *string
			if err := tx.QueryRow(ctx, `SELECT report_type_id::text FROM faqs WHERE id=$1 FOR UPDATE`, v["id"]).Scan(&currentParent); err != nil {
				return fmt.Errorf("FAQ id missing")
			}
			if !sameNullable(currentParent, parentID) {
				return fmt.Errorf("FAQ category changes require explicit CRUD")
			}
			_, err := tx.Exec(ctx, `UPDATE faqs SET question=$2,answer=$3,sort_order=$4,is_active=$5 WHERE id=$1`, v["id"], v["question"], v["answer"], order, active)
			return err
		}
		var id string
		var count int
		if parentID == nil {
			_ = tx.QueryRow(ctx, `SELECT count(*),min(id::text) FROM faqs WHERE report_type_id IS NULL AND lower(btrim(question))=lower(btrim($1))`, v["question"]).Scan(&count, &id)
		} else {
			_ = tx.QueryRow(ctx, `SELECT count(*),min(id::text) FROM faqs WHERE report_type_id=$1 AND lower(btrim(question))=lower(btrim($2))`, parentID, v["question"]).Scan(&count, &id)
		}
		if count > 1 {
			return fmt.Errorf("ambiguous FAQ upsert key")
		}
		if id != "" {
			_, err := tx.Exec(ctx, `UPDATE faqs SET answer=$2,sort_order=$3,is_active=$4 WHERE id=$1`, id, v["answer"], order, active)
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO faqs(report_type_id,question,answer,sort_order,is_active) VALUES($1,$2,$3,$4,$5)`, parentID, v["question"], v["answer"], order, active)
		return err
	case "admin-users":
		if v["id"] != "" && v["id"] == actorID {
			return problem.New(403, "SELF_ACCOUNT_PROTECTED", "Import tidak dapat mengubah akun sendiri.")
		}
		v["email"] = strings.ToLower(strings.TrimSpace(v["email"]))
		var roleID, roleName string
		if err := tx.QueryRow(ctx, `SELECT id::text,name FROM roles WHERE lower(name)=lower($1)`, v["role_name"]).Scan(&roleID, &roleName); err != nil {
			return fmt.Errorf("unknown user role")
		}
		var id string
		var currentEmail string
		var currentRole string
		var currentActive bool
		if v["id"] != "" {
			_ = tx.QueryRow(ctx, `SELECT u.id::text,u.email,r.name,u.is_active FROM admin_users u JOIN roles r ON r.id=u.role_id WHERE u.id=$1 FOR UPDATE OF u`, v["id"]).Scan(&id, &currentEmail, &currentRole, &currentActive)
		}
		var emailID string
		_ = tx.QueryRow(ctx, `SELECT id::text FROM admin_users WHERE lower(email)=lower($1)`, strings.ToLower(v["email"])).Scan(&emailID)
		if id != "" && emailID != "" && id != emailID {
			return fmt.Errorf("user id and email mismatch")
		}
		if id == "" {
			id = emailID
		}
		if id != "" {
			if id == actorID {
				return problem.New(403, "SELF_ACCOUNT_PROTECTED", "Import tidak dapat mengubah akun sendiri.")
			}
			// Preserve protection for the last active Super Admin.
			leavingSuperAdmin := currentRole == "Super Admin" && currentActive && (roleName != "Super Admin" || !parseBoolean(v["is_active"]))
			if leavingSuperAdmin {
				var count int
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM admin_users u JOIN roles r ON r.id=u.role_id WHERE r.name='Super Admin' AND u.is_active`).Scan(&count); err != nil {
					return err
				}
				if count <= 1 {
					return problem.New(409, "LAST_SUPER_ADMIN", "Super Admin aktif terakhir dilindungi.")
				}
			}
			_, err := tx.Exec(ctx, `UPDATE admin_users SET name=$2,email=lower($3),role_id=$4,is_active=$5,updated_at=now() WHERE id=$1`, id, v["name"], v["email"], roleID, parseBoolean(v["is_active"]))
			return err
		}
		generated := security.Token()[:32]
		hash, err := security.HashPassword(generated)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO admin_users(role_id,name,email,password_hash,is_active,must_change_password,temporary_credential_expires_at) VALUES($1,$2,lower($3),$4,false,true,now()+interval '24 hours')`, roleID, v["name"], v["email"], hash)
		return err
	}
	return invalid("Jenis import tidak didukung.")
}

func sameNullable(current *string, next any) bool {
	if current == nil {
		return next == nil
	}
	v, ok := next.(string)
	return ok && *current == v
}

func sortReportTypeRows(rows []importRow) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i].Data["parent_id"] != "" || rows[i].Data["parent_name"] != "" {
				if rows[j].Data["parent_id"] == "" && rows[j].Data["parent_name"] == "" {
					rows[i], rows[j] = rows[j], rows[i]
				}
			}
		}
	}
}

func importTable(module string) string {
	switch module {
	case "report-types":
		return "report_types"
	case "admin-users":
		return "admin_users"
	case "faqs":
		return "faqs"
	}
	return ""
}

func importRowHash(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, module, id string) (string, error) {
	table := importTable(module)
	if table == "" {
		return "", invalid("Jenis import tidak didukung.")
	}
	var hash string
	err := q.QueryRow(ctx, "SELECT md5(to_jsonb(t)::text) FROM "+table+" t WHERE id=$1", id).Scan(&hash)
	return hash, err
}

func lookupImportAction(ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, module string, row importRow, workbookParents map[string]bool) (string, string, string, error) {
	v := row.Data
	var id string
	var err error
	if v["id"] != "" {
		err = pool.QueryRow(ctx, "SELECT id::text FROM "+importTable(module)+" WHERE id=$1", v["id"]).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", fmt.Errorf("unknown stable ID")
		}
		if err != nil {
			return "", "", "", err
		}
	} else {
		switch module {
		case "report-types":
			if v["parent_id"] == "" && v["parent_name"] != "" {
				return "", "", "", fmt.Errorf("root row cannot have parent name")
			}
			if v["parent_id"] != "" {
				err = pool.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id=$1 AND lower(btrim(name))=lower(btrim($2))`, v["parent_id"], v["name"]).Scan(&id)
			} else if v["parent_name"] != "" {
				var parent string
				if err = pool.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1))`, v["parent_name"]).Scan(&parent); err == nil {
					err = pool.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id=$1 AND lower(btrim(name))=lower(btrim($2))`, parent, v["name"]).Scan(&id)
				} else if errors.Is(err, pgx.ErrNoRows) && workbookParents[normalizeImportKey(v["parent_name"])] {
					err = nil
				}
			} else {
				err = pool.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1))`, v["name"]).Scan(&id)
			}
		case "faqs":
			var parent any
			if v["parent_id"] != "" {
				parent = v["parent_id"]
			} else if v["parent_name"] != "" {
				if err = pool.QueryRow(ctx, `SELECT id::text FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1))`, v["parent_name"]).Scan(&parent); err != nil {
					return "", "", "", err
				}
			}
			if parent == nil {
				err = pool.QueryRow(ctx, `SELECT id::text FROM faqs WHERE report_type_id IS NULL AND lower(btrim(question))=lower(btrim($1))`, v["question"]).Scan(&id)
			} else {
				err = pool.QueryRow(ctx, `SELECT id::text FROM faqs WHERE report_type_id=$1 AND lower(btrim(question))=lower(btrim($2))`, parent, v["question"]).Scan(&id)
			}
		case "admin-users":
			err = pool.QueryRow(ctx, `SELECT id::text FROM admin_users WHERE lower(email)=lower($1)`, v["email"]).Scan(&id)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			id = ""
			err = nil
		}
		if err != nil {
			return "", "", "", err
		}
	}
	if id == "" {
		return "insert", "", "", nil
	}
	hash, err := importRowHash(ctx, pool, module, id)
	if err != nil {
		return "", "", "", err
	}
	return "update", id, hash, nil
}

func normalizeImportKey(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func validateImportReference(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, module string, row importRow, workbookParents map[string]bool) error {
	v := row.Data
	switch module {
	case "admin-users":
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE lower(btrim(name))=lower(btrim($1)))`, v["role_name"]).Scan(&exists); err != nil || !exists {
			return fmt.Errorf("role_name tidak merujuk role yang tersedia.")
		}
	case "report-types":
		if v["parent_id"] != "" {
			var name string
			if err := q.QueryRow(ctx, `SELECT name FROM report_types WHERE id=$1 AND parent_id IS NULL`, v["parent_id"]).Scan(&name); err != nil {
				return fmt.Errorf("parent_id harus merujuk kategori induk.")
			}
			if normalizeImportKey(name) != normalizeImportKey(v["parent_name"]) {
				return fmt.Errorf("parent_id dan parent_name tidak cocok.")
			}
		} else if v["parent_name"] != "" {
			var exists bool
			if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1)))`, v["parent_name"]).Scan(&exists); err != nil {
				return fmt.Errorf("kategori induk tidak dapat divalidasi.")
			}
			if !exists && !workbookParents[normalizeImportKey(v["parent_name"])] {
				return fmt.Errorf("parent_name tidak merujuk kategori induk yang tersedia.")
			}
		}
	case "faqs":
		if v["parent_id"] != "" {
			var name string
			if err := q.QueryRow(ctx, `SELECT name FROM report_types WHERE id=$1 AND parent_id IS NULL`, v["parent_id"]).Scan(&name); err != nil {
				return fmt.Errorf("FAQ hanya dapat memakai kategori induk.")
			}
			if normalizeImportKey(name) != normalizeImportKey(v["parent_name"]) {
				return fmt.Errorf("parent_id dan parent_name tidak cocok.")
			}
		} else if v["parent_name"] != "" {
			var exists bool
			if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM report_types WHERE parent_id IS NULL AND lower(btrim(name))=lower(btrim($1)))`, v["parent_name"]).Scan(&exists); err != nil || !exists {
				return fmt.Errorf("parent_name FAQ tidak merujuk kategori induk.")
			}
		}
	}
	return nil
}
