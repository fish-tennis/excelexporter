package cfg

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
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
	if strings.HasSuffix(fileName, ".pb") {
		return this.LoadPb(fileName)
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

// 从pb文件加载数据
func (this *DataMap[E]) LoadPb(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		slog.Error("LoadPbErr", "fileName", fileName, "err", err)
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	cfgMap := make(map[int32]E)
	for {
		cfg, newErr := newElement[E]()
		if newErr != nil {
			slog.Error("LoadPbErr", "fileName", fileName, "err", newErr)
			return newErr
		}
		msg, ok := any(cfg).(proto.Message)
		if !ok {
			return fmt.Errorf("type %T does not implement proto.Message", cfg)
		}
		err = protodelim.UnmarshalFrom(reader, msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("LoadPbErr", "fileName", fileName, "err", err)
			return err
		}
		cfgMap[cfg.GetCfgId()] = cfg
	}
	this.cfgs = cfgMap
	slog.Info("LoadPb", "fileName", fileName, "count", len(this.cfgs))
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
	if strings.HasSuffix(fileName, ".pb") {
		return this.LoadPb(fileName)
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

// 从pb文件加载数据
func (this *DataSlice[E]) LoadPb(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		slog.Error("LoadPbErr", "fileName", fileName, "err", err)
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var cfgList []E
	for {
		cfg, newErr := newElement[E]()
		if newErr != nil {
			slog.Error("LoadPbErr", "fileName", fileName, "err", newErr)
			return newErr
		}
		msg, ok := any(cfg).(proto.Message)
		if !ok {
			return fmt.Errorf("type %T does not implement proto.Message", cfg)
		}
		err = protodelim.UnmarshalFrom(reader, msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("LoadPbErr", "fileName", fileName, "err", err)
			return err
		}
		cfgList = append(cfgList, cfg)
	}
	this.cfgs = cfgList
	slog.Info("LoadPb", "fileName", fileName, "count", len(this.cfgs))
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

func LoadObjectFromJson(fileName string, obj proto.Message) error {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		slog.Error("LoadObjectFromFileErr", "fileName", fileName, "err", err)
		return err
	}
	err = protojson.Unmarshal(fileData, obj)
	if err != nil {
		slog.Error("LoadObjectFromFileErr", "fileName", fileName, "err", err)
		return err
	}
	return nil
}

func LoadObjectFromPb(fileName string, obj proto.Message) error {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		slog.Error("LoadObjectFromFileErr", "fileName", fileName, "err", err)
		return err
	}
	err = proto.Unmarshal(fileData, obj)
	if err != nil {
		slog.Error("LoadObjectFromFileErr", "fileName", fileName, "err", err)
		return err
	}
	return nil
}

func ResolveDataFile(fileName string) string {
	return EnsureDataFileExt(fileName, DataFileExt)
}

func EnsureDataFileExt(fileName, ext string) string {
	if strings.HasSuffix(fileName, ext) {
		return fileName
	}
	return strings.TrimSuffix(fileName, filepath.Ext(fileName)) + ext
}

func newElement[E any]() (E, error) {
	var zero E
	t := reflect.TypeOf(zero)
	if t == nil {
		return zero, errors.New("invalid nil element type")
	}
	if t.Kind() != reflect.Ptr {
		return zero, fmt.Errorf("element type must be pointer, got %v", t)
	}
	v := reflect.New(t.Elem()).Interface()
	elem, ok := v.(E)
	if !ok {
		return zero, fmt.Errorf("failed to cast element to generic type %T", zero)
	}
	return elem, nil
}

type loadable interface {
	Load(filename string) error
}

func LoadConfig[L loadable](filter func(string) bool, fileName, dataDir string, newFn func() L, target *L) error {
	if filter != nil && !filter(fileName) {
		return nil
	}
	resolvedFileName := ResolveDataFile(dataDir + fileName)
	tmp := newFn()
	if err := tmp.Load(resolvedFileName); err != nil {
		return err
	}
	*target = tmp
	return nil
}

func LoadObjectConfig[T proto.Message](filter func(string) bool, fileName, dataDir string, newFn func() T, target *T) error {
	if filter != nil && !filter(fileName) {
		return nil
	}
	resolvedFileName := ResolveDataFile(dataDir + fileName)
	tmp := newFn()
	var err error
	if strings.HasSuffix(resolvedFileName, ".pb") {
		err = LoadObjectFromPb(resolvedFileName, tmp)
	} else {
		err = LoadObjectFromJson(resolvedFileName, tmp)
	}
	if err != nil {
		return err
	}
	*target = tmp
	return nil
}

func Process[T any](fn func(T) error, data T) error {
	if fn != nil {
		return fn(data)
	}
	return nil
}
