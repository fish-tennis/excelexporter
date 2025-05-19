package tool

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"strings"
)

func ExportFile(excelFileName string, sheetName string, protoMessageName string) error {
	f, err := excelize.OpenFile(excelFileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	rows, err := f.GetRows(sheetName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for rowIdx, row := range rows {
		if len(row) == 0 {
			fmt.Println(fmt.Sprintf("empty row rowIdx:%v", rowIdx))
			continue
		}
		col1 := strings.TrimSpace(row[0])
		if strings.HasPrefix(col1, "##") {
			continue
		}
	}
	return nil
}
