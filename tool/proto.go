package tool

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
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
