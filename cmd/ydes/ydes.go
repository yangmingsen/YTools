package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	//key加密长度 // 128bit
	keySize = 16

	//ept文件后缀
	YEPT = ".yept"
	//不再使用该后缀名 废弃于20230614,使用YEPT, 这里做兼容
	EPT = ".encrypted"

	//50M => 使用50M作为大文件加密界限
	s50M = 50 * 1024 * 1024
)

func encryptBigFile(filename string, key []byte) error {
	logf("bigSize file doEncryptFileCheck begin....")
	// 打开原始文件
	originalFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		originalFile.Close()
		os.Remove(filename)
	}()

	eptName := getEptNewFileName(filename)
	_, cerr := os.Stat(eptName)
	if cerr == nil { //删除即将创建的文件
		os.Remove(eptName)
	}

	// 创建加密文件
	encryptedFile, err := os.Create(eptName)
	if err != nil {
		return err
	}
	defer encryptedFile.Close()

	// 生成随机的nonce
	nonce := make([]byte, keySize)
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

func decryptBigFile(filename string, key []byte) error {
	logf("bigSize file doDecryptFile begin....")
	// 打开加密文件
	encryptedFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer encryptedFile.Close()

	dptName := getDeptNewFileName(filename)
	_, cerr := os.Stat(dptName)
	if cerr == nil { //删除即将创建的文件
		os.Remove(dptName)
	}
	// 创建解密文件
	decryptedFile, err := os.Create(dptName)
	if err != nil {
		return err
	}
	defer decryptedFile.Close()

	// 读取nonce
	nonce := make([]byte, keySize)
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

func encryptFile(key []byte, filename string) error {
	plaintext, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	cFileName := getEptNewFileName(filename)
	_, cErr := os.Stat(cFileName)
	if cErr == nil {
		os.Remove(cFileName)
	}

	err = ioutil.WriteFile(cFileName, ciphertext, 0644)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func isBinaryData(data []byte) bool {
	n := len(data)
	if n > 512 {
		n = 512
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true // 发现二进制数据，判断为二进制文件
		}
	}

	return false // 未发现二进制数据，判断为文本文件
}

//获取解密文件名
func getDeptNewFileName(fileName string) string {
	if containsAny(fileName, EPT, YEPT) {
		fileName = strings.ReplaceAll(fileName, YEPT, "")
		fileName = strings.ReplaceAll(fileName, EPT, "")
	}

	return fileName
}

//获取加密文件名
func getEptNewFileName(fileName string) string {
	return fileName+YEPT
}

//解密文件数据实际实现
func decryptFile(key []byte, filename string) error {
	ciphertext, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	//输出文件名称
	outPutName := getDeptNewFileName(filename)
	_, err1 := os.Stat(outPutName)
	if err1 == nil {
		os.Remove(outPutName)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	if len(ciphertext) < aes.BlockSize {
		return errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)
	if flags.show && !isBinaryData(plaintext) {
		fmt.Println(string(plaintext))
		return nil
	} else {
		err =  ioutil.WriteFile(outPutName, plaintext, 0644)
		if err != nil {
			return err
		} else {
			return nil;
		}
	}
}

//加密字符串
func encryptText(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

//解密字符串
func decryptText(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

var flags struct {
	mode     string
	key      string
	fileName string
	data     string
	show     bool
	delSrc   bool
	debug   bool
}

func getUsage() {
	flag.Usage()
	fmt.Println("加密文件: ydes -m en -k 秘钥 -file ./hello.txt")
	fmt.Println("解密文件: ydes -m de -k 秘钥 -file ./hello.txt.decrypted -dsrc 任何指")
	fmt.Println("解密文件: ydes -m de -k 秘钥 -file ./hello.txt.decrypted -show 任何值")
	fmt.Println("解密字符: ydes -m en -k 秘钥 -data 明文")
	fmt.Println("解密字符: ydes -m de -k 秘钥 -data 密文")
}

//is one of method
func containsAny(key string, any ...string) bool {
	for i := range any {
		if strings.Contains(key, any[i]) {
			return true
		}
	}

	return false
}

//解密文件判断方法之一，这个地方判断是否进行大文件加密
func doDecryptFile(key string, fileName string) error {
	if !containsAny(fileName, YEPT, EPT){
		return errors.New("非解密文件...("+fileName+")")
	}
	stat, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	fileSize := stat.Size()
	if fileSize > int64(s50M) {
		err := decryptBigFile(fileName, []byte(key))
		if err != nil {
			return err
		}
	} else {
		err := decryptFile([]byte(key), fileName)
		if err != nil {
			return err
		}
	}

	if flags.delSrc {
		//删除旧文件
		os.Remove(fileName)
	}

	return nil
}

//读取用户输入的命令
func getUserInput(reader *bufio.Reader, tip string) (string, bool) {
	// 读取用户输入的命令
	fmt.Print("[",tip,"]$ ")
	command, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("获取输入命令出错: ", err)
		return "", false
	}
	command = strings.TrimSpace(command)
	if command == "" && len(command) == 0 {
		//空字符情况处理
		return "", false
	}

	return command, true
}


const authKey = "eWFuZ21pbmdzZW4="
//加密文件判断方法之一，这个地方判断是否进行大文件加密
func doEncryptFile(key, fileName string) error {
	if containsAny(fileName, YEPT, EPT) {
		return errors.New("当前文件已为加密文件,不可再加密...("+fileName+")")
	}
	stat, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	fileSize := stat.Size()
	if fileSize > int64(s50M) {
		err := encryptBigFile(fileName, []byte(key))
		if err != nil {
			return err
		}
	} else {
		err := encryptFile([]byte(key), fileName)
		if err != nil {
			return err
		}
	}
	if flags.delSrc {
		//删除源文件
		os.Remove(fileName)
	}

	return nil
}

//对于目录进行警告信息
func dirCheck(mode, key, fileName string) error {
	fmt.Println("警告：目录加密/解密具备风险性,请确认是否继续....")
	reader := bufio.NewReader(os.Stdin)
	tip := "你的选择(y/n)"
	res, ok := getUserInput(reader, tip)
	if !ok {
		return errors.New("错误指令：停止目录加密/解密...")
	}
	if strings.ToLower(res) != "y" {
		return errors.New("you choice No,停止目录加密/解密...")
	}
	fmt.Println("Ok, 请求提供身份认证秘钥...")
	tip = "请输入(认证秘钥)"
	res, ok = getUserInput(reader, tip)
	if !ok {
		return errors.New("错误输入: 停止目录加密/解密...")
	}
	deStr := base64.StdEncoding.EncodeToString([]byte(res))
	if deStr == authKey {
		fmt.Println("OK... we prepare encrypted this dir....")
		fwErr := filepath.WalkDir(fileName, func(abPath string, info fs.DirEntry, err error) error {
			if err != nil {
				fmt.Printf("访问路径时发生错误: %v\n", err)
				return nil
			}
			if info.IsDir() {
				return nil
			}

			if mode == "en" {
				fmt.Println("加密：", abPath)
				 eErr := doEncryptFile(key, abPath)
				 if eErr != nil {
				 	fmt.Println("错误：", eErr)
				 }
			} else {
				fmt.Println("解密：", abPath)
				dErr := doDecryptFile(key, abPath)
				if dErr != nil {
					fmt.Println("错误：", dErr)
				}
			}
			return nil
		})
		if fwErr != nil {
			return fwErr
		}

	} else {
		return errors.New("错误认证秘钥...")
	}

	return nil
}

//解密文件时判断，如是否是目录情况
func doDecryptFileCheck(key, fileName string) error {
	stat, err := os.Stat(fileName)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return dirCheck("de", key, fileName)
	} else {
		return doDecryptFile(key, fileName)
	}
}

//加密文件时判断，如是否是目录情况
func doEncryptFileCheck(key string, fileName string) error {
	stat, err := os.Stat(fileName)
	logf("statInfo: ", stat)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return dirCheck("en", key, fileName)
	} else {
		return doEncryptFile(key, fileName)
	}

}

var logger = log.New(os.Stderr, "", log.Lshortfile|log.LstdFlags)
func logf(f string, v ...interface{}) {
	if flags.debug {
		logger.Output(2, fmt.Sprintf(f, v...))
	}
}

func main() {
	flag.StringVar(&flags.mode, "m", "", "模式(en/de)")
	flag.StringVar(&flags.key, "k", "", "秘钥（加密==解密）")
	flag.StringVar(&flags.fileName, "file", "", "加解密文件")
	flag.StringVar(&flags.data, "data", "", "加解密数据")
	flag.BoolVar(&flags.show, "show", false, "只展示不输出到文件(只合适小文本文件)")
	flag.BoolVar(&flags.delSrc, "dsrc", false, "删除源文件")
	flag.BoolVar(&flags.debug, "debug", false, "debug mode")
	flag.Parse()

	//flags.mode = "de"
	//flags.key = "yangmingsen"
	//flags.fileName = "E:\\tempFiles\\demo-api"
	//flags.delSrc = true
	//flags.debug = true
	//flags.show = true
	//flags.data = "bdC6IQA/ygjDpYJ6WrHjLSZrSo7J2aDackOx"

	logf("输入参数：", flags)

	var command string
	check := true
	//校验模式不能为空
	if flags.mode != "" {
		if flags.mode == "en" {
			command = "encrypt"
		} else if flags.mode == "de" {
			command = "decrypt"
		} else {
			logf("错误模式：", flags.mode)
			check = false
		}
	} else {
		logf("模式不能为空")
		check = false
	}

	//校验加密key不能为空
	if flags.key == "" {
		logf("key不能为空")
		check = false
	} else {
		//加密key修改
		if len(flags.key) < keySize {
			for kLen := len(flags.key); kLen < keySize; kLen++ {
				flags.key = flags.key + "0"
			}
		} else if len(flags.key) > keySize {
			flags.key = flags.key[0:keySize]
		}
	}

	//是否 存在加密数据
	isExsitData := false

	//是否存在文件数据
	isFile := false
	if flags.fileName != "" {
		isFile = true
		isExsitData = true
	}

	//是否存在加密数据
	isData := false
	if flags.data != "" {
		isData = true
		isExsitData = true
	}

	if isExsitData == false {
		logf("加密数据不能为空")
		check = false
	}

	if !check {
		getUsage()
		return
	} else {
		//fmt.Println(flags)
	}

	key := flags.key
	filename := flags.fileName
	logf("key: ", key)
	logf("fileName: ", filename)
	logf("isFile: ", isFile)
	logf("isData: ", isData)


	switch command {
	case "encrypt":
		{
			if isFile {
				//文件加密
				if err := doEncryptFileCheck(key, filename); err != nil {
					fmt.Println("Error encrypting file:", err)
					return
				}
			} else if isData {
				logf("isData =>data => : ", flags.data)
				// 加密
				ciphertext, err := encryptText([]byte(key), []byte(flags.data))
				if err != nil {
					fmt.Println("Encryption Text error:", err)
					return
				}

				fmt.Println("Ciphertext:", base64.StdEncoding.EncodeToString(ciphertext))
			}

		}

	case "decrypt":
		{
			if isFile {
				if err := doDecryptFileCheck(key, filename); err != nil {
					fmt.Println("Error decrypting file:", err)
					return
				}
				logf("File decrypted successfully.")
			}

			if isData {
				// 解密
				decodeString, deErr := base64.StdEncoding.DecodeString(flags.data)
				if deErr != nil {
					fmt.Println("base64 decode data err:", deErr)
				}
				decryptedText, err := decryptText([]byte(key), decodeString)
				if err != nil {
					fmt.Println("Decryption Text error:", err)
					return
				}

				fmt.Println("Decrypted text: \n", string(decryptedText))
			}
		}

	default:
		fmt.Println("Invalid command:", command)
	}
}
