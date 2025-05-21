package tool

import (
	"encoding/json"
	"fmt"
	"github.com/xuri/excelize/v2"
	"os"
)

type ExportOption struct {
	ImportPath string // 导入目录(excel所在目录)
	ExportPath string // 导出目录
	MultiLine  bool   // repeated字段使用多行编辑
}

func ExportExcelToJson(exportOption *ExportOption, excelFileName string, sheetOptions []*SheetOption) error {
	f, err := excelize.OpenFile(exportOption.ImportPath + excelFileName)
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
	m, err := ConvertSheetToMap(excelFile, sheetOption)
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
	return os.WriteFile(exportOption.ExportPath+sheetOption.ExportFileName, jsonData, os.ModePerm)
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
	return os.WriteFile(exportOption.ExportPath+sheetOption.ExportFileName, jsonData, os.ModePerm)
}
