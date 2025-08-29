package tool

import (
	"fmt"
	"os"
	"path"
	"text/template"
)

type DataMgrInfo struct {
	MessageName string // proto message name
	MgrName     string
	MgrType     string // map slice object
	FileName    string // 导出文件名,不含目录
	CodeComment string // 代码注释
}

type GenerateInfo struct {
	//PackageName   string
	TemplateFiles []string
	ExportFiles   []string
	Mgrs          []*DataMgrInfo
}

func (g *GenerateInfo) AddDataMgrInfo(info *DataMgrInfo) {
	g.Mgrs = append(g.Mgrs, info)
}

// 根据模板文件,生成代码
func GenerateCode(generateInfo *GenerateInfo) error {
	for idx, templateFile := range generateInfo.TemplateFiles {
		tmpl, err := template.ParseFiles(templateFile)
		if err != nil {
			return err
		}
		codeFileName := generateInfo.ExportFiles[idx]
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
		fmt.Println(fmt.Sprintf("GenerateCode export:%v template:%v", codeFileName, templateFile))
	}
	return nil
}
