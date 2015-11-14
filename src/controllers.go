package main

import (
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
	Store  sessions.Store
}

type HandlerFunc func(http.ResponseWriter, *http.Request, *Context) error

type Handler struct {
	h     HandlerFunc
	c     *GlobalContext
	logRq bool
}

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now().UTC()
	if h.logRq {
		defer logger.LogRq(req, startTime)
	}
	//create the context
	ctx, err := NewContext(req, h.c)
	if err != nil {
		InternalError(ctx, w, req, err, "New context err")
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
		InternalError(ctx, w, req, err, "Error in handler")
		return
	}
	//save the session
	if err = sessions.Save(req, w); err != nil {
		InternalError(ctx, w, req, err, "Session save err")
		return
	}
	buf.Apply(w)
}

func ServeRobots(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	http.ServeFile(w, req, filepath.Join(conf.Server.StaticDir, "robots.txt"))
	return nil
}

func ServeFavicon(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	http.ServeFile(w, req, conf.Server.Favicon)
	return nil
}

func InternalError(c *Context, w http.ResponseWriter, req *http.Request, err error, prefix string) error {
	logger.Printf("%s: %s", prefix, err.Error())
	return PerformStatus(c, w, req, http.StatusInternalServerError)
}

//PerformStatus runs the passed in status on the request and calls the appropriate block
func PerformStatus(c *Context, w http.ResponseWriter, req *http.Request, status int) error {
	if status == 404 || status == 403 {
		html := fmt.Sprintf("%d.html", status)
		return Tmpl(c, html).Execute(w, TmplMap{})
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
