package main

import (
    "encoding/json"
    "math/rand"
    "net/http"

    "github.com/lye/mustache"
)

type CaptchaTask struct {
    Task   string
    Id     string
    Answer string
}

var (
    nextTask int = 0
    CaptchaTasks = []CaptchaTask{{
        Task:   "8 + 4 =",
        Id:     "666",
        Answer: "dvylika",
    }}
)

func GetTask() *CaptchaTask {
    return &CaptchaTasks[nextTask]
}

func GetTaskById(id string) *CaptchaTask {
    // TODO: impl
    return &CaptchaTasks[0]
}

func SetNextTask(task int) {
    if task < 0 {
        task = rand.Int() % len(CaptchaTasks)
    }
    nextTask = task
}

func CheckCaptcha(task *CaptchaTask, input string) bool {
    return input == task.Answer
}

func CaptchaHtml(task *CaptchaTask) string {
    return mustache.RenderFile("tmpl/captcha.mustache", map[string]interface{}{
        "CaptchaTask": task.Task,
    })
}

func WrongCaptchaReply(w http.ResponseWriter, req *http.Request, status string, task *CaptchaTask) {
    var response = map[string]interface{}{
        "status":     status,
        "captcha-id": task.Id,
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
