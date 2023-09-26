package main

import (
	"YTools/ycomm"
	"YTools/ylog"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"time"
)

const (
	keySize   = 32
	nonceSize = 16
)

func encryptFile(filename string, key []byte) error {
	// 打开原始文件
	originalFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	// 创建加密文件
	encryptedFile, err := os.Create(filename + ".encrypted")
	if err != nil {
		return err
	}
	defer encryptedFile.Close()

	// 生成随机的nonce
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// 写入nonce到加密文件
	if _, err := encryptedFile.Write(nonce); err != nil {
		return err
	}

	// 使用AES加密
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	stream := cipher.NewCTR(block, nonce)

	// 加密数据并写入加密文件
	buffer := make([]byte, 4096) // 缓冲区大小
	for {
		n, err := originalFile.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		encryptedData := make([]byte, n)
		stream.XORKeyStream(encryptedData, buffer[:n])

		if _, err := encryptedFile.Write(encryptedData); err != nil {
			return err
		}
	}

	return nil
}

func decryptFile(filename string, key []byte) error {
	// 打开加密文件
	encryptedFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer encryptedFile.Close()

	// 创建解密文件
	decryptedFile, err := os.Create(filename + ".decrypted")
	if err != nil {
		return err
	}
	defer decryptedFile.Close()

	// 读取nonce
	nonce := make([]byte, nonceSize)
	if _, err := encryptedFile.Read(nonce); err != nil {
		return err
	}

	// 使用AES解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	stream := cipher.NewCTR(block, nonce)

	// 解密数据并写入解密文件
	buffer := make([]byte, 4096) // 缓冲区大小
	for {
		n, err := encryptedFile.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		decryptedData := make([]byte, n)
		stream.XORKeyStream(decryptedData, buffer[:n])

		if _, err := decryptedFile.Write(decryptedData); err != nil {
			return err
		}
	}

	return nil
}

func isBinaryData(data []byte) bool {
	n := len(data)
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true // 发现二进制数据，判断为二进制文件
		}
	}

	return false // 未发现二进制数据，判断为文本文件
}

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

func fileReadByOffset(file *os.File, startOffset int64, length int64) ([]byte, error) {
	// 创建一个字节切片来存储读取到的数据
	data := make([]byte, length)

	// 移动到起始位置
	_, err := file.Seek(startOffset, io.SeekStart)
	if err != nil {
		fmt.Println("移动文件指针失败:", err)
		return nil, err
	}

	// 读取数据
	_, err = file.Read(data)
	if err != nil && err != io.EOF {
		fmt.Println("读取数据失败:", err)
		return nil, err
	}

	return data, nil
}

func FileReadByOffset1(file *os.File, data []byte, startOffset int64) error {
	// 创建一个字节切片来存储读取到的数据

	// 移动到起始位置
	_, err := file.Seek(startOffset, io.SeekStart)
	if err != nil {
		fmt.Println("移动文件指针失败:", err)
		return err
	}

	// 读取数据
	_, err = file.Read(data)
	if err != nil && err != io.EOF {
		fmt.Println("读取数据失败:", err)
		return err
	}

	return nil
}

//md5 hash算法
func MD5(input string) string {
	c := md5.New()
	c.Write([]byte(input))
	bytes := c.Sum(nil)
	return hex.EncodeToString(bytes)
}

func MD5ByByte(input []byte) string {
	c := md5.New()
	c.Write(input)
	bytes := c.Sum(nil)
	return hex.EncodeToString(bytes)
}

func SHA1(input string) string {
	c := sha1.New()
	c.Write([]byte(input))
	bytes := c.Sum(nil)
	return hex.EncodeToString(bytes)
}

func SHA1ByByte(input []byte) string {
	c := sha1.New()
	c.Write(input)
	bytes := c.Sum(nil)
	return hex.EncodeToString(bytes)
}

func CRC32(input string) uint32 {
	bytes := []byte(input)
	return crc32.ChecksumIEEE(bytes)
}

func CRC32Byte(input []byte) uint32 {
	bytes := input
	return crc32.ChecksumIEEE(bytes)
}

func main333() {
	file, err := os.Open("G:\\2020数据\\BlogServer.zip")
	if err != nil {
		fmt.Println("打开文件失败:", err)
		return
	}
	stat, err := file.Stat()
	size := stat.Size()
	buf, err := fileReadByOffset(file, 0, size)

	fmt.Println(len(buf))
	nowTime := time.Now()
	md5Str := MD5ByByte(buf)
	fmt.Println("md5str=>", md5Str+" 耗时", time.Since(nowTime))

	nowTime = time.Now()
	md5Str = SHA1ByByte(buf)
	fmt.Println("SHA1ByByte=>", md5Str+" 耗时", time.Since(nowTime))

	nowTime = time.Now()
	num := CRC32Byte(buf)
	fmt.Println("num=>", num, " 耗时", time.Since(nowTime))

}

func main123() {
	// 打开文件
	file, err := os.Open("E:\\tmp\\shadow\\test2.go")
	if err != nil {
		fmt.Println("打开文件失败:", err)
		return
	}
	defer file.Close()

	// 设置读取区间的起始和结束位置（字节偏移量）
	startOffset := int64(10) // 从文件中间开始读取
	length := int64(10)      // 读取1MB的数据

	data, err := fileReadByOffset(file, startOffset, length)
	if err != nil {
		panic(err)
	}
	n := len(data)
	str := string(data)

	fmt.Println(CRC32Byte(data))
	fmt.Println(MD5(str))

	fmt.Printf("成功读取%d字节的数据:%s\n", n, str)
	// 这里的data切片中包含了你读取的数据
}

const B = 1
const KB = B * 1024
const MB = KB * 1024
const GB = MB * 1024

func main111111() {

	s1 := "hash"
	s2 := "hasha"
	if s1 == s2 {
		fmt.Println("yes not eq")
	} else {
		fmt.Println("no not eq")

	}

	fi := ycomm.FileSliceInfo{Id: 1, Length: 25}
	if fi.Hash == "" {
		fmt.Println("hash == null")
	}

	var bb []byte = nil

	if bb == nil {
		bb = make([]byte, 10)
	}
	fmt.Println(len(bb))

	SizeLimit := int64(10 * MB)
	sliceFun := func(fSzie int64) []ycomm.FileSliceInfo {
		for i := 12; ; i++ {
			sliceLen := fSzie / int64(i)
			if sliceLen < SizeLimit {
				sliceNum := i
				fsi := make([]ycomm.FileSliceInfo, sliceNum)
				for i := 0; i < sliceNum; i++ {
					if i+1 == sliceNum {
						fsi[i] = ycomm.FileSliceInfo{Id: i, Start: sliceLen * int64(i), Length: fSzie - sliceLen*int64(i)}
					} else {
						fsi[i] = ycomm.FileSliceInfo{Id: i, Start: sliceLen * int64(i), Length: sliceLen}
					}
				}
				return fsi
			}
		}
	}

	file, err := os.Open("E:\\tmp\\shadow\\test2.go")
	if err != nil {
		fmt.Println("打开文件失败:", err)
		return
	}
	defer file.Close()

	fileInfo, _ := file.Stat()

	fun := sliceFun(fileInfo.Size())
	les := len(fun)

	buf := make([]byte, fun[les-2].Length)

	FileReadByOffset1(file, buf, fun[les-3].Start)
	fmt.Println(string(buf))

	FileReadByOffset1(file, buf, fun[les-2].Start)
	fmt.Println(string(buf))

}

func main234234() {
	pathName := "E:\\tmp\\shadow\\test2.go"
	sendTargetFile, err3 := os.Open(pathName)
	defer sendTargetFile.Close()
	if err3 != nil {
		ylog.Yprint("打开发送文件失败. ", err3)
		//目标文件打开失败直接退出
		os.Exit(-1)
	}

	getFileName := func(fName string) string {
		split := strings.Split(fName, ycomm.GetOsSparator())
		fmt.Println(split)
		return split[len(split)-1]
	}
	name := getFileName(pathName)
	fmt.Println(name)
}
