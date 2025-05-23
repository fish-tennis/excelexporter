package tool

import (
	"encoding/json"
	"fmt"
	"github.com/xuri/excelize/v2"
	"os"
	"strings"
)

type ExportOption struct {
	DataImportPath string // Excel导入目录(excel所在目录)
	DataExportPath string // 数据导出目录

	CodeTemplatePath string // 代码模板目录
	CodeExportPath   string // 代码导出目录

	ExportGroup  string // 导出分组标记 c s cs
	DefaultGroup string // 默认的分组标记
}

// 从一个总表导出所有的配置表
func ExportAll(exportOption *ExportOption, exportExcelFileName, exportSheetName string) error {
	f, err := excelize.OpenFile(exportOption.DataImportPath + exportExcelFileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	exportSheetOption := &SheetOption{
		SheetName:   exportSheetName,
		MessageName: "ExportCfg",
		KeyName:     "Sheet",
	}
	exportGroup := exportOption.ExportGroup
	exportOption.ExportGroup = ""
	m, err := ConvertSheetToMap(exportOption, f, exportSheetOption)
	if err != nil {
		fmt.Println(fmt.Sprintf("ExportAllErr err:%v", err))
		return err
	}
	exportOption.ExportGroup = exportGroup
	getMapValueFn := func(strMap map[string]any, key, defaultValue string) string {
		if v, ok := strMap[key]; ok {
			return v.(string)
		}
		return defaultValue
	}
	generateInfo := &GenerateInfo{
		PackageName: "cfg",
		TemplateFiles: []string{
			exportOption.CodeTemplatePath + "data_mgr.go.template",
		},
	}
	for k, v := range m {
		sheetName := k.(string)
		exportCfg := v.(map[string]any)
		sheetExportGroup := getMapValueFn(exportCfg, "ExportGroup", exportOption.DefaultGroup)
		if sheetExportGroup == "" {
			sheetExportGroup = exportOption.DefaultGroup
		}
		if exportOption.ExportGroup != "" && !strings.Contains(sheetExportGroup, exportOption.ExportGroup) {
			continue
		}
		sheetOption := &SheetOption{
			SheetName:   sheetName,
			MessageName: getMapValueFn(exportCfg, "Message", sheetName),
			KeyName:     getMapValueFn(exportCfg, "KeyName", ""),
			KeyType:     getMapValueFn(exportCfg, "Key", ""),
		}
		excelFileName := getMapValueFn(exportCfg, "Excel", "")
		err = ExportExcelToJson(exportOption, excelFileName, []*SheetOption{sheetOption})
		if err != nil {
			fmt.Println(fmt.Sprintf("ExportAllErr excel:%v sheet:%v err:%v", excelFileName, sheetName, err))
			return err
		}
		generateInfo.AddDataMgrInfo(&DataMgrInfo{
			MessageName: sheetOption.MessageName,
			MgrType:     getMapValueFn(exportCfg, "MgrType", "map"),
			Comment:     getMapValueFn(exportCfg, "Comment", ""),
		})
	}
	// 生成代码
	err = GenerateCode(generateInfo, exportOption.CodeExportPath)
	if err != nil {
		fmt.Println(fmt.Sprintf("GenerateCodeErr err:%v", err))
		return err
	}
	return nil
}

func ExportExcelToJson(exportOption *ExportOption, excelFileName string, sheetOptions []*SheetOption) error {
	f, err := excelize.OpenFile(exportOption.DataImportPath + excelFileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			fmt.Println(err)
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
	m, err := ConvertSheetToMap(exportOption, excelFile, sheetOption)
	if err != nil {
		return err
	}
	jsonMap := convertToJsonMapByKeyType(m, sheetOption.KeyType)
	jsonData, err := json.MarshalIndent(jsonMap, "", "  ")
	if err != nil {
		return err
	}
	if sheetOption.ExportFileName == "" {
		sheetOption.ExportFileName = fmt.Sprintf("%s.json", sheetOption.SheetName)
	}
	return os.WriteFile(exportOption.DataExportPath+sheetOption.ExportFileName, jsonData, os.ModePerm)
}

type IntOrString interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~string
}

func exportToJsonFile[K IntOrString](exportOption *ExportOption, m map[any]any, sheetOption *SheetOption) error {
	jsonMap := convertToJsonMap[K](m)
	jsonData, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}
	if sheetOption.ExportFileName == "" {
		sheetOption.ExportFileName = fmt.Sprintf("%s.json", sheetOption.SheetName)
	}
	return os.WriteFile(exportOption.DataExportPath+sheetOption.ExportFileName, jsonData, os.ModePerm)
}
