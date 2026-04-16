package tool

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"excelexporter/example/pb"
	"google.golang.org/protobuf/encoding/protodelim"
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
		CodeExportFiles:   []string{"./../cfg/data_mgr.go"},
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

func TestMergeRepeatedFields(t *testing.T) {
	columnOpts := []*ColumnOption{
		{Name: "CfgId", Merge: false},
		{Name: "Rewards", Merge: true, MergeKey: "__merge_Rewards_1__"},
		{Name: "Rewards", Merge: true, MergeKey: "__merge_Rewards_2__"},
		{Name: "Rewards", Merge: true, MergeKey: "__merge_Rewards_3__"},
	}

	rowValue := map[string]any{
		"CfgId":               1,
		"__merge_Rewards_1__": map[string]any{"CfgId": 1, "Num": 100},
		"__merge_Rewards_2__": map[string]any{"CfgId": 2, "Num": 200},
		"__merge_Rewards_3__": map[string]any{"CfgId": 3, "Num": 300},
	}

	mergeRepeatedFields(rowValue, columnOpts)

	rewards, ok := rowValue["Rewards"].([]any)
	if !ok {
		t.Fatalf("expected Rewards to be []any, got %T", rowValue["Rewards"])
	}
	if len(rewards) != 3 {
		t.Fatalf("expected 3 rewards, got %d", len(rewards))
	}

	if _, exists := rowValue["__merge_Rewards_1__"]; exists {
		t.Error("merge key should be deleted after merge")
	}

	jsonData, _ := json.MarshalIndent(rowValue, "", "  ")
	t.Logf("merged result:\n%s", string(jsonData))
}

func TestConvertColumnOptionMerge(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Rewards#Merge", true},
		{"Rewards#Merge#Field=CfgId_Num", true},
		{"Rewards#Field=CfgId_Num#Merge", true},
		{"Rewards", false},
		{"Rewards#Field=CfgId_Num", false},
	}

	for _, tt := range tests {
		opt := ConvertColumnOption(tt.input)
		if opt == nil {
			t.Fatalf("ConvertColumnOption returned nil for input: %s", tt.input)
		}
		if opt.Merge != tt.expected {
			t.Errorf("input: %s, expected Merge=%v, got Merge=%v", tt.input, tt.expected, opt.Merge)
		}
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

func TestMarshalToProtoBinary(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
	sliceData := []any{
		map[string]any{"Level": 1, "NeedExp": 100},
		map[string]any{"Level": 2, "NeedExp": 300},
	}
	opt := &SheetOption{
		MessageName: "LevelExp",
		MgrType:     "slice",
	}
	pbBytes, err := marshalToProtoBinary(sliceData, opt)
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(bytes.NewReader(pbBytes))
	count := 0
	for {
		levelExp := &pb.LevelExp{}
		err = protodelim.UnmarshalFrom(reader, levelExp)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("unexpected decoded count: %d", count)
	}
}
