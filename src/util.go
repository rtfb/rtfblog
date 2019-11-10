package rtfblog

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

func fileExistsNoErr(path string) bool {
	exists, err := fileExists(path)
	if err != nil {
		return false
	}
	return exists
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func pathToFullPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func bindir() string {
	basedir, _ := filepath.Split(pathToFullPath(os.Args[0]))
	return basedir
}

func md5Hash(s string) string {
	hash := md5.New()
	hash.Write([]byte(strings.ToLower(s)))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func capitalize(s string) string {
	firstRune, width := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(firstRune)) + s[width:]
}

func encryptBcrypt(passwd []byte) (hash string, err error) {
	h, err := bcrypt.GenerateFromPassword(passwd, bcrypt.DefaultCost)
	hash = string(h)
	return
}

func censorPostgresConnStr(conn string) string {
	parts := strings.Split(conn, " ")
	newParts := []string{}
	for _, part := range parts {
		if strings.HasPrefix(part, "password=") {
			newParts = append(newParts, "password=***")
		} else {
			newParts = append(newParts, part)
		}
	}
	return strings.Join(newParts, " ")
}
