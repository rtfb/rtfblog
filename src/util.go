package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

func GetHomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

func FileExistsNoErr(path string) bool {
	exists, err := FileExists(path)
	if err != nil {
		return false
	}
	return exists
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func PathToFullPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func Bindir() string {
	basedir, _ := filepath.Split(PathToFullPath(os.Args[0]))
	return basedir
}

func Md5Hash(s string) string {
	hash := md5.New()
	hash.Write([]byte(strings.ToLower(s)))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func Capitalize(s string) string {
	firstRune, width := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(firstRune)) + s[width:]
}

func EncryptBcrypt(passwd []byte) (hash string, err error) {
	h, err := bcrypt.GenerateFromPassword(passwd, bcrypt.DefaultCost)
	hash = string(h)
	return
}
