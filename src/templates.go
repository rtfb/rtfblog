package main

import (
	"errors"
	"html/template"
	"path/filepath"
	"sync"
)

var (
	cachedTemplates = map[string]*template.Template{}
	cachedMutex     sync.Mutex
	funcs           = template.FuncMap{
		"dict": dict,
	}
	tmplDir = "tmpl"
)

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func Tmpl(name string) *template.Template {
	cachedMutex.Lock()
	defer cachedMutex.Unlock()
	if t, ok := cachedTemplates[name]; ok {
		return t
	}
	t := template.New("base.html").Funcs(funcs)
	t = template.Must(t.ParseFiles(
		filepath.Join(tmplDir, "base.html"),
		filepath.Join(tmplDir, "sidebar.html"),
		filepath.Join(tmplDir, "post-title.html"),
		filepath.Join(tmplDir, "header.html"),
		filepath.Join(tmplDir, "author.html"),
		filepath.Join(tmplDir, "captcha.html"),
		filepath.Join(tmplDir, name),
	))
	cachedTemplates[name] = t
	return t
}
