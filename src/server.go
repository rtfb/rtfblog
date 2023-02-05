package rtfblog

// server contains a collection of dependencies needed to run the HTTP server.
type server struct {
	cryptoHelper CryptoHelper
	gctx         globalContext
}

func newServer(cryptoHelper CryptoHelper, gctx globalContext) server {
	return server{
		cryptoHelper: cryptoHelper,
		gctx:         gctx,
	}
}
