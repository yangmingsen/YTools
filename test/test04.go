package main

import "YTools/ycomm"

func main() {
	path := "E:\\tmp\\shadow\\tmp\\tmpSlice"
	path1 := "E:\\tmp\\shadow\\tmp"
	//
	//files, _ := ioutil.ReadDir(path)
	//for _, v :=  range files {
	//	fmt.Println(v.Name())
	//}

	//str := "online-杨铭森-v202309.pdf___0\nonline-杨铭森-v202309.pdf___3\nonline-杨铭森-v202309.pdf___4\nonline-杨铭森-v202309.pdf___1\nonline-杨铭森-v202309.pdf___2\nonline-杨铭森-v202309.pdf___5\nonline-杨铭森-v202309.pdf___6\nonline-杨铭森-v202309.pdf___7"
	//nameList := strings.Split(str, "\n")
	//
	//fmt.Println(nameList)

	//a := "E:\\tmp\\shadow\\tmp\\test01.go"
	//b := "E:\\tmp\\shadow\\tmp\\test02.go"
	//
	//ycomm.StreamBToA(b, a)

	ycomm.FileSequenceMerge(path, path1)

}
