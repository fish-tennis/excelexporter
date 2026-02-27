package tool

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/types/descriptorpb"
	"strings"
)

var (
	_protoDesc []*desc.FileDescriptor
)

// 解析proto文件
func ParseProtoFile(importPaths []string, filenames ...string) error {
	parser := &protoparse.Parser{
		ImportPaths: importPaths,
	}
	var err error
	protoDesc, err := parser.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	_protoDesc = append(_protoDesc, protoDesc...)
	for _, fd := range protoDesc {
		fmt.Println(fmt.Sprintf("ParseProtoFile Name:%v FullyQualifiedName:%v Package:%v",
			fd.GetName(), fd.GetFullyQualifiedName(), fd.GetPackage()))
	}
	return nil
}

// 获取message的结构描述
func FindMessageDescriptor(messageName string) *desc.MessageDescriptor {
	for _, fd := range _protoDesc {
		msgName := messageName
		if fd.GetPackage() != "" {
			msgName = fd.GetPackage() + "." + messageName
		}
		msgDesc := fd.FindMessage(msgName)
		if msgDesc != nil {
			return msgDesc
		}
	}
	return nil
}

// 获取message的字段的结构描述
func FindFieldDescriptor(msgDesc *desc.MessageDescriptor, fieldName string) *desc.FieldDescriptor {
	// 支持字段展开
	if strings.Index(fieldName, ".") > 0 {
		names := strings.Split(fieldName, ".") // child.fieldName
		if len(names) != 2 {
			fmt.Println(fmt.Sprintf("fieldName error1: %v %v", msgDesc.GetName(), fieldName))
			return nil
		}
		childName := names[0]
		childFieldName := names[1]
		childDesc := msgDesc.FindFieldByName(childName)
		if childDesc == nil {
			fmt.Println(fmt.Sprintf("fieldName error2: %v %v", msgDesc.GetName(), fieldName))
			return nil
		}
		// child必须是个message
		if childDesc.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			fmt.Println(fmt.Sprintf("fieldName error3: %v %v", msgDesc.GetName(), fieldName))
			return nil
		}
		childMessageDesc := childDesc.GetMessageType()
		if childMessageDesc == nil {
			fmt.Println(fmt.Sprintf("fieldName error4: %v %v", msgDesc.GetName(), fieldName))
			return nil
		}
		fieldDesc := childMessageDesc.FindFieldByName(childFieldName)
		if fieldDesc == nil {
			fieldDesc = childMessageDesc.FindFieldByJSONName(childFieldName)
		}
		return fieldDesc
	}
	fieldDesc := msgDesc.FindFieldByName(fieldName)
	if fieldDesc == nil {
		fieldDesc = msgDesc.FindFieldByJSONName(fieldName)
	}
	return fieldDesc
}

func IsRepeatedField(fieldDesc *desc.FieldDescriptor) bool {
	return fieldDesc.IsRepeated()
}

func IsMapField(fieldDesc *desc.FieldDescriptor) bool {
	// NOTE: fieldDesc.IsMap() 无法判断map类型
	if fieldDesc.GetMessageType() == nil {
		return false
	}
	return fieldDesc.GetMessageType().IsMapEntry()
}

// 返回值: int32 int64 uint32 uint64 string
func GetKeyTypeString(fieldDesc *desc.FieldDescriptor) string {
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
	switch fieldDesc.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		return "int32"

	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		return "int64"

	case descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		return "uint32"

	case descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		return "uint64"

	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string"

	default:
		fmt.Println(fmt.Sprintf("field type %v not support", fieldDesc.GetType()))
	}
	return ""
}
