package titan

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"time"
)

token := "hello"

type Titan struct {
	common

	premiddleware []MiddlewareFunc
	middleware []MiddlewareFunc
	maxParam *int
	router *router
	listener      net.Listener
	addr string
	pool          sync.Pool
	doneChan      chan struct{}
	closeOnce     sync.Once
	mu            sync.Mutex

	// TitanErrorHandler allows setting custom error handler
	TitanErrorHandler TitanErrorHandler
	TLSConfig *tls.Config
}

// Route contains a handler and information for matching against requests.
type Route struct {
	Path string
	Name string
}

// GeminiError represents an error that occurred while handling a request.
TitanError struct {
	Code    Status
	Message string
}

// MiddlewareFunc defines a function to process middleware.
MiddlewareFunc func(HandlerFunc) HandlerFunc

// HandlerFunc defines a function to serve requests.
HandlerFunc func(Context) error

// GeminiErrorHandler is a centralized error handler.
TitanErrorHandler func(error, Context)

// Renderer is the interface that wraps the Render function.
Renderer interface {
	Render(io.Writer, string, interface{}, Context) error
}

type storeMap map[string]interface{}

// Common struct for Gig & Group.
type common struct{}

// MIME types.
const (
	MIMETextGemini            = "text/gemini"
	MIMETextGeminiCharsetUTF8 = "text/gemini; charset=UTF-8"
	MIMETextPlain             = "text/plain"
	MIMETextPlainCharsetUTF8  = "text/plain; charset=UTF-8"
)

const (
	// Version of Titan.
	Version = "0.9.8"
	// http://patorjk.com/software/taag/#p=display&f=Small%20Slant&t=gig
	banner = `Titan Server %s
`
)

// Errors that can be inherited from using NewErrorFrom.
var (
	/*ErrTemporaryFailure          = NewError(StatusTemporaryFailure, "Temporary Failure")
	ErrServerUnavailable         = NewError(StatusServerUnavailable, "Server Unavailable")
	ErrCGIError                  = NewError(StatusCGIError, "CGI Error")
	ErrProxyError                = NewError(StatusProxyError, "Proxy Error")
	ErrSlowDown                  = NewError(StatusSlowDown, "Slow Down")
	ErrPermanentFailure          = NewError(StatusPermanentFailure, "Permanent Failure")
	ErrNotFound                  = NewError(StatusNotFound, "Not Found")
	ErrGone                      = NewError(StatusGone, "Gone")
	ErrProxyRequestRefused       = NewError(StatusProxyRequestRefused, "Proxy Request Refused")
	ErrBadRequest                = NewError(StatusBadRequest, "Bad Request")
	ErrClientCertificateRequired = NewError(StatusClientCertificateRequired, "Client Certificate Required")
	ErrCertificateNotAuthorised  = NewError(StatusCertificateNotAuthorised, "Certificate Not Authorised")
	ErrCertificateNotValid       = NewError(StatusCertificateNotValid, "Certificate Not Valid")
	*/

	ErrRendererNotRegistered = errors.New("renderer not registered")
	ErrInvalidCertOrKeyType  = errors.New("invalid cert or key type, must be string or []byte")

	ErrServerClosed = errors.New("titan: Server closed")
)

// DefaultGeminiErrorHandler is the default HTTP error handler. It sends a JSON response
// with status code.
func DefaultGeminiErrorHandler(err error, c Context) { // TODO
	he, ok := err.(*TitanError)
	if !ok {
		he = &TitanError{
			Code:    StatusPermanentFailure,
			Message: err.Error(),
		}
	}

	code := he.Code
	message := he.Message

	debugPrintf("titan: handling error: %s", err)

	// Send response
	if !c.Response().Committed {
		err = c.NoContent(code, message)
		if err != nil {
			debugPrintf("titan: could not handle error: %s", err)
		}
	}
}

func New() *Titan {
	t := &Titan {
		TLSConfig: &tls.Config {
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.RequestClientCert,
		},
		maxParam: new(int),
		doneChan: make(chan struct{}),
	}

	return t
}

// Default returns a Gig instance with Logger and Recover middleware enabled.
func Default() *Titan {
	t := New()

	// Default middlewares
	t.Use(Logger(), Recover())

	return t
}

func (t *Titan) newContext(c net.Conn, u *url.URL, requestURI string, tls *tls.ConnectionState) Context {
	return &context{
		conn:       c,
		TLS:        tls,
		u:          u,
		requestURI: requestURI,
		response:   NewResponse(c),
		store:      make(storeMap),
		titan:        t,
		pvalues:    make([]string, *t.maxParam),
		handler:    NotFoundHandler,
	}
}

// Pre adds middleware to the chain which is run before router.
func (t *Titan) Pre(middleware ...MiddlewareFunc) {
	t.premiddleware = append(t.premiddleware, middleware...)
}

// Use adds middleware to the chain which is run after router.
func (t *Titan) Use(middleware ...MiddlewareFunc) {
	t.middleware = append(t.middleware, middleware...)
}

// Handle registers a new route for a path with matching handler in the router
// with optional route-level middleware.
func (t *Titan) Handle(path string, h HandlerFunc, m ...MiddlewareFunc) *Route {
	return t.add(path, h, m...)
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (t *Titan) Static(prefix, root string) *Route {
	if root == "" {
		root = "." // For security we want to restrict to CWD.
	}

	return t.static(prefix, root, t.Handle)
}

func (common) static(prefix, root string, get func(string, HandlerFunc, ...MiddlewareFunc) *Route) *Route {
	h := func(c Context) error {
		p, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return err
		}

		name := filepath.Join(root, path.Clean("/"+p)) // "/"+ for security

		return c.File(name)
	}

	if prefix == "/" {
		return get(prefix+"*", h)
	}

	return get(prefix+"/*", h)
}

func (common) file(path, file string, get func(string, HandlerFunc, ...MiddlewareFunc) *Route,
	m ...MiddlewareFunc) *Route {
	return get(path, func(c Context) error {
		return c.File(file)
	}, m...)
}

// File registers a new route with path to serve a static file with optional route-level middleware.
func (t *Titan) File(path, file string, m ...MiddlewareFunc) *Route {
	return t.file(path, file, t.Handle, m...)
}

func (t *Titan) add(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	name := handlerName(handler)

	t.router.add(path, func(c Context) error {
		h := handler
		// Chain middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			h = middleware[i](h)
		}
		return h(c)
	})

	r := &Route{
		Path: path,
		Name: name,
	}

	g.router.routes[path] = r

	return r
}

// Group creates a new router group with prefix and optional group-level middleware.
func (t *Titan) Group(prefix string, m ...MiddlewareFunc) (gg *Group) {
	gg = &Group{prefix: prefix, titan: t}
	gg.Use(m...)

	return
}


// URL generates a URL from handler.
func (t *Titan) URL(handler HandlerFunc, params ...interface{}) string {
	name := handlerName(handler)
	return t.Reverse(name, params...)
}

// Reverse generates an URL from route name and provided parameters.
func (t *Titan) Reverse(name string, params ...interface{}) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0

	for _, r := range t.router.routes {
		if r.Name == name {
			for i, l := 0, len(r.Path); i < l; i++ {
				if r.Path[i] == ':' && n < ln {
					for ; i < l && r.Path[i] != '/'; i++ {
					}
					uri.WriteString(fmt.Sprintf("%v", params[n]))
					n++
				}

				if i < l {
					uri.WriteByte(r.Path[i])
				}
			}

			break
		}
	}

	return uri.String()
}

// Routes returns the registered routes.
func (t *Titan) Routes() []*Route {
	routes := make([]*Route, 0, len(t.router.routes))
	for _, v := range t.router.routes {
		routes = append(routes, v)
	}

	return routes
}

// ServeGemini serves Gemini request.
func (t *Titan) ServeGemini(c Context) {
	if c.Titan() != t {
		// Acquire context from correct Gig and use it instead.
		orig := c.(*context)

		ctx := t.pool.Get().(*context)
		defer t.pool.Put(ctx)

		ctx.reset(orig.conn, orig.u, orig.requestURI, orig.TLS)

		c = ctx
	}

	var h HandlerFunc

	URL := c.URL()

	if t.premiddleware == nil {
		t.router.find(getPath(URL), c)
		h = c.Handler()
		h = applyMiddleware(h, t.middleware...)
	} else {
		h = func(c Context) error {
			t.router.find(getPath(URL), c)
			h := c.Handler()
			h = applyMiddleware(h, t.middleware...)
			return h(c)
		}
		h = applyMiddleware(h, t.premiddleware...)
	}

	// Execute chain
	if err := h(c); err != nil {
		t.GeminiErrorHandler(err, c)
	}
}

func (t *Titan) Run(args ...interface{}) {
	var (
		cert, key []byte
		certFile, keyFile interface{}
		addr string
	)

	switch len(args) {
	case 2:
		addr, certFile, keyFile = "", args[0], args[1]
		if addr == "" {
			addr = ":1965" // TODO
		} else {
			addr = ":" + addr
		}
	case 3:
		addr, certFile, keyFile = args[0].(string), args[1], args[2]
	default:
		panic("must specify 2 or 3 arguments to Run")
	}

	if cert, err = filepathOrContent(certFile); err != nil {
		return
	}

	if key, err = filepathOrContent(keyFile); err != nil {
		return
	}

	t.TLSConfig.Certificates = make([]tls.Certificate, 1)

	if t.TLSConfig.Certificates[0], err = tls.X509KeyPair(cert, key); err != nil {
		return
	}

	return t.startTLS(addr)
}

func filepathOrContent(fileOrContent interface{}) (content []byte, err error) {
	switch v := fileOrContent.(type) {
	case string:
		return ioutil.ReadFile(v)
	case []byte:
		return v, nil
	default:
		return nil, ErrInvalidCertOrKeyType
	}
}

func (t *Titan) startTLS(address string) error {
	t.addr = address

	t.mu.Lock()
	if t.listener == nil {
		l, err := newListener(t.addr)
		if err != nil {
			return err
		}

		t.listener = tls.NewListener(l, t.TLSConfig)
	}
	t.mu.Unlock()

	defer t.listener.Close()

	debugPrintf("⇨ titan server started on %s\n", t.listener.Addr())

	return t.serve()
}

func (t *Titan) serve() error {
	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.doneChan:
				return ErrServerClosed
			default:
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				debugPrintf("titan: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)

				continue
			}

			return err
		}

		tc, ok := conn.(*tls.Conn)
		if !ok {
			debugPrintf("titan: non-tls connection")
			continue
		}

		go t.handleRequest(tc)
	}
}


func (t *Titan) handleRequest(conn *tls.Conn) {
	defer conn.Close()

	if d := t.ReadTimeout; d != 0 {
		err := conn.SetReadDeadline(time.Now().Add(d))
		if err != nil {
			debugPrintf("titan: could not set socket read timeout: %s", err)
		}
	}

	reader := bufio.NewReaderSize(conn, 1024)
	request, overflow, err := reader.ReadLine()

	if overflow {
		debugPrintf("titan: request overflow")

		_, _ = conn.Write([]byte(fmt.Sprintf("%d %s\r\n", StatusBadRequest, "Request too long!")))

		return
	} else if err != nil {
		if err == io.EOF {
			debugPrintf("titan: EOF reading from client, read %d bytes", len(request))
			return
		}
		debugPrintf("titan: unknown error reading request header: %s", err)

		_, _ = conn.Write([]byte(fmt.Sprintf("%d %s\r\n", StatusBadRequest, "Unknown error reading request!")))

		return
	}

	header := string(request)
	URL, err := url.Parse(header)

	if err != nil {
		debugPrintf("titan: invalid request url: %s", err)

		_, _ = conn.Write([]byte(fmt.Sprintf("%d %s\r\n", StatusBadRequest, "Error parsing URL!")))

		return
	}

	if URL.Scheme == "" {
		URL.Scheme = "titan"
	}

	if URL.Scheme != "titan" {
		debugPrintf("titan: non-titan scheme: %s", header)

		_, _ = conn.Write([]byte(fmt.Sprintf("%d %s\r\n", StatusBadRequest, "No proxying to non-Gemini content!")))

		return
	}

	if d := t.WriteTimeout; d != 0 {
		err := conn.SetWriteDeadline(time.Now().Add(d))
		if err != nil {
			debugPrintf("titan: could not set socket write timeout: %s", err)
		}
	}

	tlsState := new(tls.ConnectionState)
	*tlsState = conn.ConnectionState()

	// Acquire context
	c := t.pool.Get().(*context)
	t.reset(conn, URL, header, tlsState)

	t.ServeGemini(c) // TODO

	// Release context
	t.pool.Put(c)
}

// Close immediately stops the server.
// It internally calls `net.Listener#Close()`.
func (t *Titan) Close() error {
	t.closeOnce.Do(func() {
		close(t.doneChan)
	})
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.listener != nil {
		return t.listener.Close()
	}

	return nil
}

// NewError creates a new TitanError instance.
func NewError(code Status, message string) *TitanError {
	return &TitanError{Code: code, Message: message}
}

// NewErrorFrom creates a new TitanError instance using Code from existing TitanError.
func NewErrorFrom(err *TitanError, message string) *TitanError {
	return &TitanError{Code: err.Code, Message: message}
}

// Error makes it compatible with `error` interface.
func (ge *TitanError) Error() string {
	return fmt.Sprintf("error=%s", ge.Message)
}

// getPath returns RawPath, if it's empty returns Path from URL.
func getPath(u *url.URL) string {
	path := u.RawPath
	if path == "" {
		path = u.Path
	}

	return path
}

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}

	return t.String()
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by Run so dead TCP connections (e.g.
// closing laptop mid-download) eventually go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	if c, err = ln.AcceptTCP(); err != nil {
		return
	} else if err = c.(*net.TCPConn).SetKeepAlive(true); err != nil {
		return
	}
	// Ignore error from setting the KeepAlivePeriod as some systems, such as
	// OpenBSD, do not support setting TCP_USER_TIMEOUT on IPPROTO_TCP
	_ = c.(*net.TCPConn).SetKeepAlivePeriod(3 * time.Minute)

	return
}

func newListener(address string) (*tcpKeepAliveListener, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	return &tcpKeepAliveListener{l.(*net.TCPListener)}, nil
}

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	return h
}
