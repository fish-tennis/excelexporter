package tool

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"excelexporter/example/pb"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/encoding/protodelim"
)

func TestExportAll(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "export.proto", "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
	exportOption := &ExportOption{
		DataImportPath:    "./../data/excel/",
		DataExportPath:    []string{"./../data/json/"},
		Md5ExportPath:     []string{"./../data/json/md5.json"},
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
		DataExportPath: []string{"./../data/json/"},
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

func TestConvertSheetObjectGroup(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}

	const sheetName = "TestObjectGroup"

	makeFile := func(t *testing.T, header []interface{}, dataRows ...[]interface{}) *excelize.File {
		t.Helper()
		f := excelize.NewFile()
		idx, err := f.NewSheet(sheetName)
		if err != nil {
			t.Fatalf("NewSheet err: %v", err)
		}
		f.SetActiveSheet(idx)
		if err := f.SetSheetRow(sheetName, "A1", &header); err != nil {
			t.Fatalf("SetSheetRow header err: %v", err)
		}
		for i, r := range dataRows {
			if err := f.SetSheetRow(sheetName, fmt.Sprintf("A%d", i+2), &r); err != nil {
				t.Fatalf("SetSheetRow data err: %v", err)
			}
		}
		return f
	}

	convert := func(t *testing.T, f *excelize.File, exportGroup, defaultGroup string) map[string]any {
		t.Helper()
		exportOption := &ExportOption{
			ExportGroup:  exportGroup,
			DefaultGroup: defaultGroup,
		}
		opt := &SheetOption{
			SheetName:   sheetName,
			MessageName: "LevelExp",
			MgrType:     "object",
		}
		result, err := ConvertSheet(exportOption, f, opt)
		if err != nil {
			t.Fatalf("ConvertSheet err: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result)
		}
		return m
	}

	assertKey := func(t *testing.T, m map[string]any, key string, want bool) {
		t.Helper()
		_, ok := m[key]
		if ok != want {
			t.Errorf("key %q exist=%v, want %v (map=%v)", key, ok, want, m)
		}
	}

	t.Run("ExportGroup_s_keep_both", func(t *testing.T) {
		f := makeFile(t,
			[]interface{}{"key", "value", "group"},
			[]interface{}{"Level", "1", "cs"},
			[]interface{}{"NeedExp", "100", "s"},
		)
		defer func() { _ = f.Close() }()
		m := convert(t, f, "s", "cs")
		assertKey(t, m, "Level", true)
		assertKey(t, m, "NeedExp", true)
	})

	t.Run("ExportGroup_c_filter_NeedExp", func(t *testing.T) {
		f := makeFile(t,
			[]interface{}{"key", "value", "group"},
			[]interface{}{"Level", "1", "cs"},
			[]interface{}{"NeedExp", "100", "s"},
		)
		defer func() { _ = f.Close() }()
		m := convert(t, f, "c", "cs")
		assertKey(t, m, "Level", true)
		assertKey(t, m, "NeedExp", false)
	})

	t.Run("ExportGroup_empty_keep_both", func(t *testing.T) {
		f := makeFile(t,
			[]interface{}{"key", "value", "group"},
			[]interface{}{"Level", "1", "cs"},
			[]interface{}{"NeedExp", "100", "s"},
		)
		defer func() { _ = f.Close() }()
		m := convert(t, f, "", "cs")
		assertKey(t, m, "Level", true)
		assertKey(t, m, "NeedExp", true)
	})

	t.Run("no_group_column_use_DefaultGroup", func(t *testing.T) {
		f := makeFile(t,
			[]interface{}{"key", "value"},
			[]interface{}{"Level", "1"},
			[]interface{}{"NeedExp", "100"},
		)
		defer func() { _ = f.Close() }()
		m := convert(t, f, "s", "cs")
		assertKey(t, m, "Level", true)
		assertKey(t, m, "NeedExp", true)
	})

	t.Run("empty_group_cell_use_DefaultGroup", func(t *testing.T) {
		f := makeFile(t,
			[]interface{}{"key", "value", "group"},
			[]interface{}{"Level", "1", ""},
			[]interface{}{"NeedExp", "100", "s"},
		)
		defer func() { _ = f.Close() }()
		m := convert(t, f, "c", "cs")
		assertKey(t, m, "Level", true)
		assertKey(t, m, "NeedExp", false)
	})
}
