// deepseek assissted
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// AES加密函数（CBC模式）
func AESEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 填充明文到块大小的倍数
	plaintext = PKCS7Pad(plaintext, aes.BlockSize)

	// 创建密文缓冲区，额外空间存储IV
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))

	// 生成随机IV
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// 创建CBC加密器
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// AES解密函数（CBC模式）
func AESDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	// 提取IV
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// 检查密文长度是否是块大小的倍数
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	// 创建CBC解密器
	mode := cipher.NewCBCDecrypter(block, iv)

	// 解密
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除填充
	plaintext = PKCS7Unpad(plaintext)

	return plaintext, nil
}

// PKCS7填充
func PKCS7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// PKCS7去填充
func PKCS7Unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	padding := int(data[length-1])
	if padding < 1 || padding > aes.BlockSize {
		return data
	}
	for i := length - padding; i < length; i++ {
		if int(data[i]) != padding {
			return data
		}
	}
	return data[:length-padding]
}

// 生成随机密钥
func GenerateKey(keySize int) ([]byte, error) {
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return nil, errors.New("key size must be 16, 24, or 32 bytes")
	}
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// 使用GCM模式的加密（推荐，提供认证）
func AESGCMEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// 使用GCM模式的解密
func AESGCMDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func main() {
	// 生成256位密钥（32字节）
	key, err := GenerateKey(32)
	if err != nil {
		panic(err)
	}

	fmt.Printf("密钥: %s\n", hex.EncodeToString(key))
	fmt.Println()

	// 原始数据
	originalText := "Hello, AES加密解密测试!"
	fmt.Printf("原始文本: %s\n", originalText)
	fmt.Println()

	// 测试CBC模式
	fmt.Println("=== CBC模式加解密 ===")
	ciphertext, err := AESEncrypt([]byte(originalText), key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("加密结果: %s\n", hex.EncodeToString(ciphertext))

	decrypted, err := AESDecrypt(ciphertext, key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("解密结果: %s\n", string(decrypted))
	fmt.Println()

	// 测试GCM模式（推荐）
	fmt.Println("=== GCM模式加解密（推荐） ===")
	gcmCiphertext, err := AESGCMEncrypt([]byte(originalText), key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("GCM加密结果: %s\n", hex.EncodeToString(gcmCiphertext))

	gcmDecrypted, err := AESGCMDecrypt(gcmCiphertext, key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("GCM解密结果: %s\n", string(gcmDecrypted))
}
