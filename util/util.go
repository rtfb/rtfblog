package util

import (
    "crypto/rand"
    "crypto/sha1"
    "encoding/base64"
    "fmt"
    "io"
)

func SaltAndPepper(salt, passwd string) string {
    sha := sha1.New()
    sha.Write([]byte(salt + passwd))
    return base64.URLEncoding.EncodeToString(sha.Sum(nil))
}

func Encrypt(passwd string) (salt, hash string) {
    b := make([]byte, 16)
    n, err := io.ReadFull(rand.Reader, b)
    if n != len(b) || err != nil {
        fmt.Println("error:", err)
        return
    }
    salt = base64.URLEncoding.EncodeToString(b)
    hash = SaltAndPepper(salt, passwd)
    return
}
