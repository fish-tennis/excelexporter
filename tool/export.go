package tool

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

// 导出设置项
type ExportOption struct {
	DataImportPath string `yaml:"DataImportPath"` // Excel导入目录(excel所在目录)
	DataExportPath   string `yaml:"DataExportPath"`   // 数据导出目录
	Md5ExportPath    string `yaml:"Md5ExportPath"`    // 可选项:导出md5文件完整路径(所有格式合并)
	JsonMd5ExportPath string `yaml:"JsonMd5ExportPath"` // 可选项:导出json文件的md5完整路径
	PbMd5ExportPath  string `yaml:"PbMd5ExportPath"`  // 可选项:导出pb文件的md5完整路径

	CodeTemplatePath  string   `yaml:"CodeTemplatePath"`  // 代码模板目录
	CodeTemplateFiles []string `yaml:"CodeTemplateFiles"` // 代码模板
	CodeExportFiles   []string `yaml:"CodeExportFiles"`   // 代码模板导出文件名,和CodeTemplateFiles一一对应

	ExportGroup  string `yaml:"ExportGroup"`  // 导出分组标记 c s cs
	DefaultGroup string `yaml:"DefaultGroup"` // 默认的分组标记

	ExportAllExcelFile string `yaml:"ExportAllExcelFile"` // 导出总表的文件名
	ExportAllSheet     string `yaml:"ExportAllSheet"`     // 导出总表的sheet名

	ProtoPath  string   `yaml:"ProtoPath"`  // proto所在目录
	ProtoFiles []string `yaml:"ProtoFiles"` // 需要解析的proto文件

	ExportFormats []string `yaml:"ExportFormats"` // 导出格式: json pb
}

type ExportInfo struct {
	MgrData     any
	SheetOption *SheetOption
	MergeName   string
	CodeComment string
	//ExportFileName string // 导出的文件名
}

// 从一个总表导出所有的配置表
func ExportAll(exportOption *ExportOption, exportExcelFileName, exportSheetName string) error {
	checkExportOption(exportOption)
	f, err := excelize.OpenFile(exportOption.DataImportPath + exportExcelFileName)
	if err != nil {
		color.Red("open excel err:%v file:%v", err, exportOption.DataImportPath+exportExcelFileName)
		return err
	}
	sheets, err := parseExportSheets(exportOption.DataImportPath+exportExcelFileName, exportSheetName)
	f.Close()
	if err != nil {
		color.Red("ConvertSheetErr err:%v sheet:%v", err, exportSheetName)
		return err
	}
	fmt.Println(fmt.Sprintf("parseExportSheets excel:%v sheet:%v count:%v", exportExcelFileName, exportSheetName, len(sheets)))
	getMapValueFn := func(strMap map[string]any, key, defaultValue string) string {
		if v, ok := strMap[key]; ok {
			str := strings.TrimSpace(v.(string))
			if str != "" {
				return str
			}
		}
		return defaultValue
	}
	generateInfo := &GenerateInfo{}
	for idx, templateFile := range exportOption.CodeTemplateFiles {
		generateInfo.TemplateFiles = append(generateInfo.TemplateFiles, exportOption.CodeTemplatePath+templateFile)
		generateInfo.ExportFiles = append(generateInfo.ExportFiles, exportOption.CodeExportFiles[idx])
	}
	exportInfoMap := make(map[string]*ExportInfo)
	orderNames := make([]string, 0)
	refCheckMap := make(map[string]*ExportInfo)
	for _, v := range sheets {
		exportCfg := v.(map[string]any)
		excelName := getMapValueFn(exportCfg, "Excel", "")
		sheetName := getMapValueFn(exportCfg, "Sheet", "")
		sheetExportGroup := getMapValueFn(exportCfg, "Group", exportOption.DefaultGroup)
		if sheetExportGroup == "" {
			sheetExportGroup = exportOption.DefaultGroup
		}
		if exportOption.ExportGroup != "" && !strings.Contains(sheetExportGroup, exportOption.ExportGroup) {
			continue
		}
		sheetOption := &SheetOption{
			ExcelName:   excelName,
			SheetName:   sheetName,
			MessageName: getMapValueFn(exportCfg, "Message", sheetName),
			MgrType:     getMapValueFn(exportCfg, "MgrType", "map"),
		}
		if sheetOption.MgrType == "map" {
			sheetOption.MapKeyName = getMapValueFn(exportCfg, "MapKey", "")
		}
		codeComment := getMapValueFn(exportCfg, "CodeComment", "")
		mergeName := getMapValueFn(exportCfg, "Merge", "")
		excelFileName := getMapValueFn(exportCfg, "Excel", "")
		//exportFileName := getMapValueFn(exportCfg, "ExportName", sheetName)
		f, err = excelize.OpenFile(exportOption.DataImportPath + excelFileName)
		if err != nil {
			color.Red("open excel err:%v file:%v", err, exportOption.DataImportPath+excelFileName)
			return err
		}
		sheetData, err := ConvertSheet(exportOption, f, sheetOption)
		f.Close()
		if err != nil {
			color.Red("ConvertSheetErr err:%v sheet:%v", err, sheetOption.SheetName)
			return err
		}
		fmt.Println(fmt.Sprintf("parse excel:%v sheet:%v", excelFileName, sheetOption.SheetName))
		if mergeName == "" {
			exportInfoMap[excelName+"."+sheetName] = &ExportInfo{
				MgrData:     sheetData,
				SheetOption: sheetOption,
				CodeComment: codeComment,
				//ExportFileName: exportFileName,
			}
			orderNames = append(orderNames, excelName+"."+sheetName)
			refCheckMap[sheetName] = exportInfoMap[excelName+"."+sheetName]
		} else {
			if mergeInfo, ok := exportInfoMap[mergeName]; ok {
				mergeData, err := mergeMgrData(mergeInfo.MgrData, sheetData)
				if err != nil {
					color.Red("mergeMgrDataErr excel:%v sheet:%v merge:%v err:%v",
						excelFileName, sheetName, mergeName, err)
					return err
				}
				mergeInfo.MgrData = mergeData
				fmt.Println(fmt.Sprintf("merge:%v excel:%v sheet:%v", mergeName, excelFileName, sheetOption.SheetName))
			} else {
				exportInfoMap[mergeName] = &ExportInfo{
					MgrData:     sheetData,
					SheetOption: sheetOption,
					MergeName:   mergeName,
					CodeComment: codeComment,
				}
				orderNames = append(orderNames, mergeName)
				refCheckMap[mergeName] = exportInfoMap[mergeName]
			}
		}
	}

	enabledFormats := getEnabledExportFormats(exportOption.ExportFormats)
	// 导出
	md5Map := make(map[string]string)
	jsonMd5Map := make(map[string]string)
	pbMd5Map := make(map[string]string)
	for _, exportInfo := range exportInfoMap {
		jsonData, err := json.MarshalIndent(exportInfo.MgrData, "", "  ")
		if err != nil {
			color.Red("ExportAllErr exportFileName:%v merge:%v err:%v",
				exportInfo.SheetOption.ExportFileName, exportInfo.MergeName, err)
			return err
		}
		exportFileNameWithoutExt := ""
		if exportInfo.MergeName == "" {
			exportFileNameWithoutExt = exportInfo.SheetOption.SheetName
		} else {
			exportFileNameWithoutExt = exportInfo.MergeName
		}
		if enabledFormats["json"] {
			exportFileName := fmt.Sprintf("%s.json", exportFileNameWithoutExt)
			err = os.WriteFile(filepath.Join(exportOption.DataExportPath, exportFileName), jsonData, os.ModePerm)
			if err != nil {
				color.Red("ExportAllErr exportFileName:%v merge:%v err:%v",
					exportFileName, exportInfo.MergeName, err)
				return err
			}
			md5Str := GetMd5(jsonData)
			md5Map[exportFileName] = md5Str
			jsonMd5Map[exportFileName] = md5Str
		}
		if enabledFormats["pb"] {
			pbData, pbErr := marshalToProtoBinary(exportInfo.MgrData, exportInfo.SheetOption)
			if pbErr != nil {
				color.Red("marshalToProtoBinaryErr exportFileName:%v merge:%v err:%v",
					exportFileNameWithoutExt, exportInfo.MergeName, pbErr)
				return pbErr
			}
			exportFileName := fmt.Sprintf("%s.pb", exportFileNameWithoutExt)
			err = os.WriteFile(filepath.Join(exportOption.DataExportPath, exportFileName), pbData, os.ModePerm)
			if err != nil {
				color.Red("ExportAllErr exportFileName:%v merge:%v err:%v",
					exportFileName, exportInfo.MergeName, err)
				return err
			}
			md5Str := GetMd5(pbData)
			md5Map[exportFileName] = md5Str
			pbMd5Map[exportFileName] = md5Str
		}
	}
	if exportOption.Md5ExportPath != "" {
		jsonMd5Data, err := json.MarshalIndent(md5Map, "", "  ")
		if err != nil {
			color.Red("export md5 err:%v", err)
			return err
		}
		err = os.WriteFile(exportOption.Md5ExportPath, jsonMd5Data, os.ModePerm)
		if err != nil {
			color.Red("export md5 err:%v", err)
			return err
		}
	}
	if exportOption.JsonMd5ExportPath != "" {
		jsonMd5Data, err := json.MarshalIndent(jsonMd5Map, "", "  ")
		if err != nil {
			color.Red("export json md5 err:%v", err)
			return err
		}
		err = os.WriteFile(exportOption.JsonMd5ExportPath, jsonMd5Data, os.ModePerm)
		if err != nil {
			color.Red("export json md5 err:%v", err)
			return err
		}
	}
	if exportOption.PbMd5ExportPath != "" {
		pbMd5Data, err := json.MarshalIndent(pbMd5Map, "", "  ")
		if err != nil {
			color.Red("export pb md5 err:%v", err)
			return err
		}
		err = os.WriteFile(exportOption.PbMd5ExportPath, pbMd5Data, os.ModePerm)
		if err != nil {
			color.Red("export pb md5 err:%v", err)
			return err
		}
	}

	// 生成代码
	for _, name := range orderNames {
		exportInfo := exportInfoMap[name]
		exportFileName := ""
		if exportInfo.MergeName == "" {
			exportFileName = fmt.Sprintf("%s.json", exportInfo.SheetOption.SheetName)
		} else {
			exportFileName = fmt.Sprintf("%s.json", exportInfo.MergeName)
		}
		mgrName := exportInfo.SheetOption.MessageName + "s"
		if exportInfo.MergeName != "" {
			mgrName = exportInfo.MergeName
		}
		mgrInfo := &DataMgrInfo{
			MessageName: exportInfo.SheetOption.MessageName,
			MgrName:     mgrName,
			MgrType:     exportInfo.SheetOption.MgrType,
			MapKeyType:  exportInfo.SheetOption.MapKeyName,
			FileName:    exportFileName,
			CodeComment: exportInfo.CodeComment,
		}
		if mgrInfo.MgrType == "map" {
			mgrInfo.MapKeyType = exportInfo.SheetOption.MapKeyType
			if mgrInfo.MapKeyType == "int32" || mgrInfo.MapKeyType == "int64" || mgrInfo.MapKeyType == "uint64" {
				mgrInfo.MapKeyType = "int"
			}
		}
		generateInfo.AddDataMgrInfo(mgrInfo)
	}
	err = GenerateCode(generateInfo)
	if err != nil {
		color.Red("GenerateCodeErr err:%v", err)
		return err
	}
	// ref功能,检查数据关联
	for _, exportInfo := range exportInfoMap {
		for _, columnOption := range exportInfo.SheetOption.ColumnOpts {
			if columnOption.Ref == "" {
				continue
			}
			sheetName := exportInfo.SheetOption.SheetName
			sheetData := exportInfo.MgrData
			refInfo, ok := refCheckMap[columnOption.Ref]
			if !ok {
				color.Red("ref not exists sheetName:%v column:%v ref:%v", sheetName, columnOption.Name, columnOption.Ref)
				continue
			}
			refKeyName := refInfo.SheetOption.MapKeyName
			rangeSheetData(sheetData, columnOption.Name, refKeyName, func(checkId int32) {
				if refSheetDataMap, ok := refInfo.MgrData.(map[int32]any); ok {
					if _, ok := refSheetDataMap[checkId]; !ok {
						color.Red("ref ERROR sheetName:%v column:%v ref:%v checkId:%v", sheetName, columnOption.Name, columnOption.Ref, checkId)
					}
				}
			})
		}
	}
	return nil
}

func getEnabledExportFormats(formats []string) map[string]bool {
	result := map[string]bool{
		"json": true,
	}
	if len(formats) == 0 {
		return result
	}
	result = make(map[string]bool)
	for _, format := range formats {
		f := strings.ToLower(strings.TrimSpace(format))
		if f == "json" || f == "pb" {
			result[f] = true
		}
	}
	if len(result) == 0 {
		result["json"] = true
	}
	return result
}

func marshalToProtoBinary(v any, sheetOption *SheetOption) ([]byte, error) {
	msgDesc := FindMessageDescriptor(sheetOption.MessageName)
	if msgDesc == nil {
		return nil, fmt.Errorf("message %s not found", sheetOption.MessageName)
	}
	msgType := msgDesc.UnwrapMessage()
	switch sheetOption.MgrType {
	case "map":
		buffer := bytes.NewBuffer(nil)
		dataMap, ok := v.(map[int32]any)
		if !ok {
			return nil, fmt.Errorf("invalid map data type: %T", v)
		}
		for _, row := range dataMap {
			msg, err := toDynamicProtoMessage(msgType, row)
			if err != nil {
				return nil, err
			}
			if _, err = protodelim.MarshalTo(buffer, msg); err != nil {
				return nil, err
			}
		}
		return buffer.Bytes(), nil
	case "slice":
		buffer := bytes.NewBuffer(nil)
		dataSlice, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid slice data type: %T", v)
		}
		for _, row := range dataSlice {
			msg, err := toDynamicProtoMessage(msgType, row)
			if err != nil {
				return nil, err
			}
			if _, err = protodelim.MarshalTo(buffer, msg); err != nil {
				return nil, err
			}
		}
		return buffer.Bytes(), nil
	case "object":
		msg, err := toDynamicProtoMessage(msgType, v)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(msg)
	default:
		return nil, fmt.Errorf("unsupported mgr type: %s", sheetOption.MgrType)
	}
}

func toDynamicProtoMessage(msgType protoreflect.MessageDescriptor, v any) (proto.Message, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	msg := dynamicpb.NewMessage(msgType)
	if err = protojson.Unmarshal(jsonData, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func mergeMgrData(dst, src any) (any, error) {
	switch m := dst.(type) {
	case map[int32]any:
		if m2, ok := src.(map[int32]any); ok {
			for k, v := range m2 {
				m[k] = v
			}
		}
		return m, nil
	case []any:
		if m2, ok := src.([]any); ok {
			m = append(m, m2...)
		}
		return m, nil
	default:
		return dst, fmt.Errorf("unsupported type: %T", dst)
	}
}

func rangeSheetData(sheetData any, columnName, refKeyName string, fn func(checkId int32)) {
	switch t := sheetData.(type) {
	case map[int32]any:
		for _, row := range t {
			if m, ok := row.(map[string]any); ok {
				columnValue := m[columnName]
				switch ct := columnValue.(type) {
				case []any: // repeated
					for _, elem := range ct {
						rangeElem(elem, refKeyName, fn)
					}
				default:
					rangeElem(columnValue, refKeyName, fn)
				}
			}
		}
	}
}

func rangeElem(columnValue any, refKeyName string, fn func(checkId int32)) {
	switch ct := columnValue.(type) {
	case int32:
		fn(columnValue.(int32)) // NOTE:暂时只处理int32
	case map[string]any:
		// NOTE:暂时只处理int32的map key
		if cfgId, ok := ct[refKeyName].(int32); ok {
			fn(cfgId)
		}
		//case []any: // repeated
		//	for _, elem := range ct {
		//		rangeElem(elem, refKeyName, fn)
		//	}
	}
}

func ExportExcelToJson(exportOption *ExportOption, excelFileName string, sheetOptions []*SheetOption) error {
	f, err := excelize.OpenFile(exportOption.DataImportPath + excelFileName)
	if err != nil {
		color.Red("%v", err)
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			color.Red("%v", err)
		}
	}()
	for _, sheetOpt := range sheetOptions {
		err := ExportSheetToJson(exportOption, f, sheetOpt)
		if err != nil {
			return err
		}
	}
	return nil
}

func ExportSheetToJson(exportOption *ExportOption, excelFile *excelize.File, sheetOption *SheetOption) error {
	v, err := ConvertSheet(exportOption, excelFile, sheetOption)
	if err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if sheetOption.ExportFileName == "" {
		sheetOption.ExportFileName = fmt.Sprintf("%s.json", sheetOption.SheetName)
	}
	return os.WriteFile(exportOption.DataExportPath+sheetOption.ExportFileName, jsonData, os.ModePerm)
}

func GetMd5(bytes []byte) string {
	md5Ctx := md5.New()
	md5Ctx.Write(bytes)
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func ExportByConfig(configFile string) error {
	fileData, err := os.ReadFile(configFile)
	if err != nil {
		color.Red("read config err:%v file:%v", err, configFile)
		return err
	}
	options := &ExportOption{}
	err = yaml.Unmarshal(fileData, options)
	if err != nil {
		color.Red("parse yaml config err:%v file:%v", err, configFile)
		return err
	}
	if len(options.CodeTemplateFiles) != len(options.CodeExportFiles) {
		color.Red("len(CodeTemplateFiles) != len(CodeExportFiles) file:%v", configFile)
		return errors.New("len(CodeTemplateFiles) != len(CodeExportFiles)")
	}
	if len(options.ProtoFiles) > 0 {
		err = ParseProtoFile([]string{options.ProtoPath}, options.ProtoFiles...)
		if err != nil {
			color.Red("ParseProtoFile err:%v", err)
			return err
		}
	}
	err = ExportAll(options, options.ExportAllExcelFile, options.ExportAllSheet)
	return err
}

func checkExportOption(opt *ExportOption) {
	autoCheckDir(&opt.DataImportPath)
	autoCheckDir(&opt.DataExportPath)
	autoCheckDir(&opt.CodeTemplatePath)
	autoCheckDir(&opt.ProtoPath)
}

func autoCheckDir(dir *string) {
	if *dir == "" {
		return
	}
	if !strings.HasSuffix(*dir, "/") && !strings.HasSuffix(*dir, "\\") {
		*dir = *dir + "/"
	}
}

func parseExportSheets(excel, exportSheetName string) ([]any, error) {
	f, err := excelize.OpenFile(excel)
	if err != nil {
		color.Red("open excel err:%v file:%v", err, excel)
		return nil, err
	}
	sheets, err := parseExportSheetsFromFile(f, exportSheetName)
	f.Close()
	if err != nil {
		color.Red("ConvertSheetErr err:%v sheet:%v", err, exportSheetName)
		return nil, err
	}
	fmt.Println(fmt.Sprintf("parseExportSheets excel:%v sheet:%v", excel, exportSheetName))
	return sheets, err
}

// 解析导出总表
func parseExportSheetsFromFile(excelFile *excelize.File, exportSheetName string) ([]any, error) {
	opt := &SheetOption{
		SheetName: exportSheetName,
		MgrType:   "slice",
	}
	rows, err := excelFile.Rows(opt.SheetName)
	if err != nil {
		color.Red("sheet:%v err:%v", opt.SheetName, err)
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			color.Red("sheet:%v err:%v", opt.SheetName, err)
		}
	}()
	opt.ColumnOpts = make([]*ColumnOption, 0)
	s := make([]any, 0)
	rowIdx := -1
	for rows.Next() {
		rowIdx++
		row, err := rows.Columns()
		if err != nil {
			color.Red("sheet:%v err:%v", opt.SheetName, err)
			return nil, err
		}
		if len(row) == 0 {
			fmt.Println(fmt.Sprintf("empty row, sheet:%v", opt.SheetName))
			continue
		}
		column0 := strings.TrimSpace(row[0])
		// 解析字段列名
		// 特殊标记的##var的列或者第一行非#开始的行就是列名定义行
		if len(opt.ColumnOpts) == 0 && isColumnNameDefineRow(column0) {
			// 列名
			columnNames := row
			//fmt.Println("columnNames:")
			//fmt.Println(columnNames)
			for columnIndex, columnName := range columnNames {
				columnName = strings.TrimSpace(columnName)
				// 跳过注释列
				if columnName == "" || strings.HasPrefix(columnName, "#") {
					continue
				}
				columnOpt := ConvertColumnOption(columnName)
				if columnOpt == nil {
					return nil, errors.New(fmt.Sprintf("columnName err %v sheet:%v", columnName, opt.SheetName))
				}
				columnOpt.ColumnIndex = columnIndex
				opt.ColumnOpts = append(opt.ColumnOpts, columnOpt)
			}
			//fmt.Println(fmt.Sprintf("keyName:%v keyType:%v", opt.MapKeyName, opt.MapKeyType))
			continue
		}
		// 跳过非数据行
		if strings.HasPrefix(column0, "#") {
			continue
		}
		// 解析数据行
		rowValue := make(map[string]any)
		for _, columnOpt := range opt.ColumnOpts {
			if columnOpt.ColumnIndex >= len(row) {
				continue // 跳过空的cell
			}
			cell := strings.TrimSpace(row[columnOpt.ColumnIndex]) // 移除首尾的空字符串
			rowValue[columnOpt.Name] = cell
		}
		s = append(s, rowValue)
	}
	return s, nil
}
