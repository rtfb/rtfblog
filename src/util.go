package main

import (
	"database/sql"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"code.google.com/p/go.crypto/bcrypt"
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

func insertTestAuthor(db *sql.DB, uname, passwd, fullname, email, www string) error {
	passwdHash, err := Encrypt(passwd)
	if err != nil {
		return err
	}
	stmt, _ := db.Prepare(`insert into author
		(disp_name, passwd, full_name, email, www)
		values ($1, $2, $3, $4, $5)`)
	defer stmt.Close()
	stmt.Exec(uname, passwdHash, fullname, email, www)
	return nil
}

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
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}

func Basedir() string {
	basedir, _ := filepath.Split(PathToFullPath(os.Args[0]))
	return basedir
}
