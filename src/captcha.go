package main

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "net/http"
)

type CaptchaTask struct {
    Task   string
    ID     string
    Answer string
}

var (
    nextTask     int
    CaptchaTasks []CaptchaTask
)

func init() {
    answers := []string{
        "vienuolika",
        "dvylika",
        "trylika",
        "keturiolika",
        "penkiolika",
        "šešiolika",
        "septyniolika",
        "aštuoniolika",
        "devyniolika",
    }
    CaptchaTasks = make([]CaptchaTask, 0, 0)
    for i, answer := range answers {
        task := CaptchaTask{
            Task:   fmt.Sprintf("9 + %d =", i+2),
            ID:     fmt.Sprintf("%d", 666+i),
            Answer: answer,
        }
        CaptchaTasks = append(CaptchaTasks, task)
    }
}

func GetTask() *CaptchaTask {
    return &CaptchaTasks[nextTask]
}

func GetTaskByID(id string) *CaptchaTask {
    for _, t := range CaptchaTasks {
        if t.ID == id {
            return &t
        }
    }
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

func WrongCaptchaReply(w http.ResponseWriter, req *http.Request, status string, task *CaptchaTask) {
    var response = map[string]interface{}{
        "status":     status,
        "captcha-id": task.ID,
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
