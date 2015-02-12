package main

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/rtfb/httpbuf"
)

type GlobalContext struct {
	Router *pat.Router
	Db     Data
	Root   string // Root directory where the binary and all our data subdirectories reside
}

type HandlerFunc func(http.ResponseWriter, *http.Request, *Context) error

type Handler struct {
	h HandlerFunc
	c *GlobalContext
}

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	tm := time.Now().UTC()
	defer logRequest(req, tm)
	//create the context
	ctx, err := NewContext(req, h.c)
	if err != nil {
		InternalError(ctx, w, req, "new context err: "+err.Error())
		return
	}
	//defer ctx.Close()
	// We're using httpbuf here to satisfy an unobvious requirement:
	// sessions.Save() *must* be called before anything is written to
	// ResponseWriter. So we pass this buffer in place of writer here, then
	// call Save() and finally apply the buffer to the real writer.
	buf := new(httpbuf.Buffer)
	err = h.h(buf, req, ctx)
	if err != nil {
		InternalError(ctx, w, req, "Error in handler: "+err.Error())
		return
	}
	//save the session
	if err = sessions.Save(req, w); err != nil {
		InternalError(ctx, w, req, "session save err: "+err.Error())
		return
	}
	buf.Apply(w)
}

func ServeRobots(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	http.ServeFile(w, req, filepath.Join(conf.Get("staticdir"), "robots.txt"))
	return nil
}

func logRequest(req *http.Request, sTime time.Time) {
	var logEntry bytes.Buffer
	requestPath := req.URL.Path
	// TODO: remove this hack. Make Handler configurable logging-wise, specify
	// it when setting up the routes
	if requestPath == "/favicon.ico" {
		return
	}
	duration := time.Now().Sub(sTime)
	ip := GetIPAddress(req)
	format := "%s - \033[32;1m %s %s\033[0m - %v"
	fmt.Fprintf(&logEntry, format, ip, req.Method, requestPath, duration)
	if len(req.Form) > 0 {
		fmt.Fprintf(&logEntry, " - \033[37;1mParams: %v\033[0m\n", req.Form)
	}
	logger.Print(logEntry.String())
}

//InternalError is what is called when theres an error processing something
func InternalError(c *Context, w http.ResponseWriter, req *http.Request, err string) error {
	logger.Printf("Error serving request page: %s", err)
	return PerformStatus(c, w, req, http.StatusInternalServerError)
}

//PerformStatus runs the passed in status on the request and calls the appropriate block
func PerformStatus(c *Context, w http.ResponseWriter, req *http.Request, status int) error {
	if status == 404 || status == 403 {
		html := fmt.Sprintf("%d.html", status)
		return Tmpl(c, html).Execute(w, map[string]interface{}{})
	}
	w.Write([]byte(fmt.Sprintf(L10n("HTTP Error %d"), status)))
	return nil
}

func (c *Context) routeByName(name string, things ...interface{}) string {
	//convert the things to strings
	strs := make([]string, len(things))
	for i, th := range things {
		strs[i] = fmt.Sprint(th)
	}
	//grab the route
	u, err := c.Router.GetRoute(name).URL(strs...)
	if err != nil {
		logger.LogIff(err, "routeByName(%s %v)", name, things)
		return "#"
	}
	return u.Path
}

func checkPerm(handler *Handler) *Handler {
	return &Handler{
		h: func(w http.ResponseWriter, req *http.Request, ctx *Context) error {
			if !ctx.AdminLogin {
				PerformStatus(ctx, w, req, http.StatusForbidden)
				return nil
			}
			return handler.h(w, req, ctx)
		},
		c: handler.c,
	}
}
