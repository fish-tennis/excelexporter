package tool

import (
	"encoding/json"
	"errors"
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
	MgrType        string // map slice object
	MapKeyName     string // 填空直接使用第一个非注释列作为key名(MgrType=map时才有效)
	MapKeyType     string // int int32 int64 uint uint32 uint64 string(MgrType=map时才有效)
	ExportFileName string // 填空直接使用SheetName作为文件名
	ColumnOpts     []*ColumnOption
}

type ColumnOption struct {
	Name        string
	ColumnIndex int
	ExportGroup string // 导出分组标记 c s cs
	// 特殊的格式 如format=json
	// 些复杂结构的数据在excel里编辑是很麻烦的,这时候可以考虑直接使用json格式
	Format string
	// 字段是message,对应的字段名,如#Field=Field1_Field2_Field3
	// no和full是特定格式
	//	---------------------------------------------
	//	| Item1     | Item2           | Item3        |
	//	| #Field=no | #Field=Id_Count | #Field=full  |
	//	---------------------------------------------
	//	| 1_3       | 1_3             | Id_1#Count_3 |
	//	---------------------------------------------
	FieldNames []string
	Ref        string // 关联的其他配置表的sheet名
}

// 简洁模式,不需要字段名(#Field=no)
func (c *ColumnOption) IsNoFieldName() bool {
	return len(c.FieldNames) == 1 && c.FieldNames[0] == "no"
}

// #Field=full 每一行都需要填上字段名
func (c *ColumnOption) IsFullFieldName() bool {
	return len(c.FieldNames) == 1 && c.FieldNames[0] == "full"
}

// ColumnName#format=json#arg=value
func ConvertColumnOption(cell string) *ColumnOption {
	cell = strings.TrimSpace(cell)
	cell = strings.ReplaceAll(cell, "\n", "")
	// 支持换行,如:
	//	---------------------------------------------
	//	| Item1     | Item2           | Item3        |
	//	| #Field=no | #Field=Id_Count | #Field=full  |
	//	---------------------------------------------
	//	| 1_3       | 1_3             | Id_1#Count_3 |
	//	---------------------------------------------
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
		if len(kv) == 0 {
			continue
		}
		switch strings.ToLower(kv[0]) {
		case "field":
			if len(kv) == 2 {
				opt.FieldNames = strings.Split(kv[1], "_")
			}
		case "format":
			if len(kv) == 2 {
				opt.Format = kv[1]
			}
		case "ref":
			if len(kv) == 2 {
				opt.Ref = kv[1]
			}
		}
	}
	return opt
}

// opt.MgrType="map"时,返回map[key]any
// opt.MgrType="slice"时,返回[]any
func ConvertSheet(exportOption *ExportOption, excelFile *excelize.File, opt *SheetOption) (any, error) {
	msgDesc := FindMessageDescriptor(opt.MessageName)
	if msgDesc == nil {
		return nil, fmt.Errorf("message %s not found, sheet:%v", opt.MessageName, opt.SheetName)
	}
	var mapKeyFieldDesc *desc.FieldDescriptor
	if opt.MgrType == "map" {
		mapKeyFieldDesc = FindFieldDescriptor(msgDesc, opt.MapKeyName)
		if mapKeyFieldDesc != nil {
			opt.MapKeyName = mapKeyFieldDesc.GetJSONName() // 因为要导出为json格式,所以用json名
		}
		if opt.MapKeyType == "" && mapKeyFieldDesc != nil {
			opt.MapKeyType = GetKeyTypeString(mapKeyFieldDesc)
		}
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
	hasParseExportGroupRow := false
	opt.ColumnOpts = make([]*ColumnOption, 0)
	m := make(map[any]any)
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
				// 如果没有指定MapKey,则默认第一个非注释列为MapKey
				if opt.MgrType == "map" && opt.MapKeyName == "" {
					opt.MapKeyName = columnOpt.Name
					mapKeyFieldDesc = FindFieldDescriptor(msgDesc, opt.MapKeyName)
					if mapKeyFieldDesc != nil {
						opt.MapKeyName = mapKeyFieldDesc.GetJSONName() // 因为要导出为json格式,所以用json名
					}
					if opt.MapKeyType == "" && mapKeyFieldDesc != nil {
						opt.MapKeyType = GetKeyTypeString(mapKeyFieldDesc)
					}
				}
			}
			//fmt.Println(fmt.Sprintf("keyName:%v keyType:%v", opt.MapKeyName, opt.MapKeyType))
			continue
		}
		// 解析导出分组标记
		if !hasParseExportGroupRow && len(opt.ColumnOpts) > 0 && exportOption.ExportGroup != "" && isExportGroupRow(column0) {
			for _, columnOpt := range opt.ColumnOpts {
				group := exportOption.DefaultGroup
				if columnOpt.ColumnIndex < len(row) {
					group = strings.TrimSpace(row[columnOpt.ColumnIndex])
					if group == "" {
						group = exportOption.DefaultGroup
					}
				}
				columnOpt.ExportGroup = group
			}
			hasParseExportGroupRow = true
			continue
		}
		// 跳过非数据行
		if strings.HasPrefix(column0, "#") {
			continue
		}
		// 解析数据行
		rowValue := make(map[string]any)
		for _, columnOpt := range opt.ColumnOpts {
			if exportOption.ExportGroup != "" && !strings.Contains(columnOpt.ExportGroup, exportOption.ExportGroup) {
				continue
			}
			fieldDesc := FindFieldDescriptor(msgDesc, columnOpt.Name)
			if fieldDesc == nil {
				fmt.Println(fmt.Sprintf("FieldNameNotFound row%v name:%s sheet:%v", rowIdx, columnOpt.Name, opt.SheetName))
				continue
			}
			if columnOpt.ColumnIndex >= len(row) {
				continue // 跳过空的cell
			}
			cell := strings.TrimSpace(row[columnOpt.ColumnIndex]) // 移除首尾的空字符串
			// format扩展 json
			if columnOpt.Format == "json" {
				// 允许不填最外层的{}
				if len(cell) > 0 && cell[0] != '{' && cell[len(cell)-1] != '}' {
					cell = "{" + cell + "}"
				}
				err = SetFieldValueJson(rowValue, fieldDesc, columnOpt, cell)
				if err != nil {
					fmt.Println(fmt.Sprintf("SetFieldValueJsonErr row%v sheet:%v err:%v", rowIdx, opt.SheetName, err))
					continue
				}
			} else {
				err = SetFieldValue(rowValue, fieldDesc, columnOpt, cell, false)
				if err != nil {
					fmt.Println(fmt.Sprintf("SetFieldValueErr row%v sheet:%v err:%v", rowIdx, opt.SheetName, err))
					continue
				}
			}
		}
		if opt.MgrType == "map" {
			keyValue := rowValue[opt.MapKeyName]
			if keyValue == nil {
				fmt.Println(fmt.Sprintf("row%v sheet:%v key %s not found", rowIdx, opt.SheetName, opt.MapKeyName))
				continue
			}
			m[keyValue] = rowValue
		} else if opt.MgrType == "slice" {
			s = append(s, rowValue)
		}
	}
	if opt.MgrType == "map" {
		// 把key转换成实际类型
		return convertToJsonMapByKeyType(m, opt.MapKeyType), nil
	} else if opt.MgrType == "slice" {
		return s, nil
	}
	return nil, errors.New(fmt.Sprintf("unsupported MgrType %v sheet:%v", opt.MgrType, opt.SheetName))
}

func isColumnNameDefineRow(column0 string) bool {
	if strings.HasPrefix(column0, "##var") {
		return true
	}
	if strings.HasPrefix(column0, "#") {
		return false
	}
	return true
}

func isExportGroupRow(column0 string) bool {
	if strings.HasPrefix(column0, "##group") {
		return true
	}
	if strings.HasPrefix(column0, "#") {
		return false
	}
	return true
}

type IntOrString interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~string
}

func convertToJsonMap[K IntOrString](m map[any]any) any {
	jsonMap := make(map[K]any)
	for k, v := range m {
		jsonMap[k.(K)] = v
	}
	return jsonMap
}

func convertToJsonMapByKeyType(m map[any]any, keyType string) any {
	switch keyType {
	case "int":
		return convertToJsonMap[int](m)
	case "int8":
		return convertToJsonMap[int8](m)
	case "int16":
		return convertToJsonMap[int16](m)
	case "int32":
		return convertToJsonMap[int32](m)
	case "int64":
		return convertToJsonMap[int64](m)
	case "uint":
		return convertToJsonMap[uint](m)
	case "uint8":
		return convertToJsonMap[uint8](m)
	case "uint16":
		return convertToJsonMap[uint16](m)
	case "uint32":
		return convertToJsonMap[uint32](m)
	case "uint64":
		return convertToJsonMap[uint64](m)
	case "string":
		return convertToJsonMap[string](m)
	}
	fmt.Println(fmt.Sprintf("convertToJsonMapByKeyType err keyType:%v", keyType))
	return m
}

func SetFieldValue(m map[string]any, fieldDesc *desc.FieldDescriptor, opt *ColumnOption, cellValue string, isSubMsg bool) error {
	var fieldValue any
	// [] or map
	if fieldDesc.IsRepeated() {
		// map字段
		if fieldDesc.IsMap() {
			mapField := make(map[any]any)
			keyType := fieldDesc.GetMessageType().FindFieldByNumber(1)
			valueType := fieldDesc.GetMessageType().FindFieldByNumber(2)
			// map字段,同时支持换行和;
			lines := strings.Split(cellValue, "\n")
			for _, line := range lines {
				// k1_v1;k2_v2
				sepChar := ";"
				if isSubMsg {
					sepChar = "," // 第一层用了;做分隔符 嵌套的子对象用,分割
				}
				pairValues := strings.Split(line, sepChar)
				for _, pairValue := range pairValues {
					kv := strings.SplitN(pairValue, "_", 2)
					if len(kv) != 2 {
						continue
					}
					k := ConvertFieldValue(keyType, opt, kv[0])
					v := ConvertFieldValue(valueType, opt, kv[1])
					mapField[k] = v
				}
			}
			if len(mapField) > 0 {
				fieldValue = convertToJsonMapByKeyType(mapField, GetKeyTypeString(keyType))
			}
		} else {
			// repeated字段
			var repeatedElems []any
			// repeated字段,同时支持换行和;
			lines := strings.Split(cellValue, "\n")
			for _, line := range lines {
				sepChar := ";"
				if isSubMsg {
					sepChar = "," // 第一层用了;做分隔符 嵌套的子对象用,分割
				}
				elemValues := strings.Split(line, sepChar)
				for _, elemValue := range elemValues {
					elem := ConvertFieldValue(fieldDesc, opt, elemValue)
					if elem != nil {
						repeatedElems = append(repeatedElems, elem)
					}
				}
			}
			if len(repeatedElems) > 0 {
				fieldValue = repeatedElems
			}
		}
	} else {
		// 普通字段
		fieldValue = ConvertFieldValue(fieldDesc, opt, cellValue)
	}
	if fieldValue == nil {
		return nil
	}
	m[fieldDesc.GetJSONName()] = fieldValue
	//fmt.Println(fmt.Sprintf("SetFieldValue %v:%v", fieldDesc.GetJSONName(), fieldValue))
	return nil
}

func SetFieldValueJson(m map[string]any, fieldDesc *desc.FieldDescriptor, opt *ColumnOption, cellValue string) error {
	if len(cellValue) == 0 {
		return nil
	}
	var jsonValue any
	// [] or map
	if fieldDesc.IsRepeated() {
		// map字段
		if fieldDesc.IsMap() {
			jsonValue = make(map[string]any)
		} else {
			// []
			jsonValue = make([]any, 0)
		}
	} else {
		jsonValue = make(map[string]any)
	}
	err := json.Unmarshal([]byte(cellValue), &jsonValue)
	if err != nil {
		fmt.Println(fmt.Sprintf("SetFieldValueJsonErr err:%v", err))
		return err
	}
	m[fieldDesc.GetJSONName()] = jsonValue
	//fmt.Println(fmt.Sprintf("SetFieldValueJson %v:%v", fieldDesc.GetJSONName(), jsonValue))
	return nil
}

func ConvertFieldValue(fieldDesc *desc.FieldDescriptor, columnOption *ColumnOption, cellValue string) any {
	if len(cellValue) == 0 {
		return nil
	}
	//	+-------------------------+-----------+
	//	|       Declared Type     |  Go Type  |
	//	+-------------------------+-----------+
	//	| int32, sint32, sfixed32 | int32     |
	//	| int64, sint64, sfixed64 | int64     |
	//	| uint32, fixed32         | uint32    |
	//	| uint64, fixed64         | uint64    |
	//	| float                   | float32   |
	//	| double                  | double32  |
	//	| bool                    | bool      |
	//	| string                  | string    |
	//	| bytes                   | []byte    |
	//	+-------------------------+-----------+
	var fieldValue any
	switch fieldDesc.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		fieldValue = int32(Atoi(cellValue))

	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		fieldValue = int64(Atoi(cellValue))

	case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		fieldValue = uint32(Atou(cellValue))

	case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		fieldValue = uint64(Atou(cellValue))

	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		f, _ := strconv.ParseFloat(cellValue, 32)
		fieldValue = float32(f)

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
		//subMsgName := subMsgDesc.GetName()
		// 简洁模式,不需要字段名,不支持多层结构 如1_2_5
		if columnOption.IsNoFieldName() {
			fieldValues := strings.Split(cellValue, "_")
			for fieldIndex, fieldStr := range fieldValues {
				if fieldIndex >= len(subMsgDesc.GetFields()) {
					break
				}
				subFieldDesc := subMsgDesc.GetFields()[fieldIndex]
				SetFieldValue(subMsgValue, subFieldDesc, columnOption, fieldStr, true)
			}
		} else if columnOption.IsFullFieldName() {
			// 默认使用字段名模式,该模块填写略复杂,但是兼容性好一些 如CfgId_2#Args_1
			kvs := convertPairString(nil, cellValue, "#", "_")
			for _, kv := range kvs {
				subFieldDesc := FindFieldDescriptor(subMsgDesc, kv.Key)
				if subFieldDesc == nil {
					fmt.Println(fmt.Sprintf("field %s not found", kv.Key))
					continue
				}
				SetFieldValue(subMsgValue, subFieldDesc, columnOption, kv.Value, true)
			}
		} else {
			// #Field=Field1_Field2_Field3
			fieldValues := strings.Split(cellValue, "_")
			for fieldIndex, fieldStr := range fieldValues {
				if fieldIndex >= len(subMsgDesc.GetFields()) {
					break
				}
				//subFieldName := columnOption.FieldNames[fieldIndex]
				subFieldDesc := subMsgDesc.GetFields()[fieldIndex]
				if subFieldDesc == nil {
					fmt.Println(fmt.Sprintf("fieldIdx %v not found", fieldIndex))
					continue
				}
				SetFieldValue(subMsgValue, subFieldDesc, columnOption, fieldStr, true)
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
