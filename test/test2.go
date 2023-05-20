package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"
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

func main() {
	// 32字节的密钥（256位）
	key := []byte("0123456789123456")

	// 要加密的文件
	filename := "G:\\tmp\\DG5461432_x64-22.zip"

	// 加密文件
	//err := encryptFile(filename, key)
	//if err != nil {
	//	log.Fatal("加密文件失败:", err)
	//}
	//log.Println("加密文件成功")

	// 解密文件
	err := decryptFile(filename+".encrypted", key)
	if err != nil {
		log.Fatal("解密文件失败:", err)
	}
	log.Println("解密文件成功")
}
