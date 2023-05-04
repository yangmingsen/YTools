package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var flags struct {
	searchPath string
	searchString string
	searchType string
	exPath string
	suffix string
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: yfind ")
}

const (
	FILE_NAME = "fn"
	FILE_CONTENT = "fc"
)

func check(fileName string) bool {
	expArray := strings.Split(flags.exPath, ",")

	if flags.exPath != "" {
		for _, name := range expArray {
			if strings.Contains(fileName, name) {
				return true
			}
		}
	}

	if flags.suffix != "" {
		if strings.Contains(fileName, flags.suffix) {
			return false
		} else {
			return true
		}
	}

	return false
}


func findContent(searchPath string, searchString string) {
	fmt.Printf("Searching for '%s' in '%s'...\n\n", searchString, searchPath)
	filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path '%s': %s\n", path, err)
			return nil
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				fmt.Printf("Error opening file '%s': %s\n", path, err)
				return nil
			}
			defer file.Close()
			if check(file.Name()) {
				return nil
			}
			scanner := bufio.NewScanner(file)
			lineNum := 1
			var tmpStr strings.Builder
			tmpStr.WriteString("位置："+path+"\n")
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, searchString) {
					tmpStr.WriteString("\tLine["+strconv.Itoa(lineNum)+"]: "+line+"\n")
				}
				lineNum++
			}
			if strings.Contains(tmpStr.String(), "Line") {
				fmt.Println(tmpStr.String())
			}

			if err := scanner.Err(); err != nil {
				fmt.Printf("Error scanning file '%s': %s\n", path, err)
			}
		}

		return nil
	})
}

func findFileName(directory  string, searchTerm  string)  {
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.Contains(info.Name(), searchTerm) || strings.HasSuffix(info.Name(), searchTerm)) {
			fmt.Println(path)
		}

		return nil
	})
}

func main() {
	flag.StringVar(&flags.searchString, "k", "", "搜索内容")
	flag.StringVar(&flags.searchPath, "p", "", "搜索路径")
	flag.StringVar(&flags.searchType, "t", "", "搜索类型(fn-文件名称, fc-文件内容)")
	flag.StringVar(&flags.searchType, "ex", "", "排除类型(如 target,.git等,使用 , 分隔)")
	flag.StringVar(&flags.suffix, "sux", "", "后缀 .java,.c")
	flag.Parse()

	//flags.searchPath = "D:\\Project\\MyDemo\\"
	//flags.searchString = "public static void main"
	//flags.searchType = "fc"

	if flags.searchType == "" || flags.searchPath == "" || flags.searchString =="" {
		flag.Usage()
		return
	}

	switch flags.searchType {
	case FILE_NAME:
		findFileName(flags.searchPath, flags.searchString)
	case FILE_CONTENT:
		findContent(flags.searchPath, flags.searchString)
	}
	
}

