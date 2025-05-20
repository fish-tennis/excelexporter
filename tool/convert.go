package tool

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/types/descriptorpb"
	"strconv"
	"strings"
)

type SheetOption struct {
	SheetName      string
	MessageName    string
	KeyName        string
	ExportFileName string
}

type ColumnOption struct {
	Name   string
	Format string
}

// ColumnName#format=json#arg=value
func ConvertColumnOption(cell string) *ColumnOption {
	cell = strings.TrimSpace(cell)
	nameAndArgs := strings.Split(cell, "#")
	if len(nameAndArgs) == 0 {
		return nil
	}
	opt := &ColumnOption{
		Name: nameAndArgs[0],
	}
	for i := 1; i < len(nameAndArgs); i++ {
		arg := nameAndArgs[i]
		kv := strings.Split(arg, "=")
		if len(kv) == 2 {
			switch kv[0] {
			case "format":
				opt.Format = kv[1]
			}
		}
	}
	return opt
}

type IntOrString interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~string
}

// 把excel的一个工作簿里的数据转换成json格式的map
func ConvertSheetToJsonMap[K IntOrString](excelFileName string, opt *SheetOption) (map[K]any, error) {
	msgDesc := FindMessageDescriptor(opt.MessageName)
	if msgDesc == nil {
		return nil, fmt.Errorf("message %s not found", opt.MessageName)
	}
	keyFieldDesc := FindFieldDescriptor(msgDesc, opt.KeyName)
	if keyFieldDesc != nil {
		opt.KeyName = keyFieldDesc.GetJSONName() // 因为要导出为json格式,所以用json名
	}
	f, err := excelize.OpenFile(excelFileName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	rows, err := f.Rows(opt.SheetName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	var columnNames []string
	m := make(map[K]any)
	rowIdx := 0
	for rows.Next() {
		rowIdx++
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
		// 跳过非数据行
		if strings.HasPrefix(column0, "#") {
			continue
		}
		rowValue := make(map[string]any)
		for colIdx, colCell := range row {
			columnName := columnNames[colIdx]
			// 跳过注释列
			if strings.HasPrefix(strings.TrimSpace(columnName), "#") {
				continue
			}
			// ColumnName#format=json#arg=value
			columnOpt := ConvertColumnOption(columnName)
			err = SetFieldValue(rowValue, msgDesc, columnOpt, colCell)
			if err != nil {
				fmt.Println(fmt.Sprintf("row%v err:%v", rowIdx, err))
				continue
			}
		}
		keyValue := rowValue[opt.KeyName]
		if keyValue == nil {
			fmt.Println(fmt.Sprintf("row%v key %s not found", rowIdx, opt.KeyName))
			continue
		}
		m[keyValue.(K)] = rowValue
		fmt.Println()
	}
	return m, nil
}

func SetFieldValue(m map[string]any, msgDesc *desc.MessageDescriptor, opt *ColumnOption, cellValue string) error {
	fieldDesc := FindFieldDescriptor(msgDesc, opt.Name)
	if fieldDesc == nil {
		return fmt.Errorf("field %s not found", opt.Name)
	}
	var fieldValue any
	if fieldDesc.IsRepeated() {
		var repeatedElems []any
		elemValues := strings.Split(cellValue, ";")
		for _, elemValue := range elemValues {
			elem := ConvertFieldValue(fieldDesc, elemValue)
			if elem != nil {
				repeatedElems = append(repeatedElems, elem)
			}
		}
		if len(repeatedElems) > 0 {
			fieldValue = repeatedElems
		}
	} else {
		fieldValue = ConvertFieldValue(fieldDesc, cellValue)
	}
	if fieldValue == nil {
		return nil
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
		// 嵌套结构,递归解析
		subMsgValue := make(map[string]any)
		subMsgDesc := fieldDesc.GetMessageType()
		kvs := convertPairString(nil, cellValue, "#", "_")
		for _, kv := range kvs {
			subFieldDesc := FindFieldDescriptor(subMsgDesc, kv.Key)
			if subFieldDesc == nil {
				fmt.Println(fmt.Sprintf("field %s not found", kv.Key))
				continue
			}
			subFiledValue := ConvertFieldValue(subFieldDesc, kv.Value)
			if subFiledValue != nil {
				subMsgValue[subFieldDesc.GetJSONName()] = subFiledValue
			}
		}
		if len(subMsgValue) == 0 {
			return nil
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

func ToString(i any) string {
	switch i.(type) {
	case int:
		return strconv.Itoa(i.(int))
	case int8:
		return strconv.Itoa(int(i.(int8)))
	case int16:
		return strconv.Itoa(int(i.(int16)))
	case int32:
		return strconv.Itoa(int(i.(int32)))
	case int64:
		return strconv.FormatInt(i.(int64), 10)
	case uint:
		return strconv.FormatUint(uint64(i.(uint)), 10)
	case uint8:
		return strconv.FormatUint(uint64(i.(uint8)), 10)
	case uint16:
		return strconv.FormatUint(uint64(i.(uint16)), 10)
	case uint32:
		return strconv.FormatUint(uint64(i.(uint32)), 10)
	case uint64:
		return strconv.FormatUint(i.(uint64), 10)
	case float32:
		return strconv.FormatFloat(float64(i.(float32)), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(i.(float64), 'f', -1, 64)
	case string:
		return i.(string)
	case bool:
		return strconv.FormatBool(i.(bool))
	}
	return fmt.Sprintf("%v", i)
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
