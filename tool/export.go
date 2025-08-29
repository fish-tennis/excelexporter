package tool

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

// 导出设置项
type ExportOption struct {
	DataImportPath string `yaml:"DataImportPath"` // Excel导入目录(excel所在目录)
	DataExportPath string `yaml:"DataExportPath"` // 数据导出目录
	Md5ExportPath  string `yaml:"Md5ExportPath"`  // 可选项:导出md5文件完整路径

	CodeTemplatePath  string   `yaml:"CodeTemplatePath"`  // 代码模板目录
	CodeTemplateFiles []string `yaml:"CodeTemplateFiles"` // 代码模板
	CodeExportFiles   []string `yaml:"CodeExportFiles"`   // 代码模板导出文件名,和CodeTemplateFiles一一对应

	ExportGroup  string `yaml:"ExportGroup"`  // 导出分组标记 c s cs
	DefaultGroup string `yaml:"DefaultGroup"` // 默认的分组标记

	ExportAllExcelFile string `yaml:"ExportAllExcelFile"` // 导出总表的文件名
	ExportAllSheet     string `yaml:"ExportAllSheet"`     // 导出总表的sheet名

	ProtoPath  string   `yaml:"ProtoPath"`  // proto所在目录
	ProtoFiles []string `yaml:"ProtoFiles"` // 需要解析的proto文件
}

type ExportInfo struct {
	MgrData     any
	SheetOption *SheetOption
	MergeName   string
	CodeComment string
}

// 从一个总表导出所有的配置表
func ExportAll(exportOption *ExportOption, exportExcelFileName, exportSheetName string) error {
	checkExportOption(exportOption)
	f, err := excelize.OpenFile(exportOption.DataImportPath + exportExcelFileName)
	if err != nil {
		fmt.Println(fmt.Sprintf("open excel err:%v file:%v", err, exportOption.DataImportPath+exportExcelFileName))
		return err
	}
	sheets, err := parseExportSheets(exportOption.DataImportPath+exportExcelFileName, exportSheetName)
	f.Close()
	if err != nil {
		fmt.Println(fmt.Sprintf("ConvertSheetErr err:%v sheet:%v", err, exportSheetName))
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
	for _, v := range sheets {
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
			fmt.Println(fmt.Sprintf("open excel err:%v file:%v", err, exportOption.DataImportPath+excelFileName))
			return err
		}
		sheetData, err := ConvertSheet(exportOption, f, sheetOption)
		f.Close()
		if err != nil {
			fmt.Println(fmt.Sprintf("ConvertSheetErr err:%v sheet:%v", err, sheetOption.SheetName))
			return err
		}
		fmt.Println(fmt.Sprintf("parse excel:%v sheet:%v", excelFileName, sheetOption.SheetName))
		if mergeName == "" {
			exportInfoMap[sheetName] = &ExportInfo{
				MgrData:     sheetData,
				SheetOption: sheetOption,
				CodeComment: codeComment,
			}
			orderNames = append(orderNames, sheetName)
		} else {
			if mergeInfo, ok := exportInfoMap[mergeName]; ok {
				mergeData, err := mergeMgrData(mergeInfo.MgrData, sheetData)
				if err != nil {
					fmt.Println(fmt.Sprintf("mergeMgrDataErr excel:%v sheet:%v merge:%v err:%v",
						excelFileName, sheetName, mergeName, err))
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
			}
		}
	}

	// 导出
	md5Map := make(map[string]string)
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
		md5Map[exportFileName] = GetMd5(jsonData)
	}
	if exportOption.Md5ExportPath != "" {
		// 导出文件md5码
		jsonData, err := json.MarshalIndent(md5Map, "", "  ")
		if err != nil {
			fmt.Println(fmt.Sprintf("export md5 err:%v", err))
			return err
		}
		err = os.WriteFile(exportOption.Md5ExportPath, jsonData, os.ModePerm)
		if err != nil {
			fmt.Println(fmt.Sprintf("export md5 err:%v", err))
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
		generateInfo.AddDataMgrInfo(&DataMgrInfo{
			MessageName: exportInfo.SheetOption.MessageName,
			MgrName:     mgrName,
			MgrType:     exportInfo.SheetOption.MgrType,
			FileName:    exportFileName,
			CodeComment: exportInfo.CodeComment,
		})
	}
	err = GenerateCode(generateInfo)
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

func GetMd5(bytes []byte) string {
	md5Ctx := md5.New()
	md5Ctx.Write(bytes)
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func ExportByConfig(configFile string) error {
	fileData, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println(fmt.Sprintf("read config err:%v file:%v", err, configFile))
		return err
	}
	options := &ExportOption{}
	err = yaml.Unmarshal(fileData, options)
	if err != nil {
		fmt.Println(fmt.Sprintf("parse yaml config err:%v file:%v", err, configFile))
		return err
	}
	if len(options.CodeTemplateFiles) != len(options.CodeExportFiles) {
		fmt.Println(fmt.Sprintf("len(CodeTemplateFiles) != len(CodeExportFiles) file:%v", configFile))
		return errors.New("len(CodeTemplateFiles) != len(CodeExportFiles)")
	}
	if len(options.ProtoFiles) > 0 {
		err = ParseProtoFile([]string{options.ProtoPath}, options.ProtoFiles...)
		if err != nil {
			fmt.Println(fmt.Sprintf("ParseProtoFile err:%v", err))
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
		fmt.Println(fmt.Sprintf("open excel err:%v file:%v", err, excel))
		return nil, err
	}
	sheets, err := parseExportSheetsFromFile(f, exportSheetName)
	f.Close()
	if err != nil {
		fmt.Println(fmt.Sprintf("ConvertSheetErr err:%v sheet:%v", err, exportSheetName))
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
		fmt.Println(fmt.Sprintf("sheet:%v err:%v", opt.SheetName, err))
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			fmt.Println(fmt.Sprintf("sheet:%v err:%v", opt.SheetName, err))
		}
	}()
	opt.ColumnOpts = make([]*ColumnOption, 0)
	s := make([]any, 0)
	rowIdx := -1
	for rows.Next() {
		rowIdx++
		row, err := rows.Columns()
		if err != nil {
			fmt.Println(fmt.Sprintf("sheet:%v err:%v", opt.SheetName, err))
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
