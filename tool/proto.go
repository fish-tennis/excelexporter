package tool

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/types/descriptorpb"
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
	fieldDesc := msgDesc.FindFieldByName(fieldName)
	if fieldDesc == nil {
		fieldDesc = msgDesc.FindFieldByJSONName(fieldName)
	}
	return fieldDesc
}

// 返回值: int32 int64 uint32 uint64 string
func GetKeyTypeString(fieldDesc *desc.FieldDescriptor) string {
	switch fieldDesc.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		return "int32"

	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		return "int64"

	case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		return "uint32"

	case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		return "uint64"

	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string"

	default:
		fmt.Println(fmt.Sprintf("field type %v not support", fieldDesc.GetType()))
	}
	return ""
}
