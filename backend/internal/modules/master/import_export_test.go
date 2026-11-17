package master

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWorkbookImportValidation(t *testing.T) {
	data, err := workbook("report-types", [][]string{
		{"id", "parent_id", "parent_name", "name", "description", "is_active"},
		{"", "", "", "Parent", "category", "true"},
		{"", "", "Parent", "Child", "detail", "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := parseWorkbook(data, "report-types")
	if err != nil || len(plan.Rows) != 2 {
		t.Fatalf("valid workbook rejected: rows=%d err=%v", len(plan.Rows), err)
	}
	if len(plan.Rows[0].Errors) != 0 || len(plan.Rows[1].Errors) != 0 {
		t.Fatalf("unexpected row errors: %+v", plan.Rows)
	}

	duplicate, err := workbook("faqs", [][]string{
		{"id", "parent_id", "parent_name", "question", "answer", "sort_order", "is_active"},
		{"", "", "", "Same question", "Answer 1", "0", "true"},
		{"", "", "", "Same question", "Answer 2", "1", "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = parseWorkbook(duplicate, "faqs")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Rows[1].Errors) == 0 {
		t.Fatal("duplicate business key was not reported")
	}
}

func TestWorkbookRejectsFormulaAndUnexpectedHeader(t *testing.T) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "Import"); err != nil {
		t.Fatal(err)
	}
	headers := importColumns["faqs"]
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		if err := f.SetCellStr("Import", cell, header); err != nil {
			t.Fatal(err)
		}
	}
	for index, value := range []string{"", "", "", "Question", "Answer", "0", "true"} {
		cell, _ := excelize.CoordinatesToCellName(index+1, 2)
		if err := f.SetCellStr("Import", cell, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SetCellFormula("Import", "D2", `"=HYPERLINK(\"https://bad.test\")"`); err != nil {
		t.Fatal(err)
	}
	buffer, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	plan, err := parseWorkbook(buffer.Bytes(), "faqs")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Rows) != 1 || len(plan.Rows[0].Errors) == 0 {
		t.Fatal("formula cell was accepted")
	}
	wrong, err := workbook("faqs", [][]string{{"question", "answer"}, {"Q", "A"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parseWorkbook(wrong, "faqs"); err == nil {
		t.Fatal("unexpected workbook header accepted")
	}
}

func TestExportWorkbookWritesUntrustedFormulaPrefixAsText(t *testing.T) {
	data, err := workbook("admin-users", [][]string{{"name"}, {"=HYPERLINK(\"https://attacker.invalid\")"}})
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	formula, err := f.GetCellFormula("Import", "A2")
	if err != nil {
		t.Fatal(err)
	}
	if formula != "" {
		t.Fatalf("exported untrusted text became formula: %q", formula)
	}
	value, err := f.GetCellValue("Import", "A2")
	if err != nil || value != "=HYPERLINK(\"https://attacker.invalid\")" {
		t.Fatalf("exported text changed: %q %v", value, err)
	}
}
