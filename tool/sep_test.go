package tool

import (
	"encoding/json"
	"testing"
)

func initProtoForTest(t *testing.T) {
	t.Helper()
	err := ParseProtoFile([]string{"./../proto"}, "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConvertColumnOption_Sep(t *testing.T) {
	tests := []struct {
		input    string
		wantSep  string
		wantName string
	}{
		{"Item#Field=no#Sep=|", "|", "Item"},
		{"Item#Field=no", "", "Item"},
		{"Item#Sep=,", ",", "Item"},
		{"Rewards#Field=CfgId_Num#Sep=|", "|", "Rewards"},
	}

	for _, tt := range tests {
		opt := ConvertColumnOption(tt.input)
		if opt == nil {
			t.Fatalf("ConvertColumnOption returned nil for input: %s", tt.input)
		}
		if opt.Name != tt.wantName {
			t.Errorf("input: %s, expected Name=%s, got Name=%s", tt.input, tt.wantName, opt.Name)
		}
		if opt.Sep != tt.wantSep {
			t.Errorf("input: %s, expected Sep=%q, got Sep=%q", tt.input, tt.wantSep, opt.Sep)
		}
	}
}

func TestGetSep(t *testing.T) {
	tests := []struct {
		sep  string
		want string
	}{
		{"", "_"},
		{"|", "|"},
		{",", ","},
		{":", ":"},
	}
	for _, tt := range tests {
		opt := &ColumnOption{Sep: tt.sep}
		got := opt.GetSep()
		if got != tt.want {
			t.Errorf("Sep=%q, expected GetSep()=%q, got %q", tt.sep, tt.want, got)
		}
	}
}

func TestSep_FieldNo_DefaultSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "CfgArg",
		FieldNames: []string{"no"},
	}
	progressTemplateField := FindMessageDescriptor("QuestCfg").FindFieldByName("ProgressTemplate")
	if progressTemplateField == nil {
		t.Fatal("ProgressTemplate field not found")
	}

	result := ConvertFieldValue(progressTemplateField, opt, "1_100")
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if resultMap["CfgId"] != int32(1) {
		t.Errorf("expected CfgId=1, got %v", resultMap["CfgId"])
	}
	if resultMap["Arg"] != int32(100) {
		t.Errorf("expected Arg=100, got %v", resultMap["Arg"])
	}
}

func TestSep_FieldNo_CustomSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "CfgArg",
		FieldNames: []string{"no"},
		Sep:        "|",
	}
	progressTemplateField := FindMessageDescriptor("QuestCfg").FindFieldByName("ProgressTemplate")
	if progressTemplateField == nil {
		t.Fatal("ProgressTemplate field not found")
	}

	result := ConvertFieldValue(progressTemplateField, opt, "1|100")
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if resultMap["CfgId"] != int32(1) {
		t.Errorf("expected CfgId=1, got %v", resultMap["CfgId"])
	}
	if resultMap["Arg"] != int32(100) {
		t.Errorf("expected Arg=100, got %v", resultMap["Arg"])
	}
}

func TestSep_FieldNames_DefaultSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "CfgArg",
		FieldNames: []string{"CfgId", "Arg"},
	}
	progressTemplateField := FindMessageDescriptor("QuestCfg").FindFieldByName("ProgressTemplate")
	if progressTemplateField == nil {
		t.Fatal("ProgressTemplate field not found")
	}

	result := ConvertFieldValue(progressTemplateField, opt, "5_200")
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if resultMap["CfgId"] != int32(5) {
		t.Errorf("expected CfgId=5, got %v", resultMap["CfgId"])
	}
	if resultMap["Arg"] != int32(200) {
		t.Errorf("expected Arg=200, got %v", resultMap["Arg"])
	}
}

func TestSep_FieldNames_CustomSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "CfgArg",
		FieldNames: []string{"CfgId", "Arg"},
		Sep:        "|",
	}
	progressTemplateField := FindMessageDescriptor("QuestCfg").FindFieldByName("ProgressTemplate")
	if progressTemplateField == nil {
		t.Fatal("ProgressTemplate field not found")
	}

	result := ConvertFieldValue(progressTemplateField, opt, "5|200")
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if resultMap["CfgId"] != int32(5) {
		t.Errorf("expected CfgId=5, got %v", resultMap["CfgId"])
	}
	if resultMap["Arg"] != int32(200) {
		t.Errorf("expected Arg=200, got %v", resultMap["Arg"])
	}
}

func TestSep_FieldFull_DefaultSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "ItemNum",
		FieldNames: []string{"full"},
	}
	rewardsField := FindMessageDescriptor("QuestCfg").FindFieldByName("Rewards")
	if rewardsField == nil {
		t.Fatal("Rewards field not found")
	}

	result := ConvertFieldValue(rewardsField, opt, "CfgId_10#Num_99")
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if resultMap["CfgId"] != int32(10) {
		t.Errorf("expected CfgId=10, got %v", resultMap["CfgId"])
	}
	if resultMap["Num"] != int32(99) {
		t.Errorf("expected Num=99, got %v", resultMap["Num"])
	}
}

func TestSep_FieldFull_CustomSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "ItemNum",
		FieldNames: []string{"full"},
		Sep:        "|",
	}
	rewardsField := FindMessageDescriptor("QuestCfg").FindFieldByName("Rewards")
	if rewardsField == nil {
		t.Fatal("Rewards field not found")
	}

	result := ConvertFieldValue(rewardsField, opt, "CfgId|10#Num|99")
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if resultMap["CfgId"] != int32(10) {
		t.Errorf("expected CfgId=10, got %v", resultMap["CfgId"])
	}
	if resultMap["Num"] != int32(99) {
		t.Errorf("expected Num=99, got %v", resultMap["Num"])
	}
}

func TestSep_OnlyFirstLevel_RepeatedMessage(t *testing.T) {
	initProtoForTest(t)

	t.Run("custom Sep=| on first level", func(t *testing.T) {
		opt := &ColumnOption{
			Name:       "ArgValues",
			FieldNames: []string{"no"},
			Sep:        "|",
		}
		argValuesField := FindMessageDescriptor("TestCfgArgValues").FindFieldByName("ArgValues")
		if argValuesField == nil {
			t.Fatal("ArgValues field not found")
		}

		m := make(map[string]any)
		SetFieldValue(m, argValuesField, opt, "1|2,3|4,5;10|20,30|40,50", false)

		argValues, ok := m["ArgValues"].([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", m["ArgValues"])
		}
		if len(argValues) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(argValues))
		}

		elem0 := argValues[0].(map[string]any)
		if elem0["CfgId"] != int32(1) {
			t.Errorf("elem0: expected CfgId=1, got %v", elem0["CfgId"])
		}
		args0 := elem0["Args"].([]any)
		if len(args0) != 2 || args0[0] != int32(2) || args0[1] != int32(3) {
			t.Errorf("elem0: expected Args=[2,3], got %v", args0)
		}
		values0 := elem0["Values"].([]any)
		if len(values0) != 2 || values0[0] != int32(4) || values0[1] != int32(5) {
			t.Errorf("elem0: expected Values=[4,5], got %v", values0)
		}

		elem1 := argValues[1].(map[string]any)
		if elem1["CfgId"] != int32(10) {
			t.Errorf("elem1: expected CfgId=10, got %v", elem1["CfgId"])
		}
		args1 := elem1["Args"].([]any)
		if len(args1) != 2 || args1[0] != int32(20) || args1[1] != int32(30) {
			t.Errorf("elem1: expected Args=[20,30], got %v", args1)
		}
		values1 := elem1["Values"].([]any)
		if len(values1) != 2 || values1[0] != int32(40) || values1[1] != int32(50) {
			t.Errorf("elem1: expected Values=[40,50], got %v", values1)
		}

		jsonData, _ := json.MarshalIndent(m, "", "  ")
		t.Logf("result:\n%s", string(jsonData))
	})

	t.Run("backward compatible without Sep", func(t *testing.T) {
		opt := &ColumnOption{
			Name:       "ArgValues",
			FieldNames: []string{"no"},
		}
		argValuesField := FindMessageDescriptor("TestCfgArgValues").FindFieldByName("ArgValues")
		if argValuesField == nil {
			t.Fatal("ArgValues field not found")
		}

		m := make(map[string]any)
		SetFieldValue(m, argValuesField, opt, "1_2,3_4,5;10_20,30_40,50", false)

		argValues := m["ArgValues"].([]any)
		if len(argValues) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(argValues))
		}

		elem0 := argValues[0].(map[string]any)
		if elem0["CfgId"] != int32(1) {
			t.Errorf("elem0: expected CfgId=1, got %v", elem0["CfgId"])
		}
		args0 := elem0["Args"].([]any)
		if len(args0) != 2 || args0[0] != int32(2) || args0[1] != int32(3) {
			t.Errorf("elem0: expected Args=[2,3], got %v", args0)
		}
		values0 := elem0["Values"].([]any)
		if len(values0) != 2 || values0[0] != int32(4) || values0[1] != int32(5) {
			t.Errorf("elem0: expected Values=[4,5], got %v", values0)
		}

		jsonData, _ := json.MarshalIndent(m, "", "  ")
		t.Logf("result:\n%s", string(jsonData))
	})
}

func TestSep_SubLevelStillUsesDefaultSep(t *testing.T) {
	initProtoForTest(t)

	opt := &ColumnOption{
		Name:       "Rewards",
		FieldNames: []string{"no"},
		Sep:        "|",
	}
	rewardsField := FindMessageDescriptor("QuestCfg").FindFieldByName("Rewards")
	if rewardsField == nil {
		t.Fatal("Rewards field not found")
	}

	m := make(map[string]any)
	SetFieldValue(m, rewardsField, opt, "1|100;2|200", false)

	rewards := m["Rewards"].([]any)
	if len(rewards) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(rewards))
	}

	elem0 := rewards[0].(map[string]any)
	if elem0["CfgId"] != int32(1) {
		t.Errorf("elem0: expected CfgId=1, got %v", elem0["CfgId"])
	}
	if elem0["Num"] != int32(100) {
		t.Errorf("elem0: expected Num=100, got %v", elem0["Num"])
	}

	elem1 := rewards[1].(map[string]any)
	if elem1["CfgId"] != int32(2) {
		t.Errorf("elem1: expected CfgId=2, got %v", elem1["CfgId"])
	}
	if elem1["Num"] != int32(200) {
		t.Errorf("elem1: expected Num=200, got %v", elem1["Num"])
	}

	jsonData, _ := json.MarshalIndent(m, "", "  ")
	t.Logf("result:\n%s", string(jsonData))
}
