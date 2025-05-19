package tool

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/types/descriptorpb"
	"strconv"
	"strings"
)

func ConvertSheetToMap(excelFileName string, sheetName string, messageName string) (map[string]any, error) {
	msgDesc := FindMessageDescriptor(messageName)
	if msgDesc == nil {
		return nil, fmt.Errorf("message %s not found", messageName)
	}
	f, err := excelize.OpenFile(excelFileName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	rows, err := f.Rows(sheetName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var columnNames []string
	m := make(map[string]any)
	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if len(row) == 0 {
			fmt.Println("empty row")
			continue
		}
		// 标记行
		column0 := strings.TrimSpace(row[0])
		if strings.HasPrefix(column0, "##var") {
			// 列名
			columnNames = row
			fmt.Println("columnNames:")
			fmt.Println(columnNames)
			continue
		}
		// skip
		if strings.HasPrefix(column0, "#") {
			continue
		}
		for colIdx, colCell := range row {
			columnName := columnNames[colIdx]
			SetFieldValue(m, msgDesc, columnName, colCell)
		}
		fmt.Println()
	}
	if err = rows.Close(); err != nil {
		fmt.Println(err)
	}
	return m, nil
}

func SetFieldValue(m map[string]any, msgDesc *desc.MessageDescriptor, fieldName string, cellValue string) error {
	fieldDesc := msgDesc.FindFieldByName(fieldName)
	if fieldDesc == nil {
		fieldDesc = msgDesc.FindFieldByJSONName(fieldName)
	}
	if fieldDesc == nil {
		return fmt.Errorf("field %s not found", fieldName)
	}
	var fieldValue any
	if fieldDesc.IsRepeated() {
		var repeatedValue []any
		elemValues := strings.Split(cellValue, ";")
		for _, elemValue := range elemValues {
			elem := ConvertFieldValue(fieldDesc, elemValue)
			if elem != nil {
				repeatedValue = append(repeatedValue, elem)
			}
		}
		fieldValue = repeatedValue
	} else {
		fieldValue = ConvertFieldValue(fieldDesc, cellValue)
	}
	m[fieldDesc.GetJSONName()] = fieldValue
	fmt.Println(fmt.Sprintf("SetFieldValue %v:%v", fieldDesc.GetJSONName(), fieldValue))
	return nil
}

func ConvertFieldValue(fieldDesc *desc.FieldDescriptor, cellValue string) any {
	if len(cellValue) == 0 {
		return nil
	}
	var fieldValue any
	switch fieldDesc.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32, descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		fieldValue = Atoi(cellValue)

	case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		fieldValue = Atou(cellValue)

	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		f, _ := strconv.ParseFloat(cellValue, 32)
		fieldValue = f

	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		f, _ := strconv.ParseFloat(cellValue, 64)
		fieldValue = f

	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		fieldValue = cellValue

	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		fieldValue = strings.ToLower(cellValue) == "true" || cellValue == "1"

	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		fieldValue = Atoi(cellValue) // NOTE: 枚举暂时当作整数处理

	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		// TODO:嵌套结构,递归解析
		subMsgValue := make(map[string]any)
		subMsgDesc := fieldDesc.GetMessageType()
		kvs := convertPairString(nil, cellValue, "#", "_")
		for _, kv := range kvs {
			subFieldDesc := subMsgDesc.FindFieldByName(kv.Key)
			if subFieldDesc == nil {
				subFieldDesc = subMsgDesc.FindFieldByJSONName(kv.Key)
			}
			if subFieldDesc == nil {
				fmt.Println(fmt.Sprintf("field %s not found", kv.Key))
				continue
			}
			subFiledValue := ConvertFieldValue(subFieldDesc, kv.Value)
			if subFiledValue != nil {
				subMsgValue[subFieldDesc.GetJSONName()] = subFiledValue
			}
		}
		fieldValue = subMsgValue

	default:
		fmt.Println(fmt.Sprintf("field type %v not support", fieldDesc.GetType()))
	}
	return fieldValue
}

func Atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func Atoi64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func Atou(s string) uint64 {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return u
}

type StringPair struct {
	Key   string
	Value string
}

// 把K1_V1#K2_V2#K3_V3转换成StringPair数组(如[{K1,V1},{K2,V2},{K3,V3}]
func convertPairString(pairs []*StringPair, cellString, pairSeparator, kvSeparator string) []*StringPair {
	pairSlice := strings.Split(cellString, pairSeparator)
	for _, pairString := range pairSlice {
		kv := strings.SplitN(pairString, kvSeparator, 2)
		if len(kv) != 2 {
			continue
		}
		pairs = append(pairs, &StringPair{
			Key:   kv[0],
			Value: kv[1],
		})
	}
	return pairs
}
