package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	//key加密长度 // 128bit
	keySize = 16

	//50M => 使用50M作为大文件加密界限
	s50M = 50 * 1024 * 1024
)

func encryptBigFile(filename string, key []byte) error {
	fmt.Println("bigSize file doEncryptFile begin....")
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
	fmt.Println("bigSize file doDecryptFile begin....")
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

	return ioutil.WriteFile(filename+".encrypted", ciphertext, 0644)
}

func decryptFile(key []byte, filename string) error {
	ciphertext, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	outPutName := strings.ReplaceAll(filename, ".encrypted", "")

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

	return ioutil.WriteFile(outPutName, plaintext, 0644)
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
}

func getUsage() {
	flag.Usage()
	fmt.Println("加密文件: ydes -m en -k 秘钥 -file ./hello.txt")
	fmt.Println("解密文件: ydes -m de -k 秘钥 -file ./hello.txt.decrypted")
	fmt.Println("解密字符: ydes -m en -k 秘钥 -data 明文")
	fmt.Println("解密字符: ydes -m de -k 秘钥 -data 密文")
}

func doDecryptFile(key string, fileName string) error {
	stat, err := os.Stat(fileName)
	if stat.IsDir() {
		panic("加密文件不能是目录")
	}
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

	return nil
}

func doEncryptFile(key string, fileName string) error {
	stat, err := os.Stat(fileName)
	if stat.IsDir() {
		panic("加密文件不能是目录")
	}
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

	return nil
}

func main() {
	flag.StringVar(&flags.mode, "m", "", "模式")
	flag.StringVar(&flags.key, "k", "", "秘钥（加密==解密）")
	flag.StringVar(&flags.fileName, "file", "", "加解密文件")
	flag.StringVar(&flags.data, "data", "", "加解密数据")
	flag.Parse()

	//flags.mode = "de"
	//flags.key = "123"
	//flags.fileName = "G:\\tmp"
	//flags.data = "bdC6IQA/ygjDpYJ6WrHjLSZrSo7J2aDackOx"

	var command string
	check := true
	//校验模式不能为空
	if flags.mode != "" {
		if flags.mode == "en" {
			command = "encrypt"
		} else if flags.mode == "de" {
			command = "decrypt"
		} else {
			fmt.Println("错误模式：", flags.mode)
			check = false
		}
	} else {
		fmt.Println("模式不能为空")
		check = false
	}

	//校验加密key不能为空
	if flags.key == "" {
		fmt.Println("key不能为空")
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
		fmt.Println("加密数据不能为空")
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

	switch command {
	case "encrypt":
		{
			if isFile {
				//文件加密
				if err := doEncryptFile(key, filename); err != nil {
					fmt.Println("Error encrypting file:", err)
					return
				}
				fmt.Println("File encrypted successfully.")
			} else if isData {
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
				if err := doDecryptFile(key, filename); err != nil {
					fmt.Println("Error decrypting file:", err)
					return
				}
				fmt.Println("File decrypted successfully.")
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

				fmt.Println("Decrypted text: ", string(decryptedText))
			}
		}

	default:
		fmt.Println("Invalid command:", command)
	}
}
