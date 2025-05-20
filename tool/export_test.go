package tool

import (
	"encoding/json"
	"fmt"
	"github.com/jhump/protoreflect/desc/protoparse"
	"testing"
)

func TestExport(t *testing.T) {
	err := ParseProtoFile([]string{"./../proto"}, "cfg.proto")
	if err != nil {
		t.Fatal(err)
	}
	excelFileName := "./../data/excel/questcfg.xlsx"
	opt := &SheetOption{
		SheetName:   "questcfg",
		MessageName: "QuestCfg",
		//KeyName:        "CfgId",
		ExportFileName: "./../data/json/questcfg.json",
	}
	err = ExportSheetToJson(excelFileName, opt)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportJson(t *testing.T) {
	m := make(map[any]any)
	for i := 0; i < 10; i++ {
		m[i] = fmt.Sprintf("str%v", i)
	}
	jsonData, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", string(jsonData))
}

func TestProtoLoad(t *testing.T) {
	parser := protoparse.Parser{
		ImportPaths: []string{"E:\\work\\netmessage"}, // 设置 .proto 文件的导入路径
	}

	// 解析指定的 .proto 文件
	fds, err := parser.ParseFiles("conf.proto")
	if err != nil {
		panic(err)
	}

	//// 遍历文件中的消息定义
	//for _, fd := range fds {
	//	for _, msg := range fd.GetMessageTypes() {
	//		fmt.Println("Message Name:", msg.GetName())
	//		for _, field := range msg.GetFields() {
	//			fmt.Printf("  Field: %s, Type: %v\n", field.GetName(), field.GetType())
	//		}
	//	}
	//}

	fd := fds[0]
	msg := fd.FindMessage("QuestCfg")
	for _, field := range msg.GetFields() {
		fmt.Printf("  Field: %s,%s,%s,%s Type: %v\n",
			field.GetName(),
			field.GetFullyQualifiedName(),
			field.GetJSONName(),
			field.GetFullyQualifiedJSONName(),
			field.GetType(),
		)
	}
	newMsg := msg.AsDescriptorProto().ProtoReflect().New()
	t.Logf("newMsg:%v", newMsg)
}
