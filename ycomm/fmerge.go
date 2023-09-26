package ycomm

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func FileSequenceMerge(src, dest string) {
	files, _ := ioutil.ReadDir(src)
	newFileName := ""
	if len(files) > 0 {
		newFileName = strings.Split(files[0].Name(), SPLIT_FLAG)[0]
	}
	fmt.Println("newFileName= ", newFileName)
	getId := func(spName string) int {
		idStr := strings.Split(spName, SPLIT_FLAG)[1]
		id, _ := strconv.Atoi(idStr)
		return id
	}

	aFile, err := os.OpenFile(newFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("FileSequenceMerge 打开A文件失败:", err)
		return
	}
	defer aFile.Close()
	sortFiles := make([]fs.FileInfo, len(files))
	for i := 0; i < len(files); i++ {
		id := getId(files[i].Name())
		sortFiles[id] = files[i]
	}

	for _, v := range sortFiles {
		//appendPath := TmpSlice+GetOsSparator()+v.Name()
		appendPath := dest + GetOsSparator() + TmpSlice + GetOsSparator() + v.Name()
		fmt.Println(appendPath)
		bFile, err1 := os.Open(appendPath)
		if err1 != nil {
			fmt.Println("打开B文件失败:", err)
			bFile.Close()
			return
		}

		// 从B文件读取内容并追加到A文件
		_, err = io.Copy(aFile, bFile)
		if err != nil {
			fmt.Println("追加文件内容失败:", err)
			return
		}
		bFile.Close()
		err1 = os.Remove(appendPath)
		if err1 != nil {
			fmt.Println(err1)
		}

	}

}

func StreamBToA(b, a string) {
	// 打开A文件以进行追加操作
	aFile, err := os.OpenFile(a, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	if err != nil {
		fmt.Println("打开A文件失败:", err)
		return
	}
	defer aFile.Close()

	// 打开B文件以进行读取
	bFile, err := os.Open(b)
	if err != nil {
		fmt.Println("打开B文件失败:", err)
		return
	}
	defer bFile.Close()

	// 从B文件读取内容并追加到A文件
	_, err = io.Copy(aFile, bFile)
	if err != nil {
		fmt.Println("追加文件内容失败:", err)
		return
	}
}
