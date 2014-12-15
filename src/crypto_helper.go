package main

import (
	"golang.org/x/crypto/bcrypt"
)

var (
	Decrypt = decrypt
)

func Encrypt(passwd string) (hash string, err error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	hash = string(hashBytes)
	return
}

func decrypt(hash, passwd []byte) error {
	return bcrypt.CompareHashAndPassword(hash, passwd)
}
