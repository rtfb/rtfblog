package util

import (
    "crypto/rand"
    "crypto/sha1"
    "encoding/base64"
    "fmt"
    "io"
)

func Encrypt(passwd string) (salt, hash string) {
    b := make([]byte, 16)
    n, err := io.ReadFull(rand.Reader, b)
    if n != len(b) || err != nil {
        fmt.Println("error:", err)
        return
    }
    salt = base64.URLEncoding.EncodeToString(b)
    sha := sha1.New()
    sha.Write([]byte(salt + passwd))
    hash = base64.URLEncoding.EncodeToString(sha.Sum(nil))
    return
}
