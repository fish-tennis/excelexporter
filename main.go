package main

import (
	"excelexporter/tool"
	"flag"
	"fmt"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "exporter.yaml", "config file")
	flag.Parse()
	err := tool.ExportByConfig(configFile)
	if err != nil {
		fmt.Println(fmt.Sprintf("err:%v", err))
	}
}
