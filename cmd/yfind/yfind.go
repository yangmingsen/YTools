package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var flags struct {
	searchPath   string
	searchString string
	searchType   string
	//排除搜索（含路径，文件名称）
	exSearch string
	suffix   string
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: yfind -t 搜索类型 -k 搜索内容 -p 搜索路径")
	fmt.Println("注意：yfind默认会排除二进制文件搜索...")
}

const (
	FILE_NAME    = "fn"
	FILE_CONTENT = "fc"
)

//判断当前文件 是否是二进制 文件
//true => 是 ; false => 否
func isBinaryFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512) // 读取文件的前512个字节

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		fmt.Println("无法读取文件:", err)
		return false
	}

	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true // 发现二进制数据，判断为二进制文件
		}
	}

	return false // 未发现二进制数据，判断为文本文件
}

func check(fileName string) bool {
	//二进制文件排除
	if isBinaryFile(fileName) {
		return true
	}

	//文件路径，文件名称 排除
	expArray := strings.Split(flags.exSearch, ",")
	if flags.exSearch != "" {
		for _, name := range expArray {

			if strings.Contains(fileName, name) {
				return true
			}
		}
	}

	//后缀排除检查
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
			tmpStr.WriteString("位置：" + path + "\n")
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, searchString) {
					tmpStr.WriteString("\tLine[" + strconv.Itoa(lineNum) + "]: " + line + "\n")
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

func findFileName(directory string, searchTerm string) {
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
	flag.StringVar(&flags.exSearch, "ex", "", "排除类型(如 target,.git等,使用 , 分隔)")
	flag.StringVar(&flags.suffix, "sux", "", "后缀 .java,.c")
	flag.Parse()

	//flags.searchPath = "G:\\Project\\Java\\other\\MyDemoCode"
	//flags.searchString = "public static void main"
	//flags.searchType = "fc"

	if flags.searchType == "" || flags.searchPath == "" || flags.searchString == "" {
		getUsage()
		return
	}

	switch flags.searchType {
	case FILE_NAME:
		findFileName(flags.searchPath, flags.searchString)
	case FILE_CONTENT:
		findContent(flags.searchPath, flags.searchString)
	}

}
