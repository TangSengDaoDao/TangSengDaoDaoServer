package wkrsa

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

// Sign rsa签名
// pemPrivKey 私钥key 类似 -----BEGIN RSA PRIVATE KEY-----
// xxxx
// -----END RSA PRIVATE KEY-----
func SignWithMD5(data []byte, pemPrivKey []byte) (string, error) {
	hashMd5 := md5.Sum(data)
	hashed := hashMd5[:]
	block, _ := pem.Decode(pemPrivKey)
	if block == nil {
		return "", errors.New("private key error")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.MD5, hashed)
	return base64.StdEncoding.EncodeToString(signature), err
}
