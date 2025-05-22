package tool

import (
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
)

type DataMgrInfo struct {
	Name    string
	MgrType string // map slice object
	Comment string
}

type GenerateInfo struct {
	PackageName   string
	TemplateFiles []string
	Mgrs          []*DataMgrInfo
	DataMaps      []*DataMgrInfo
	DataSlices    []*DataMgrInfo
	DataObjects   []*DataMgrInfo
}

func (g *GenerateInfo) AddDataMgrInfo(info *DataMgrInfo) {
	switch info.MgrType {
	case "map":
		g.DataMaps = append(g.DataMaps, info)
	case "slice":
		g.DataSlices = append(g.DataSlices, info)
	case "object":
		g.DataObjects = append(g.DataObjects, info)
	}
	g.Mgrs = append(g.Mgrs, info)
}

func GenerateCode(generateInfo *GenerateInfo, codeExportPath string) error {
	for _, templateFile := range generateInfo.TemplateFiles {
		tmpl, err := template.ParseFiles(templateFile)
		if err != nil {
			return err
		}
		codeFileName := codeExportPath + strings.TrimSuffix(path.Base(templateFile), ".template")
		err = os.Mkdir(path.Dir(codeFileName), os.ModePerm)
		if err != nil && !os.IsExist(err) {
			fmt.Println(fmt.Sprintf("create %v err:%v", path.Dir(codeFileName), err))
			return err
		}
		outFile, err := os.OpenFile(codeFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			fmt.Println(fmt.Sprintf("open %v err:%v", codeFileName, err))
			return err
		}
		defer outFile.Close()
		err = tmpl.Execute(outFile, generateInfo)
		if err != nil {
			fmt.Println(fmt.Sprintf("tmpl.Execute %v err:%v", codeFileName, err))
			return err
		}
	}
	return nil
}
