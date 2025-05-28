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

type ExportInfo struct {
	MgrData     any
	SheetOption *SheetOption
	MergeName   string
	CodeComment string
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
	exportInfoMap := make(map[string]*ExportInfo)
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
		codeComment := getMapValueFn(exportCfg, "CodeComment", "")
		mergeName := getMapValueFn(exportCfg, "Merge", "")
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
		if mergeName == "" {
			exportInfoMap[sheetName] = &ExportInfo{
				MgrData:     sheetData,
				SheetOption: sheetOption,
				CodeComment: codeComment,
			}
		} else {
			if mergeInfo, ok := exportInfoMap[mergeName]; ok {
				mergeData, err := mergeMgrData(mergeInfo.MgrData, sheetData)
				if err != nil {
					fmt.Println(fmt.Sprintf("ExportAllErr excel:%v sheet:%v merge:%v err:%v",
						excelFileName, sheetName, mergeName, err))
					return err
				}
				mergeInfo.MgrData = mergeData
			} else {
				exportInfoMap[mergeName] = &ExportInfo{
					MgrData:     sheetData,
					SheetOption: sheetOption,
					MergeName:   mergeName,
					CodeComment: codeComment,
				}
			}
		}
	}
	// 导出
	for _, exportInfo := range exportInfoMap {
		jsonData, err := json.MarshalIndent(exportInfo.MgrData, "", "  ")
		if err != nil {
			fmt.Println(fmt.Sprintf("ExportAllErr exportFileName:%v merge:%v err:%v",
				exportInfo.SheetOption.ExportFileName, exportInfo.MergeName, err))
			return err
		}
		exportFileName := ""
		if exportInfo.MergeName == "" {
			exportFileName = fmt.Sprintf("%s.json", exportInfo.SheetOption.SheetName)
		} else {
			exportFileName = fmt.Sprintf("%s.json", exportInfo.MergeName)
		}
		err = os.WriteFile(exportOption.DataExportPath+exportFileName, jsonData, os.ModePerm)
		if err != nil {
			fmt.Println(fmt.Sprintf("ExportAllErr exportFileName:%v merge:%v err:%v",
				exportFileName, exportInfo.MergeName, err))
			return err
		}
		mgrName := exportInfo.SheetOption.MessageName + "s"
		if exportInfo.MergeName != "" {
			mgrName = exportInfo.MergeName
		}
		generateInfo.AddDataMgrInfo(&DataMgrInfo{
			MessageName: exportInfo.SheetOption.MessageName,
			MgrName:     mgrName,
			MgrType:     exportInfo.SheetOption.MgrType,
			FileName:    exportFileName,
			CodeComment: exportInfo.CodeComment,
		})
	}
	// 生成代码
	err = GenerateCode(generateInfo, exportOption.CodeExportPath)
	if err != nil {
		fmt.Println(fmt.Sprintf("GenerateCodeErr err:%v", err))
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
			refInfo, ok := exportInfoMap[columnOption.Ref]
			if !ok {
				fmt.Println(fmt.Sprintf("ref not exists sheetName:%v column:%v ref:%v", sheetName, columnOption.Name, columnOption.Ref))
				continue
			}
			refKeyName := refInfo.SheetOption.MapKeyName
			rangeSheetData(sheetData, columnOption.Name, refKeyName, func(checkId int32) {
				if refSheetDataMap, ok := refInfo.MgrData.(map[int32]any); ok {
					if _, ok := refSheetDataMap[checkId]; !ok {
						fmt.Println(fmt.Sprintf("ref ERROR sheetName:%v column:%v ref:%v checkId:%v", sheetName, columnOption.Name, columnOption.Ref, checkId))
					}
				}
			})
		}
	}
	return nil
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
