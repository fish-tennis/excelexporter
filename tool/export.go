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

	CodeTemplatePath  string   // 代码模板目录
	CodeExportPath    string   // 代码导出目录
	CodeTemplateFiles []string // 代码模板

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
	exportSheetOption := &SheetOption{
		SheetName:   exportSheetName,
		MessageName: "ExportCfg",
		MgrType:     "slice",
	}
	exportGroup := exportOption.ExportGroup
	exportOption.ExportGroup = "" // 总表没有单独设置分组标记的行
	sheets, err := ConvertSheet(exportOption, f, exportSheetOption)
	if err != nil {
		fmt.Println(fmt.Sprintf("ExportAllErr err:%v", err))
		return err
	}
	f.Close()
	exportOption.ExportGroup = exportGroup
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
	for _, templateFile := range exportOption.CodeTemplateFiles {
		generateInfo.TemplateFiles = append(generateInfo.TemplateFiles, exportOption.CodeTemplatePath+templateFile)
	}
	// TODO: Merge功能,把不同的sheet的数据合并到一个文件中
	sheetDataMap := make(map[string]any)
	sheetOptionMap := make(map[string]*SheetOption)
	for _, v := range sheets.([]any) {
		exportCfg := v.(map[string]any)
		sheetName := getMapValueFn(exportCfg, "Sheet", "")
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
			MgrType:     getMapValueFn(exportCfg, "MgrType", "map"),
		}
		if sheetOption.MgrType == "map" {
			sheetOption.MapKeyName = getMapValueFn(exportCfg, "MapKey", "")
		}
		excelFileName := getMapValueFn(exportCfg, "Excel", "")
		f, err = excelize.OpenFile(exportOption.DataImportPath + excelFileName)
		if err != nil {
			fmt.Println(err)
			return err
		}
		sheetData, err := ConvertSheet(exportOption, f, sheetOption)
		f.Close()
		if err != nil {
			return err
		}
		jsonData, err := json.MarshalIndent(sheetData, "", "  ")
		if err != nil {
			return err
		}
		if sheetOption.ExportFileName == "" {
			sheetOption.ExportFileName = fmt.Sprintf("%s.json", sheetOption.SheetName)
		}
		err = os.WriteFile(exportOption.DataExportPath+sheetOption.ExportFileName, jsonData, os.ModePerm)
		if err != nil {
			fmt.Println(fmt.Sprintf("ExportAllErr excel:%v sheet:%v err:%v", excelFileName, sheetName, err))
			return err
		}
		generateInfo.AddDataMgrInfo(&DataMgrInfo{
			MessageName: sheetOption.MessageName,
			MgrType:     sheetOption.MgrType,
			CodeComment: getMapValueFn(exportCfg, "CodeComment", ""),
		})
		sheetDataMap[sheetName] = sheetData
		sheetOptionMap[sheetName] = sheetOption
	}
	// 生成代码
	err = GenerateCode(generateInfo, exportOption.CodeExportPath)
	if err != nil {
		fmt.Println(fmt.Sprintf("GenerateCodeErr err:%v", err))
		return err
	}
	// ref功能,检查数据关联
	for sheetName, sheetOption := range sheetOptionMap {
		for _, columnOption := range sheetOption.ColumnOpts {
			if columnOption.Ref == "" {
				continue
			}
			sheetData, _ := sheetDataMap[sheetName]
			refSheetData, ok := sheetDataMap[columnOption.Ref]
			if !ok {
				fmt.Println(fmt.Sprintf("ref not exists sheetName:%v column:%v ref:%v", sheetName, columnOption.Name, columnOption.Ref))
				continue
			}
			refKeyName := sheetOptionMap[columnOption.Ref].MapKeyName
			rangeSheetData(sheetData, columnOption.Name, refKeyName, func(checkId int32) {
				if refSheetDataMap, ok := refSheetData.(map[int32]any); ok {
					if _, ok := refSheetDataMap[checkId]; !ok {
						fmt.Println(fmt.Sprintf("ref ERROR sheetName:%v column:%v ref:%v checkId:%v", sheetName, columnOption.Name, columnOption.Ref, checkId))
					}
				}
			})
		}
	}
	return nil
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
