package tool

import (
	"encoding/json"
	"fmt"
	"os"
)

func ExportSheetToJson(excelFileName string, opt *SheetOption) error {
	m, err := ConvertSheetToMap(excelFileName, opt)
	if err != nil {
		return err
	}
	switch opt.KeyType {
	case "int":
		err = exportToJsonFile[int](m, opt)
	case "int8":
		err = exportToJsonFile[int8](m, opt)
	case "int16":
		err = exportToJsonFile[int16](m, opt)
	case "int32":
		err = exportToJsonFile[int32](m, opt)
	case "int64":
		err = exportToJsonFile[int64](m, opt)
	case "uint":
		err = exportToJsonFile[uint](m, opt)
		return err
	case "uint8":
		err = exportToJsonFile[uint8](m, opt)
		return err
	case "uint16":
		err = exportToJsonFile[uint16](m, opt)
		return err
	case "uint32":
		err = exportToJsonFile[uint32](m, opt)
		return err
	case "uint64":
		err = exportToJsonFile[uint64](m, opt)
		return err
	case "string":
		err = exportToJsonFile[string](m, opt)
		return err
	default:
		fmt.Println(fmt.Sprintf("unsupported key type:%v", opt.KeyType))
	}
	return err
}

func exportToJsonFile[K IntOrString](m map[any]any, opt *SheetOption) error {
	jsonMap := convertToJsonMap[K](m)
	jsonData, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}
	return os.WriteFile(opt.ExportFileName, jsonData, os.ModePerm)
}
