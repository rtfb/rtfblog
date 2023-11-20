package rtfblog

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log/slog"
	"net/http"
	"time"
)

type LangDetector interface {
	Detect(text string, log *slog.Logger) string
}

type XeroxLangDetector struct{}

var (
	langDetector LangDetector = new(XeroxLangDetector)
)

func (d XeroxLangDetector) Detect(text string, log *slog.Logger) string {
	var rq = map[string]string{
		"document": text,
	}
	b, err := json.Marshal(rq)
	if err != nil {
		log.Error("XeroxLangDetector.Detect json.Marshal", E(err))
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
		log.Error("XeroxLangDetector.Detect http request", E(err))
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("XeroxLangDetector.Detect read response", E(err))
		return ""
	}
	return string(body)
}

func DetectLanguageWithTimeout(text string, log *slog.Logger) string {
	c := make(chan string, 1)
	go func() {
		c <- langDetector.Detect(text, log)
	}()
	select {
	case lang := <-c:
		return lang
	case <-time.After(1500 * time.Millisecond):
		return "timedout"
	}
}
