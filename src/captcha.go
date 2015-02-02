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

type Deck struct {
	nextTask int
	tasks    []CaptchaTask
}

var (
	deck *Deck
)

func NewDeck() *Deck {
	var deck Deck
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
	deck.tasks = make([]CaptchaTask, 0, 0)
	for i, answer := range answers {
		task := CaptchaTask{
			Task:   fmt.Sprintf("9 + %d =", i+2),
			ID:     fmt.Sprintf("%d", 666+i),
			Answer: answer,
		}
		deck.tasks = append(deck.tasks, task)
	}
	return &deck
}

func init() {
	deck = NewDeck()
}

func (d *Deck) GetTask() *CaptchaTask {
	return &d.tasks[d.nextTask]
}

func (d *Deck) GetTaskByID(id string) *CaptchaTask {
	for _, t := range d.tasks {
		if t.ID == id {
			return &t
		}
	}
	return &d.tasks[0]
}

func (d *Deck) SetNextTask(task int) {
	if task < 0 {
		task = rand.Int() % len(d.tasks)
	}
	d.nextTask = task
}

func CheckCaptcha(task *CaptchaTask, input string) bool {
	return input == task.Answer
}

func WrongCaptchaReply(w http.ResponseWriter, req *http.Request, status string, task *CaptchaTask) {
	var response = map[string]interface{}{
		"status":       status,
		"captcha-id":   task.ID,
		"captcha-task": task.Task,
		"name":         req.FormValue("name"),
		"email":        req.FormValue("email"),
		"website":      req.FormValue("website"),
		"body":         req.FormValue("text"),
	}
	b, err := json.Marshal(response)
	if logger.LogIf(err) == nil {
		w.Write(b)
	}
}

func RightCaptchaReply(w http.ResponseWriter, redir string) {
	var response = map[string]interface{}{
		"status": "accepted",
		"redir":  redir,
	}
	b, err := json.Marshal(response)
	if logger.LogIf(err) == nil {
		w.Write(b)
	}
}
