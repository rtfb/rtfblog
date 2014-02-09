package main

import (
    "fmt"
    "net/http"

    "github.com/goods/httpbuf"
    "github.com/gorilla/sessions"
)

type Handler func(http.ResponseWriter, *http.Request, *Context) error

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

//InternalError is what is called when theres an error processing something
func InternalError(w http.ResponseWriter, req *http.Request, err string) error {
    logger.Printf("Error serving request page: %s", err)
    return PerformStatus(w, req, http.StatusInternalServerError)
}

//PerformStatus runs the passed in status on the request and calls the appropriate block
func PerformStatus(w http.ResponseWriter, req *http.Request, status int) error {
    if status == 404 || status == 403 {
        render(w, fmt.Sprintf("%d", status), nil)
        return nil
    }
    w.Write([]byte(fmt.Sprintf("Error %d", status)))
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
