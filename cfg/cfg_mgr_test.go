package cfg

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	"excelexporter/example/pb"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)

func TestResolveDataFile(t *testing.T) {
	tempDir := t.TempDir()
	jsonFile := filepath.Join(tempDir, "item.json")
	pbFile := filepath.Join(tempDir, "item.pb")
	if err := os.WriteFile(pbFile, []byte{1, 2, 3}, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	resolved := ResolveDataFile(jsonFile)
	if resolved != pbFile {
		t.Fatalf("expected %s, got %s", pbFile, resolved)
	}
}

func TestDataMapLoadPb(t *testing.T) {
	buffer := bufio.NewWriterSize(nil, 0)
	tmpFile := filepath.Join(t.TempDir(), "cfg.pb")
	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	buffer = bufio.NewWriter(file)
	msg1 := &pb.QuestCfg{CfgId: 1, Name: "A"}
	msg2 := &pb.QuestCfg{CfgId: 2, Name: "B"}
	if _, err = protodelim.MarshalTo(buffer, msg1); err != nil {
		t.Fatal(err)
	}
	if _, err = protodelim.MarshalTo(buffer, msg2); err != nil {
		t.Fatal(err)
	}
	if err = buffer.Flush(); err != nil {
		t.Fatal(err)
	}
	file.Close()

	mgr := NewDataMap[*pb.QuestCfg]()
	if err = mgr.LoadPb(tmpFile); err != nil {
		t.Fatal(err)
	}
	if len(mgr.cfgs) != 2 || mgr.cfgs[1].Name != "A" || mgr.cfgs[2].Name != "B" {
		t.Fatalf("unexpected map data: %+v", mgr.cfgs)
	}
}

func TestDataSliceLoadPb(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "cfg.pb")
	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	buffer := bufio.NewWriter(file)
	msg1 := &pb.QuestCfg{CfgId: 1, Name: "A"}
	msg2 := &pb.QuestCfg{CfgId: 2, Name: "B"}
	if _, err = protodelim.MarshalTo(buffer, msg1); err != nil {
		t.Fatal(err)
	}
	if _, err = protodelim.MarshalTo(buffer, msg2); err != nil {
		t.Fatal(err)
	}
	if err = buffer.Flush(); err != nil {
		t.Fatal(err)
	}
	file.Close()

	mgr := &DataSlice[*pb.QuestCfg]{}
	if err = mgr.LoadPb(tmpFile); err != nil {
		t.Fatal(err)
	}
	if mgr.Len() != 2 || mgr.GetCfg(0).Name != "A" || mgr.GetCfg(1).Name != "B" {
		t.Fatalf("unexpected slice data: %+v", mgr.cfgs)
	}
}

func TestLoadObjectFromPb(t *testing.T) {
	source := &pb.QuestCfg{CfgId: 1, Name: "A"}
	pbBytes, err := proto.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	tempFile := filepath.Join(t.TempDir(), "obj.pb")
	if err = os.WriteFile(tempFile, pbBytes, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	dst := &pb.QuestCfg{}
	if err = LoadObjectFromPb(tempFile, dst); err != nil {
		t.Fatal(err)
	}
	if dst.Name != "A" || dst.CfgId != 1 {
		t.Fatalf("unexpected object data: %+v", dst)
	}
}
