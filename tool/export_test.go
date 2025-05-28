package tool

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestExportAll(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "export.proto", "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
	exportOption := &ExportOption{
		DataImportPath:    "./../data/excel/",
		DataExportPath:    "./../data/json/",
		Md5ExportPath:     "./../data/json/md5.json",
		CodeTemplatePath:  "./../template/",
		CodeExportPath:    "./../cfg/",
		CodeTemplateFiles: []string{"data_mgr.go.template"},
		ExportGroup:       "s",
		DefaultGroup:      "cs",
	}
	excelFileName := "all.xlsx"
	err = ExportAll(exportOption, excelFileName, "ExportCfg")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExport(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
	exportOption := &ExportOption{
		DataImportPath: "./../data/excel/",
		DataExportPath: "./../data/json/",
	}
	excelFileName := "questcfg.xlsx"
	opts := []*SheetOption{
		{
			SheetName:   "questcfg",
			MessageName: "QuestCfg",
		},
	}
	err = ExportExcelToJson(exportOption, excelFileName, opts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportJson(t *testing.T) {
	m := make(map[any]any)
	for i := 0; i < 10; i++ {
		m[i] = fmt.Sprintf("str%v", i)
	}
	jsonData, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", string(jsonData))
}

func TestProtoLoad(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "export.proto", "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
	msg := FindMessageDescriptor("QuestCfg")
	for idx, field := range msg.GetFields() {
		typeStr := field.GetType().String()
		if field.IsRepeated() {
			if field.IsMap() {
				keyType := field.GetMapKeyType()
				valueType := field.GetMapValueType()
				typeStr = fmt.Sprintf("map[%v]%v", keyType.GetType(), valueType.GetType())
			} else {
				typeStr = fmt.Sprintf("[]%v", field.GetType())
			}
		} else if field.IsExtension() {
			typeStr = "ext"
		}
		fmt.Printf("  Field%v: %s,%s,%s,%s Type: %v\n",
			idx,
			field.GetName(),
			field.GetFullyQualifiedName(),
			field.GetJSONName(),
			field.GetFullyQualifiedJSONName(),
			typeStr,
		)
	}
}
