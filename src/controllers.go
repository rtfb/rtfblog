package main

import (
    "bytes"
    "errors"
    "fmt"
    "html/template"
    "net/http"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/sessions"
    "github.com/rtfb/httpbuf"
)

type Handler func(http.ResponseWriter, *http.Request, *Context) error

var (
    cachedTemplates = map[string]*template.Template{}
    cachedMutex     sync.Mutex
    funcs           = template.FuncMap{
        "dict": dict,
    }
    tmplDir = "tmpl"
)

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    tm := time.Now().UTC()
    defer logRequest(req, tm)
    //create the context
    ctx, err := NewContext(req)
    if err != nil {
        InternalError(w, req, "new context err: "+err.Error())
        return
    }
    //defer ctx.Close()
    // We're using httpbuf here to satisfy an unobvious requirement:
    // sessions.Save() *must* be called before anything is written to
    // ResponseWriter. So we pass this buffer in place of writer here, then
    // call Save() and finally apply the buffer to the real writer.
    buf := new(httpbuf.Buffer)
    err = h(buf, req, ctx)
    if err != nil {
        InternalError(w, req, "buffer err: "+err.Error())
        return
    }
    //save the session
    if err = sessions.Save(req, w); err != nil {
        InternalError(w, req, "session save err: "+err.Error())
        return
    }
    buf.Apply(w)
}

func stripPort(s string) string {
    idx := strings.LastIndex(s, ":")
    if idx == -1 {
        return s
    }
    return s[:idx]
}

func getIPAddress(req *http.Request) string {
    hdrForwardedFor := req.Header.Get("X-Forwarded-For")
    if hdrForwardedFor == "" {
        return stripPort(req.RemoteAddr)
    }
    // X-Forwarded-For is potentially a list of addresses separated with ","
    parts := strings.Split(hdrForwardedFor, ",")
    for i, p := range parts {
        parts[i] = strings.TrimSpace(p)
    }
    // TODO: should return first non-local address
    return parts[0]
}

func logRequest(req *http.Request, sTime time.Time) {
    var logEntry bytes.Buffer
    requestPath := req.URL.Path
    duration := time.Now().Sub(sTime)
    ip := getIPAddress(req)
    format := "%s - \033[32;1m %s %s\033[0m - %v"
    fmt.Fprintf(&logEntry, format, ip, req.Method, requestPath, duration)
    if len(req.Form) > 0 {
        fmt.Fprintf(&logEntry, " - \033[37;1mParams: %v\033[0m\n", req.Form)
    }
    logger.Print(logEntry.String())
}

//InternalError is what is called when theres an error processing something
func InternalError(w http.ResponseWriter, req *http.Request, err string) error {
    logger.Printf("Error serving request page: %s", err)
    return PerformStatus(w, req, http.StatusInternalServerError)
}

//PerformStatus runs the passed in status on the request and calls the appropriate block
func PerformStatus(w http.ResponseWriter, req *http.Request, status int) error {
    if status == 404 || status == 403 {
        html := fmt.Sprintf("%d.html", status)
        return Tmpl(html).Execute(w, map[string]interface{}{})
    }
    w.Write([]byte(fmt.Sprintf(L10n("HTTP Error %d"), status)))
    return nil
}

func reverse(name string, things ...interface{}) string {
    //convert the things to strings
    strs := make([]string, len(things))
    for i, th := range things {
        strs[i] = fmt.Sprint(th)
    }
    //grab the route
    u, err := Router.GetRoute(name).URL(strs...)
    if err != nil {
        logger.Printf("reverse (%s %v): %s", name, things, err.Error())
        return "#"
    }
    return u.Path
}

func checkPerm(handler Handler) Handler {
    return func(w http.ResponseWriter, req *http.Request, ctx *Context) error {
        if !ctx.AdminLogin {
            PerformStatus(w, req, http.StatusForbidden)
            return nil
        }
        handler(w, req, ctx)
        return nil
    }
}

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