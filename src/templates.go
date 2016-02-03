package main

import (
	"errors"
	"html/template"
	"path/filepath"
	"sync"
)

const (
	tmpl = "tmpl"
)

type TmplMap map[string]interface{}

var (
	cachedTemplates = map[string]*template.Template{}
	cachedMutex     sync.Mutex
	funcs           = template.FuncMap{
		"dict": dict,
	}
)

func dict(values ...interface{}) (TmplMap, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(TmplMap, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func AddTemplateFunc(name string, f interface{}) {
	funcs[name] = f
}

func Tmpl(c *Context, name string) *template.Template {
	cachedMutex.Lock()
	defer cachedMutex.Unlock()
	if t, ok := cachedTemplates[name]; ok {
		return t
	}
	t := template.New("base.html").Funcs(funcs)
	for _, s := range []string{
		"base.html",
		"sidebar.html",
		"post-title.html",
		"header.html",
		"author.html",
		"captcha.html",
		name,
	} {
		t = template.Must(t.Parse(string(c.assets.MustLoad(filepath.Join(tmpl, s)))))
	}
	cachedTemplates[name] = t
	return t
}
