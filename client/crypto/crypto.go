package crypto

import (
	"client/config"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// DecryptWithPrivateKey 使用RSA私钥解密数据
func DecryptWithPrivateKey(encryptedData string) (string, error) {
	// 加载配置文件
	config, err := config.LoadConfig("config.toml")
	if err != nil {
		return "", fmt.Errorf("加载配置文件失败: %v", err)
	}

	// 从配置获取私钥
	privateKeyPEM := config.Client.PrivateKey
	if privateKeyPEM == "" {
		return "", fmt.Errorf("配置中缺少客户端私钥")
	}

	// 解码PEM格式的私钥
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无法解码私钥")
	}

	// 解析私钥
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析私钥失败: %v", err)
	}

	// 解码Base64编码的加密数据
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("解码加密数据失败: %v", err)
	}

	// 使用私钥解密数据
	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	if err != nil {
		return "", fmt.Errorf("解密失败: %v", err)
	}

	return string(plaintext), nil
}
