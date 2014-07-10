package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

var (
	DetectLanguage = detectLanguage
)

func detectLanguage(text string) string {
	var rq = map[string]string{
		"document": text,
	}
	b, err := json.Marshal(rq)
	if err != nil {
		logger.Println(err.Error())
		return ""
	}
	url := "https://services.open.xerox.com/RestOp/LanguageIdentifier/GetLanguageForString"
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	// XXX: the docs say I need to specify Content-Length, but in practice I
	// see that it works without it:
	//req.Header.Add("Content-Length", fmt.Sprintf("%d", len(string(b))))
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err.Error())
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Println(err.Error())
		return ""
	}
	return string(body)
}
