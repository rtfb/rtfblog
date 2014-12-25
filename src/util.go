package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

func MkLogger(fname string) *log.Logger {
	f, err := os.Create(fname)
	if err != nil {
		panic("MkLogger: " + err.Error())
	}
	return log.New(f, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func GetHomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
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
