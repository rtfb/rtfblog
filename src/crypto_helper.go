package main

import (
	"golang.org/x/crypto/bcrypt"
)

type CryptoHelper interface {
	Encrypt(passwd string) (hash string, err error)
	Decrypt(hash, passwd []byte) error
}

type BcryptHelper struct{}

var (
	cryptoHelper CryptoHelper = new(BcryptHelper)
)

func (h BcryptHelper) Encrypt(passwd string) (hash string, err error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	hash = string(hashBytes)
	return
}

func (h BcryptHelper) Decrypt(hash, passwd []byte) error {
	return bcrypt.CompareHashAndPassword(hash, passwd)
}
