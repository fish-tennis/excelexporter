package example

import (
	"encoding/json"
	"excelexporter/cfg"
	"excelexporter/example/pb"
	"os"
	"sync"
	"testing"
	"time"
)

// 加载单个配置文件
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

// 加载所有配置文件(由TestExportAll导出的文件)
func TestImportAll(t *testing.T) {
	err := cfg.Load("./../data/json/", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// 并发加载
func TestImportConcurrency(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx == 5 {
				time.Sleep(time.Second)
			}
			err := cfg.Load("./../data/json/", nil, nil)
			if err != nil {
				t.Logf("%v Load err:%v", idx, err)
			} else {
				t.Logf("%v load ok", idx)
			}
		}(i)
	}
	wg.Wait()
}

// reload
func TestReload(t *testing.T) {
	loadedFiles := make(map[string]struct{})
	preprocessFn := func(mgr any, mgrName, messageName, fileName string) error {
		t.Logf("preprocess: %v fileName:%v", mgrName, fileName)
		loadedFiles[fileName] = struct{}{}
		return nil
	}
	filterFn := func(fileName string) bool {
		if _, ok := loadedFiles[fileName]; ok {
			return false
		}
		return true
	}
	for i := 0; i < 2; i++ {
		err := cfg.Load("./../data/json/", preprocessFn, filterFn)
		if err != nil {
			t.Fatal(err)
		}
	}
}
