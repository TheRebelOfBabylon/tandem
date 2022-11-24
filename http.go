package tandem

import (
	"bufio"
	"net"
	"net/http"

	"github.com/rs/zerolog"
)

type CustomHandler struct {
	zerolog.Logger
	handlerFunc LogHandleFunc
}

type LogHandleFunc = func(w http.ResponseWriter, r *http.Request, log zerolog.Logger)

// ServeHTTP satisfies the http.Handler interface
func (c *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handlerFunc(w, r, c.Logger)
}

// CustomHandlerFactory creates new handlers automatically instead of creating them for each endpoint
func CustomHandlerFactory(log zerolog.Logger) func(LogHandleFunc) *CustomHandler {
	return func(hf LogHandleFunc) *CustomHandler {
		return &CustomHandler{Logger: log, handlerFunc: hf}
	}
}

type ResponseObserver struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

// Write satisfies the ResponseWriter interface
func (o *ResponseObserver) Write(p []byte) (n int, err error) {
	if !o.wroteHeader {
		o.WriteHeader(http.StatusOK)
	}
	n, err = o.ResponseWriter.Write(p)
	o.written += int64(n)
	return
}

// WriteHeader satisfies the ResponseWriter interface
func (o *ResponseObserver) WriteHeader(code int) {
	if o.wroteHeader {
		return
	}
	o.ResponseWriter.WriteHeader(code)
	o.wroteHeader = true
	o.status = code
}

// Hijack satisfies the Hijack interface
func (o *ResponseObserver) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := o.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, ErrHijackUnsupported
	}
	return h.Hijack()
}
