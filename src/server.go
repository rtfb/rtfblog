package rtfblog

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// server contains a collection of dependencies needed to run the HTTP server.
type server struct {
	cryptoHelper CryptoHelper
	gctx         globalContext
	conf         Config
	mets         metrics
}

type metrics struct {
	registry              *prometheus.Registry
	numRobotsServed       prometheus.Counter
	numForbiddenResponses prometheus.Counter
	numAdminRequests      prometheus.Counter
	numNonAdminRequests   prometheus.Counter
}

func initMetrics() metrics {
	reg := prometheus.NewRegistry()
	numRobotsServed := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_robots_txt_served",
		Help:      "The total number of times robots.txt was served",
	})
	numForbiddenResponses := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_403s",
		Help:      "The total number of Forbidden responses",
	})
	numAdminRequests := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_admin_reqs",
		Help:      "The total number of requests to admin area",
	})
	numNonAdminRequests := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Namespace: "rtfblog",
		Subsystem: "server",
		Name:      "num_non_admin_reqs",
		Help:      "The total number of requests to public pages",
	})
	return metrics{
		registry:              reg,
		numRobotsServed:       numRobotsServed,
		numForbiddenResponses: numForbiddenResponses,
		numAdminRequests:      numAdminRequests,
		numNonAdminRequests:   numNonAdminRequests,
	}
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
		mets:         initMetrics(),
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
	s.mets.numRobotsServed.Inc()
	return s.serveStaticFile(w, req, ctx, "robots.txt")
}

func (s *server) serveFavicon(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if s.conf.Server.Favicon == "" {
		return performSimpleStatus(w, http.StatusNotFound)
	}
	return s.serveStaticFile(w, req, ctx, s.conf.Server.Favicon)
}
