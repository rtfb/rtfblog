package main

import (
    "log"
    "os"
    "os/user"

    "code.google.com/p/go.crypto/bcrypt"
)

func Encrypt(passwd string) (hash string, err error) {
    hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
    hash = string(hashBytes)
    return
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
