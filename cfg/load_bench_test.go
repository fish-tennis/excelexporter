package cfg

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"excelexporter/example/pb"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/encoding/protojson"
)

func buildPbFromJsonFile(t testing.TB, jsonFile, pbFile string) {
	t.Helper()
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("read json failed: %v", err)
	}
	var object map[string]any
	if err = json.Unmarshal(data, &object); err != nil {
		t.Fatalf("unmarshal json failed: %v", err)
	}
	file, err := os.Create(pbFile)
	if err != nil {
		t.Fatalf("create pb file failed: %v", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, row := range object {
		rowData, marshalErr := json.Marshal(row)
		if marshalErr != nil {
			t.Fatalf("marshal row failed: %v", marshalErr)
		}
		msg := &pb.QuestCfg{}
		if unmarshalErr := protojson.Unmarshal(rowData, msg); unmarshalErr != nil {
			t.Fatalf("unmarshal row to proto failed: %v", unmarshalErr)
		}
		if _, marshalErr = protodelim.MarshalTo(writer, msg); marshalErr != nil {
			t.Fatalf("marshal delimited proto failed: %v", marshalErr)
		}
	}
	if err = writer.Flush(); err != nil {
		t.Fatalf("flush pb writer failed: %v", err)
	}
}

func BenchmarkLoadQuestCfgJson(b *testing.B) {
	jsonFile := filepath.Join("..", "data", "json", "Quests.json")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr := NewDataMap[*pb.QuestCfg]()
		if err := mgr.LoadJson(jsonFile); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadQuestCfgPb(b *testing.B) {
	jsonFile := filepath.Join("..", "data", "json", "Quests.json")
	pbFile := filepath.Join(b.TempDir(), "Quests.pb")
	buildPbFromJsonFile(b, jsonFile, pbFile)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr := NewDataMap[*pb.QuestCfg]()
		if err := mgr.LoadPb(pbFile); err != nil {
			b.Fatal(err)
		}
	}
}
