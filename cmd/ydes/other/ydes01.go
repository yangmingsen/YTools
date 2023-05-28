package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

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

func main() {
	key := []byte("0123456789abcdef") // 16-byte key

	plaintext := []byte("Hello, World!")

	// 加密
	ciphertext, err := encryptText(key, plaintext)
	if err != nil {
		fmt.Println("Encryption error:", err)
		return
	}

	base64Str := base64.StdEncoding.EncodeToString(ciphertext)
	fmt.Println("Ciphertext:", base64Str)

	decodeString, _ := base64.StdEncoding.DecodeString(base64Str)

	// 解密
	decryptedText, err := decryptText(key, decodeString)
	if err != nil {
		fmt.Println("Decryption error:", err)
		return
	}

	fmt.Println("Decrypted text:", string(decryptedText))
}
