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

type globalContext struct {
	Router *pat.Router
	Db     Data
	assets *AssetBin
	Store  sessions.Store
}

type handlerFunc func(http.ResponseWriter, *http.Request, *Context) error

type handler struct {
	h     handlerFunc
	c     *globalContext
	logRq bool
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now().UTC()
	if h.logRq {
		defer logger.LogRq(req, startTime)
	}
	//create the context
	ctx, err := NewContext(req, h.c)
	if err != nil {
		internalError(ctx, w, req, err, "New context err")
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
		internalError(ctx, w, req, err, "Error in handler")
		return
	}
	//save the session
	if err = sessions.Save(req, w); err != nil {
		internalError(ctx, w, req, err, "Session save err")
		return
	}
	buf.Apply(w)
}

func serveStaticFile(w http.ResponseWriter, req *http.Request, ctx *Context, fileName string) error {
	filePath := filepath.Join(conf.Server.StaticDir, fileName)
	file, err := ctx.assets.Open(filePath)
	defer file.Close()
	if err != nil {
		return err
	}
	var zero time.Time
	http.ServeContent(w, req, fileName, zero, file)
	return nil
}

func serveRobots(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	return serveStaticFile(w, req, ctx, "robots.txt")
}

func serveFavicon(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if conf.Server.Favicon == "" {
		return performSimpleStatus(w, http.StatusNotFound)
	}
	return serveStaticFile(w, req, ctx, conf.Server.Favicon)
}

func internalError(c *Context, w http.ResponseWriter, req *http.Request, err error, prefix string) error {
	logger.Printf("%s: %s", prefix, err.Error())
	return performStatus(c, w, req, http.StatusInternalServerError)
}

//PerformStatus runs the passed in status on the request and calls the appropriate block
func performStatus(c *Context, w http.ResponseWriter, req *http.Request, status int) error {
	if status == 404 || status == 403 {
		html := fmt.Sprintf("%d.html", status)
		return tmpl(c, html).Execute(w, nil)
	}
	return performSimpleStatus(w, status)
}

func performSimpleStatus(w http.ResponseWriter, status int) error {
	w.Write([]byte(fmt.Sprintf(L10n("HTTP Error %d\n"), status)))
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
