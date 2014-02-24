package main

import (
    "encoding/json"
    "net/http"

    "github.com/lye/mustache"
)

func CheckCaptcha(input string) bool {
    return input == "dvylika"
}

func CaptchaHtml() string {
    return mustache.RenderFile("tmpl/captcha.mustache", map[string]interface{}{
        "CaptchaTask": "8 + 4 =",
    })
}

func WrongCaptchaReply(w http.ResponseWriter, req *http.Request, status string) {
    var response = map[string]interface{}{
        "status":     status,
        "captcha-id": "666",
        "name":       req.FormValue("name"),
        "email":      req.FormValue("email"),
        "website":    req.FormValue("website"),
        "body":       req.FormValue("text"),
    }
    b, err := json.Marshal(response)
    if err != nil {
        logger.Println(err.Error())
        return
    }
    w.Write(b)
}

func RightCaptchaReply(w http.ResponseWriter, redir string) {
    var response = map[string]interface{}{
        "status": "accepted",
        "redir":  redir,
    }
    b, err := json.Marshal(response)
    if err != nil {
        logger.Println(err.Error())
        return
    }
    w.Write(b)
}
