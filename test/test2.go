package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
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

func main() {

	// 要加密的文件
	//filename := "G:\\tmp\\DG5461432_x64.zip"
	filename := "G:\\Project\\Java\\other\\MyDemoCode\\src\\main\\java\\top\\yms\\recent\\c202209\\Code16.java"

	isBinary := isBinaryFile(filename)

	if isBinary {
		fmt.Println("文件是二进制文件")
	} else {
		fmt.Println("文件是文本文件")
	}

}
