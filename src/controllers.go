package rtfblog

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/rtfb/httpbuf"
	"github.com/rtfb/rtfblog/src/assets"
)

type globalContext struct {
	Router *pat.Router
	Db     Data
	assets *assets.Bin
	Store  sessions.Store
}

type handlerFunc func(http.ResponseWriter, *http.Request, *Context) error

type handler struct {
	h     handlerFunc
	c     *globalContext
	logRq bool
	log   *slog.Logger
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now().UTC()
	if h.logRq {
		defer h.log.Info("request served",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.String("query", req.URL.RawQuery),
			slog.Duration("duration", time.Now().Sub(startTime)),
		)
	}
	//create the context
	ctx, err := NewContext(req, h.c)
	if err != nil {
		internalError(ctx, w, req, err, "New context err")
		return
	}
	defer func() {
		r := recover()
		if r != nil {
			var err error
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("Unknown error")
			}
			logger.Printf("%s: %s\n", err, debug.Stack())
			internalError(ctx, w, req, err, "Panic in handler")
		}
	}()
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

func internalError(c *Context, w http.ResponseWriter, req *http.Request, err error, prefix string) error {
	logger.Printf("%s: %s", prefix, err.Error())
	return performStatus(c, w, req, http.StatusInternalServerError)
}

// PerformStatus runs the passed in status on the request and calls the appropriate block
func performStatus(c *Context, w http.ResponseWriter, req *http.Request, status int) error {
	if status == 404 || status == 403 {
		html := fmt.Sprintf("%d.html", status)
		return tmpl(c, html).Execute(w, nil)
	}
	return performSimpleStatus(w, status)
}

func performSimpleStatus(w http.ResponseWriter, status int) error {
	fmt.Fprintf(w, L10n("HTTP Error %d\n"), status)
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
