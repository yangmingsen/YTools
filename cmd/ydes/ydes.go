package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

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

var flags struct {
	mode        string
	key         string
	fileName    string
}

func getUsage() {
	flag.Usage()
	fmt.Println("加密文件: ydes -m en -k 123 -file ./hello.txt")
	fmt.Println("解密文件: ydes -m de -k 123 -file ./hello.txt.decrypted")
}

func main() {
	flag.StringVar(&flags.mode, "m","","模式")
	flag.StringVar(&flags.key, "k","","秘钥（加密==解密）")
	flag.StringVar(&flags.fileName, "file","","加密文件")
	flag.Parse()

	var command string
	check := true
	if flags.mode != "" {
		if flags.mode == "en" {
			command = "encrypt"
		} else if flags.mode == "de" {
			command = "decrypt"
		} else {
			fmt.Println("错误模式：",flags.mode)
			check = false
		}
	} else {
		fmt.Println("模式不能为空")
		check = false
	}

	if flags.key == "" {
		fmt.Println("key不能为空")
		check = false
	} else {
		if len(flags.key) < 16 {
			for kLen := len(flags.key); kLen <16; kLen++ {
				flags.key = flags.key+"0"
			}
			fmt.Println("new Key: ", flags.key)
		}
	}

	if flags.fileName == "" {
		fmt.Println("加密文件不能为空")
		check = false
	}

	if !check {
		getUsage()
		return
	} else {
		fmt.Println(flags)
	}

	key := []byte(flags.key)
	filename := flags.fileName

	switch command {
	case "encrypt":
		if err := encryptFile(key, filename); err != nil {
			fmt.Println("Error encrypting file:", err)
			return
		}
		fmt.Println("File encrypted successfully.")
	case "decrypt":
		if err := decryptFile(key, filename); err != nil {
			fmt.Println("Error decrypting file:", err)
			return
		}
		fmt.Println("File decrypted successfully.")
	default:
		fmt.Println("Invalid command:", command)
	}
}
