package util

import (
    "crypto/rand"
    "crypto/sha1"
    "encoding/base64"
    "io"
    "log"
    "os"
)

func SaltAndPepper(salt, passwd string) string {
    sha := sha1.New()
    sha.Write([]byte(salt + passwd))
    return base64.URLEncoding.EncodeToString(sha.Sum(nil))
}

func Encrypt(passwd string) (salt, hash string, err error) {
    b := make([]byte, 16)
    n, err := io.ReadFull(rand.Reader, b)
    if n != len(b) || err != nil {
        return
    }
    salt = base64.URLEncoding.EncodeToString(b)
    hash = SaltAndPepper(salt, passwd)
    return
}

func MkLogger(fname string) *log.Logger {
    f, err := os.Create(fname)
    if err != nil {
        panic("MkLogger: " + err.Error())
    }
    return log.New(f, "", log.Ldate|log.Ltime|log.Lshortfile)
}
