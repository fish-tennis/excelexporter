package example

import (
	"encoding/json"
	"excelexporter/example/pb"
	"os"
	"testing"
)

func TestImport(t *testing.T) {
	m := make(map[int32]*pb.QuestCfg)
	fileData, err := os.ReadFile("./../data/json/questcfg.json")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(fileData, &m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", m)
	jsonData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", string(jsonData))
}
