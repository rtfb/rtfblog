package rtfblog

import (
	"golang.org/x/crypto/bcrypt"
)

type CryptoHelper interface {
	Encrypt(passwd string) (hash string, err error)
	Decrypt(hash, passwd []byte) error
}

type BcryptHelper struct{}

func (h BcryptHelper) Encrypt(passwd string) (hash string, err error) {
	cost := bcrypt.DefaultCost
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), cost)
	hash = string(hashBytes)
	return
}

func (h BcryptHelper) Decrypt(hash, passwd []byte) error {
	return bcrypt.CompareHashAndPassword(hash, passwd)
}
