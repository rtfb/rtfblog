package main

import (
    "encoding/json"
    "net/http"
)

func CheckCaptcha(input string) bool {
    return input == "dvylika"
}

func CaptchaHtml() string {
    html := `<p>
My Nazi spam-filter presumes everybody is a bot.<br />
Please solve this captcha to prove you're not:</p>
<input id="captcha-id" name="captcha-id" type="hidden" value="" />
<p class="captcha-prompt">
    8 + 4 =
    <input
        id="captcha-input"
        class="text"
        name="captcha"
        type="text"
        style="display: inline"
        placeholder="lietuviškai, žodžiu"
        />
</p>`
    return html
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
