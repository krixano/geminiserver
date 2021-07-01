package titan

import (
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type (
	// Context represents the context of the current request. It holds connection
	// reference, path, path parameters, data and registered handler.
	// DO NOT retain Context instance, as it will be reused by other connections.
	Context interface {
		// Response returns `*Response`.
		Response() *Response

		// IP returns the client's network address.
		IP() string

		// Certificate returns client's leaf certificate or nil if none provided
		Certificate() *x509.Certificate

		// CertHash returns a hash of client's leaf certificate or empty string is none
		CertHash() string

		// URL returns the URL for the context.
		URL() *url.URL

		// Path returns the registered path for the handler.
		Path() string

		// QueryString returns unescaped URL query string or error if the raw query
		// could not be unescaped. Use Context#URL().RawQuery to get raw query string.
		QueryString() (string, error)

		// RequestURI is the unmodified URL string as sent by the client
		// to a server. Usually the URL() or Path() should be used instead.
		RequestURI() string

		// Param returns path parameter by name.
		Param(name string) string

		// Get retrieves data from the context.
		Get(key string) interface{}

		// Set saves data in the context.
		Set(key string, val interface{})

		// Render renders a template with data and sends a text/gemini response with status
		// code Success. Renderer must be registered using `Titan.Renderer`.
		Render(name string, data interface{}) error

		// Titan sends a text/gemini response with status code Success.
		Titan(text string, args ...interface{}) error

		// NoContent sends a response with no body, and a status code and meta field.
		// Use for any non-2x status codes
		NoContent(code Status, meta string, values ...interface{}) error

		// Error invokes the registered error handler. Generally used by middleware.
		Error(err error)

		// Handler returns the matched handler by router.
		Handler() HandlerFunc

		// Titan returns the `Titan` instance.
		Titan() *Titan
	}

	context struct {
		conn       net.Conn
		TLS        *tls.ConnectionState
		u          *url.URL
		response   *Response
		path       string
		requestURI string
		pnames     []string
		pvalues    []string
		handler    HandlerFunc
		store      storeMap
		titan        *Titan
		lock       sync.RWMutex
	}
)

const (
	indexPage = "index.gmi"
)

func (c *context) Response() *Response {
	return c.response
}

func (c *context) IP() string {
	ra, _, _ := net.SplitHostPort(c.conn.RemoteAddr().String())
	return ra
}

func (c *context) Certificate() *x509.Certificate {
	if c.TLS == nil || len(c.TLS.PeerCertificates) == 0 {
		return nil
	}

	return c.TLS.PeerCertificates[0]
}

func (c *context) CertHash() string {
	cert := c.Certificate()
	if cert == nil {
		return ""
	}

	return fmt.Sprintf("%x", md5.Sum(cert.Raw))
}

func (c *context) URL() *url.URL {
	return c.u
}

func (c *context) Path() string {
	return c.path
}

func (c *context) RequestURI() string {
	return c.requestURI
}

func (c *context) Param(name string) string {
	for i, n := range c.pnames {
		if i < len(c.pvalues) {
			if n == name {
				return c.pvalues[i]
			}
		}
	}

	return ""
}

func (c *context) QueryString() (string, error) {
	return url.QueryUnescape(c.u.RawQuery)
}

func (c *context) Get(key string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.store[key]
}

func (c *context) Set(key string, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.store == nil {
		c.store = make(storeMap)
	}

	c.store[key] = val
}

func (c *context) Render(name string, data interface{}) (err error) {
	if c.titan.Renderer == nil {
		return ErrRendererNotRegistered
	}

	if err = c.response.WriteHeader(StatusSuccess, MIMETextTitan); err != nil {
		return
	}

	return c.titan.Renderer.Render(c.response, name, data, c)
}

func (c *context) Titan(format string, values ...interface{}) error {
	return c.TitanBlob([]byte(fmt.Sprintf(format, values...)))
}


func (c *context) TitanBlob(b []byte) (err error) {
	return c.Blob(MIMETextTitan, b)
}


func (c *context) Text(format string, values ...interface{}) (err error) {
	return c.Blob(MIMETextPlain, []byte(fmt.Sprintf(format, values...)))
}

func (c *context) Blob(contentType string, b []byte) (err error) {
	err = c.response.WriteHeader(StatusSuccess, contentType)
	if err != nil {
		return
	}

	_, err = c.response.Write(b)

	return
}

/*
func (c *context) Stream(contentType string, r io.Reader) (err error) {
	err = c.response.WriteHeader(StatusSuccess, contentType)
	if err != nil {
		return
	}

	_, err = io.Copy(c.response, r)

	return
}

func (c *context) File(file string) (err error) {
	if containsDotDot(file) {
		c.Error(ErrBadRequest)
		return
	}

	s, err := os.Stat(file)
	if err != nil {
		c.Error(ErrNotFound)
		return
	}

	if uint64(s.Mode().Perm())&0444 != 0444 {
		c.Error(ErrGone)
		return
	}

	if s.IsDir() {
		files, err := ioutil.ReadDir(file)
		if err != nil {
			c.Error(ErrTemporaryFailure)
			return err
		}

		for _, f := range files {
			if f.Name() == indexPage {
				return c.File(path.Join(file, indexPage))
			}
		}

		err = c.response.WriteHeader(StatusSuccess, "text/gemini")

		if err != nil {
			return err
		}

		_, _ = c.response.Write([]byte(fmt.Sprintf("# Listing %s\n\n", c.u.Path)))

		sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

		for _, file := range files {
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}

			if uint64(file.Mode().Perm())&0444 != 0444 {
				continue
			}

			_, _ = c.response.Write([]byte(fmt.Sprintf("=> %s %s [ %v ]\n", filepath.Clean(path.Join(c.u.Path, file.Name())), file.Name(), bytefmt(file.Size()))))
		}

		return nil
	}

	ext := filepath.Ext(file)

	var mimeType string
	if ext == ".gmi" {
		mimeType = "text/gemini"
	} else {
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "octet/stream"
		}
	}

	f, err := os.OpenFile(file, os.O_RDONLY, 0600)
	if err != nil {
		c.Error(ErrTemporaryFailure)
		return
	}
	defer f.Close()

	err = c.response.WriteHeader(StatusSuccess, mimeType)

	if err != nil {
		return
	}

	_, err = io.Copy(c.response, f)

	if err != nil {
		// .. remote closed the connection, nothing we can do besides log
		// or io error, but status is already sent, everything is broken!
		c.Error(ErrTemporaryFailure)
	}

	return
}
*/

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}

	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}

	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

func (c *context) NoContent(code Status, meta string, values ...interface{}) error {
	return c.response.WriteHeader(code, fmt.Sprintf(meta, values...))
}

func (c *context) Error(err error) {
	c.titan.TitanErrorHandler(err, c)
}

func (c *context) Titan() *Titan {
	return c.titan
}

func (c *context) Handler() HandlerFunc {
	return c.handler
}

func (c *context) reset(conn net.Conn, u *url.URL, requestURI string, tls *tls.ConnectionState) {
	c.conn = conn
	c.TLS = tls
	c.u = u
	c.requestURI = requestURI
	c.response.reset(conn)
	c.handler = NotFoundHandler
	c.store = nil
	c.path = ""
	c.pnames = nil
	// NOTE: Don't reset because it has to have length c.titan.maxParam at all times
	for i := 0; i < *c.titan.maxParam; i++ {
		c.pvalues[i] = ""
	}
}

func bytefmt(b int64) string {
	const unit = 1000

	if b < unit {
		return fmt.Sprintf("%dB", b)
	}

	div, exp := int64(unit), 0

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}
