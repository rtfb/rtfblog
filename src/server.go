package rtfblog

import (
	"net/http"
	"path/filepath"
	"time"
)

// server contains a collection of dependencies needed to run the HTTP server.
type server struct {
	cryptoHelper CryptoHelper
	gctx         globalContext
	conf         Config
}

func newServer(
	cryptoHelper CryptoHelper,
	gctx globalContext,
	conf Config,
) server {
	return server{
		cryptoHelper: cryptoHelper,
		gctx:         gctx,
		conf:         conf,
	}
}

func (s *server) serveStaticFile(w http.ResponseWriter, req *http.Request, ctx *Context, fileName string) error {
	filePath := filepath.Join(s.conf.Server.StaticDir, fileName)
	file, err := ctx.assets.Open(filePath)
	defer file.Close()
	if err != nil {
		return err
	}
	var zero time.Time
	http.ServeContent(w, req, fileName, zero, file)
	return nil
}

func (s *server) serveRobots(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	return s.serveStaticFile(w, req, ctx, "robots.txt")
}

func (s *server) serveFavicon(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if s.conf.Server.Favicon == "" {
		return performSimpleStatus(w, http.StatusNotFound)
	}
	return s.serveStaticFile(w, req, ctx, s.conf.Server.Favicon)
}
