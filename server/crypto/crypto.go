package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

// GenerateRandomChallenge 生成随机挑战字符串
func GenerateRandomChallenge() (string, error) {
	// 生成32字节的随机数
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// 将随机字节转换为base64字符串
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// EncryptWithPublicKey 使用RSA公钥加密数据
func EncryptWithPublicKey(data string, publicKeyPEM string) (string, error) {
	// 解码PEM格式的公钥
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", errors.New("failed to decode PEM block containing public key")
	}

	// 解析公钥
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 类型断言为RSA公钥
	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("not an RSA public key")
	}

	// 加密数据
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(data))
	if err != nil {
		return "", err
	}

	// 将加密后的数据转换为base64字符串
	return base64.StdEncoding.EncodeToString(encrypted), nil
}
