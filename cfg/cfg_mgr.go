package cfg

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
)

type CfgData interface {
	GetCfgId() int32
}

// map类型的配置数据管理
type DataMap[E CfgData] struct {
	cfgs map[int32]E
}

func NewDataMap[E CfgData]() *DataMap[E] {
	return &DataMap[E]{
		cfgs: make(map[int32]E),
	}
}

func (this *DataMap[E]) GetCfg(cfgId int32) E {
	return this.cfgs[cfgId]
}

func (this *DataMap[E]) Range(f func(e E) bool) {
	for _, cfg := range this.cfgs {
		if !f(cfg) {
			return
		}
	}
}

// 加载配置数据,支持json和csv
func (this *DataMap[E]) Load(fileName string) error {
	if this.cfgs == nil {
		this.cfgs = make(map[int32]E)
	}
	if strings.HasSuffix(fileName, ".json") {
		return this.LoadJson(fileName)
	}
	return errors.New("unsupported file type")
}

// 从json文件加载数据
func (this *DataMap[E]) LoadJson(fileName string) error {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		slog.Error("LoadJsonErr", "fileName", fileName, "err", err)
		return err
	}
	cfgMap := make(map[int32]E)
	err = json.Unmarshal(fileData, &cfgMap)
	if err != nil {
		slog.Error("LoadJsonErr", "fileName", fileName, "err", err)
		return err
	}
	this.cfgs = cfgMap
	slog.Info("LoadJson", "fileName", fileName, "count", len(this.cfgs))
	return nil
}

// slice类型的配置数据管理
type DataSlice[E any] struct {
	cfgs []E
}

func (this *DataSlice[E]) Len() int {
	return len(this.cfgs)
}

func (this *DataSlice[E]) GetCfg(index int) E {
	return this.cfgs[index]
}

func (this *DataSlice[E]) Range(f func(e E) bool) {
	for _, cfg := range this.cfgs {
		if !f(cfg) {
			return
		}
	}
}

// 加载配置数据,支持json和csv
func (this *DataSlice[E]) Load(fileName string) error {
	if strings.HasSuffix(fileName, ".json") {
		return this.LoadJson(fileName)
	}
	return errors.New("unsupported file type")
}

// 从json文件加载数据
func (this *DataSlice[E]) LoadJson(fileName string) error {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		slog.Error("LoadJsonErr", "fileName", fileName, "err", err)
		return err
	}
	var cfgList []E
	err = json.Unmarshal(fileData, &cfgList)
	if err != nil {
		slog.Error("LoadJsonErr", "fileName", fileName, "err", err)
		return err
	}
	this.cfgs = cfgList
	slog.Info("LoadJson", "fileName", fileName, "count", len(this.cfgs))
	this.checkDuplicateCfgId(fileName)
	return nil
}

// 如果配置项是CfgData,检查id是否重复
func (this *DataSlice[E]) checkDuplicateCfgId(fileName string) {
	for i := 0; i < len(this.cfgs); i++ {
		cfgDataI, ok := any(this.cfgs[i]).(CfgData)
		if !ok {
			return
		}
		for j := i + 1; j < len(this.cfgs); j++ {
			cfgDataJ := any(this.cfgs[j]).(CfgData)
			if cfgDataI.GetCfgId() == cfgDataJ.GetCfgId() {
				slog.Error("duplicate id", "fileName", fileName, "id", cfgDataI.GetCfgId())
			}
		}
	}
}
