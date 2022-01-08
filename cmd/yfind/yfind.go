package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

var findTypeChoose = 1 //defualt value is 1 是找后缀; 2是找文件名; 3找路径中含有关键字的路径

func errFunc(info string, err error) bool {
	if err != nil {
		log.Println("info:", info, " error:", err)
		return false
	}
	return true
}

func isSpecificFile(fileName string, speciType string) bool {
	return strings.HasSuffix(fileName, speciType)
}

func printSpecificFile(fileName string, speciType string) {
	if isSpecificFile(fileName, speciType) {
		fmt.Println("find: ", fileName)
	}
}

//如果目录路径最后没有分隔符则加上,
//如果有则不加
func checkHaveSeparatorInEnd(path string) string {
	idxSperator := strings.LastIndex(path, "/")
	if (idxSperator + 1) < len(path) {
		path += "/"
	}
	return path
}

func openDir(path string) []os.FileInfo {
	path = checkHaveSeparatorInEnd(path)
	curPath, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if errFunc("opend dir error in openDir..", err) == false {
		os.Exit(2)
	}

	dir1, err2 := curPath.Readdir(-1)
	if !errFunc("read dir error in openDir..", err2) {
		os.Exit(2)
	}
	curPath.Close()
	return dir1
}

func findSpecificFile(path string, speciType string) {
	dir1 := openDir(path)
	for _, name := range dir1 {
		if name.IsDir() {
			findSpecificFile(path+name.Name()+"/", speciType)
		} else {
			printSpecificFile(path+name.Name(), speciType)
		}
	}
	return
}

const SUF string = "-suf"   //找后缀
const NAME string = "-name" //找文件

func goFind(findPath string, findType string) {
	maxProcs := runtime.NumCPU()

	var wg sync.WaitGroup
	dir1 := openDir(findPath)

	dirCount := 0
	for _, na := range dir1 {
		if na.IsDir() {
			dirCount++
		}
	}

	if dirCount > maxProcs {
		runtime.GOMAXPROCS(maxProcs / 2)
	} else {
		runtime.GOMAXPROCS(1)
	}

	wg.Add(dirCount)

	for _, name := range dir1 {
		if name.IsDir() {
			go func() {
				defer wg.Done()
				findTypeIsDirOption(findPath+"/"+name.Name()+"/", findType)
			}()

		} else {
			findTypeIsFileOption(findPath+"/"+name.Name(), findType)
		}
	}
	wg.Wait()
}

func findTypeIsDirOption(fileName string, speciType string) {
	switch findTypeChoose {
	case 1:
		findSpecificFile(fileName, speciType)
	case 2:
		fmt.Println("no implementDir1")
	case 3:
		fmt.Println("no implementDir2")
	}
}

func findTypeIsFileOption(fileName string, speciType string) {
	switch findTypeChoose {
	case 1:
		printSpecificFile(fileName, speciType)
	case 2:
		fmt.Println("no implementFile1") //
	case 3:
		fmt.Println("no implementFile2") //
	}
}

//args format
//example:   findTools -suf .txt [path]
//explain:  在(可选path或者默认当前工作路径)路径中找到后缀以 .txt结尾的文件
func main() {
	args := os.Args
	var opt, findType, findPath string
	if args == nil || len(args) < 3 || len(args) > 4 {
		log.Println("args format error")
		return
	}

	opt = args[1]      //获取选项
	findType = args[2] //获取查找类型
	//如果只有3个选项 那么查找路径默认为当前路径
	if len(args) == 3 {
		workDir, err := os.Getwd() //获取当前工作路径
		if !errFunc("open wordir error in main", err) {
			return
		}
		findPath = workDir
	} else if len(args) == 4 {
		findPath = args[3]
	}

	switch opt {
	case SUF:
		{
			goFind(findPath, findType)
		}
	case NAME:
		log.Println("no implement the method")
	default:
		log.Println("no something to do!")
	}

}
