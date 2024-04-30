package coco

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DownloadOption struct {
	MaxAge       int
	LastModified bool
	Headers      map[string]string
	Dotfiles     string
	AcceptRanges bool
	CacheControl bool
	Immutable    bool
}

var (
	defaultDownloadOptions = &DownloadOption{
		MaxAge:       0,
		LastModified: false,
		Headers:      nil,
		Dotfiles:     "ignore",
		AcceptRanges: true,
		CacheControl: true,
		Immutable:    false,
	}
	ErrDirPath      = errors.New("specified path is a directory")
	ErrDotfilesDeny = errors.New("serving dotfiles is not allowed")
)

// func (r *Response) setContentType(contentType string) {
// 	if cty := r.ww.Header().Get("Content-Type"); cty == "" {
// 		r.ww.Header().Set("Content-Type", contentType)
// 	}
// }

type wrappedWriter struct {
	http.ResponseWriter
	statusCode        int
	statusCodeWritten bool
	hijacker          http.Hijacker
	flusher           http.Flusher
}

func wrapWriter(original http.ResponseWriter) *wrappedWriter {
	w := &wrappedWriter{ResponseWriter: original}
	if h, ok := original.(http.Hijacker); ok {
		w.hijacker = h
	}
	if f, ok := original.(http.Flusher); ok {
		w.flusher = f
	}
	return w
}

func (w *wrappedWriter) WriteHeader(code int) {
	if !w.statusCodeWritten {
		w.statusCodeWritten = true
		w.statusCode = code
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
	if !w.statusCodeWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *wrappedWriter) _statusCode() int {
	return w.statusCode
}

func (w *wrappedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {

	if w.hijacker == nil {
		return nil, nil, http.ErrHijacked
	}
	return w.hijacker.Hijack()
}

func (w *wrappedWriter) Flush() {
	if w.flusher == nil {
		return
	}
	w.flusher.Flush()
}

type Response struct {
	ww  *wrappedWriter
	ctx *context
}

// Append sets the specified value to the HTTP response header field.
// If the header is not already set, it creates the header with the specified value.
func (r *Response) Append(key, value string) *Response {
	r.ww.Header().Add(key, value)
	return r
}

// Attachment sets the Content-Disposition header to “attachment”.
// If a filename is given, then the Content-Type header is set based on the filename’s extension.
func (r *Response) Attachment(filename string) *Response {
	contentDisposition := "attachment"

	if filename != "" {
		cleanFilename := filepath.Clean(filename)
		escapedFilename := url.PathEscape(cleanFilename)
		contentDisposition = fmt.Sprintf("attachment; filename=%s", escapedFilename)
		ext := filepath.Ext(cleanFilename)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		r.ww.Header().Set("Content-Type", mimeType)
	}

	r.ww.Header().Set("Content-Disposition", contentDisposition)
	return r
}

// Cookie sets cookie name to value
func (r *Response) Cookie(cookie *http.Cookie) *Response {
	http.SetCookie(r.ww, cookie)
	return r
}

// SignedCookie SecureCookie sets a signed cookie
func (r *Response) SignedCookie(cookie *http.Cookie, secret string) *Response {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(cookie.Value))
	signature := mac.Sum(nil)
	encodedSignature := base64.StdEncoding.EncodeToString(signature)
	cookie.Value = fmt.Sprintf("%s.%s", encodedSignature, cookie.Value)
	http.SetCookie(r.ww, cookie)
	return r
}

// ClearCookie clears the cookie by setting the MaxAge to -1
func (r *Response) ClearCookie(name string) *Response {
	cookie := &http.Cookie{Name: name, MaxAge: -1}
	http.SetCookie(r.ww, cookie)
	return r
}

// Download transfers the file at the given path.
// Sets the Content-Type response HTTP header field based on the filename’s extension.
func (r *Response) Download(filepath, filename string, options *DownloadOption, cb func(error)) {
	if options == nil {
		options = defaultDownloadOptions
	}

	if options.Dotfiles == "deny" && strings.HasPrefix(filename, ".") {
		cb(ErrDotfilesDeny)
		return
	}

	r.SendFile(filepath, filename, options, cb)
}

// SendFile transfers the file at the given path.
// Sets the Content-Type response HTTP header field based on the filename’s extension.
func (r *Response) SendFile(filePath, fileName string, options *DownloadOption, cb func(error)) {
	if cb == nil {
		cb = func(err error) {
			if err != nil {
				fmt.Printf("Error sending file: %v\n", err)
			}
		}
	}

	if options == nil {
		options = defaultDownloadOptions
	}

	fi, err := os.Stat(filePath)
	if err != nil {
		cb(err)
		return
	}

	if fi.IsDir() {
		cb(ErrDirPath)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		cb(err)
		return
	}
	defer file.Close()

	ext := filepath.Ext(fileName)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	r.ww.Header().Set("Content-Type", mimeType)

	if options.CacheControl {
		cacheControlValues := []string{"public"}
		if options.MaxAge > 0 {
			cacheControlValues = append(cacheControlValues, fmt.Sprintf("max-age=%d", options.MaxAge))
		} else {
			cacheControlValues = append(cacheControlValues, "max-age=0")
		}
		if options.Immutable {
			cacheControlValues = append(cacheControlValues, "immutable")
		}
		r.ww.Header().Set("Cache-Control", strings.Join(cacheControlValues, ", "))
	}

	if options.LastModified {
		r.ww.Header().Set("Last-Modified", fi.ModTime().Format(time.RFC1123))
	}
	if options.AcceptRanges {
		r.ww.Header().Set("Accept-Ranges", "bytes")
	}
	for key, value := range options.Headers {
		r.ww.Header().Set(key, value)
	}

	encodedFilename := url.PathEscape(fileName)
	r.ww.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedFilename))

	http.ServeContent(r.ww, r.ctx.request(), fileName, fi.ModTime(), file)

	cb(nil)
}

// JSON sends a JSON response with the given payload.
func (r *Response) JSON(v interface{}) *Response {
	jsn, err := json.Marshal(v)
	if err != nil {
		http.Error(r.ww, err.Error(), http.StatusInternalServerError)
		return r
	}

	r.Set("Content-Type", "application/json; charset=utf-8")
	_, err = r.ww.Write(jsn)
	if err != nil {
		http.Error(r.ww, err.Error(), http.StatusInternalServerError)
	}

	return r
}

// Send sends the HTTP response.
func (r *Response) Send(body interface{}) *Response {
	var data []byte
	var err error

	switch v := body.(type) {
	case string:
		r.Set("Content-Type", "text/plain; charset=utf-8")
		data = []byte(v)
	case []byte:
		data = v
	default:
		return r.JSON(v)
	}

	_, err = r.ww.Write(data)
	if err != nil {
		http.Error(r.ww, err.Error(), http.StatusInternalServerError)
	}

	return r
}

// Set sets the specified value to the HTTP response header field.
func (r *Response) Set(key string, value string) *Response {
	key = http.CanonicalHeaderKey(key)
	if key == "Content-Type" && !strings.Contains(value, "charset") {
		if strings.HasPrefix(value, "text/") || strings.Contains(value, "application/json") {
			value += "; charset=utf-8"
		}
	}
	r.ww.Header().Set(key, value)
	return r
}

// SendStatus sends the HTTP response status code.
func (r *Response) SendStatus(statusCode int) *Response {
	r.Set("Content-Type", "text/plain; charset=utf-8")
	r.ww.WriteHeader(statusCode)
	_, err := r.ww.Write([]byte(http.StatusText(statusCode)))
	if err != nil {
		http.Error(r.ww, err.Error(), http.StatusInternalServerError)
	}
	return r
}

// Status sets the HTTP status for the response.
func (r *Response) Status(code int) *Response {
	r.ww.WriteHeader(code)
	r.ww.WriteHeader(code)
	return r
}

// Get returns the HTTP response header specified by field.
func (r *Response) Get(key string) string {
	return r.ww.Header().Get(http.CanonicalHeaderKey(key))
}

// Location sets the response Location HTTP header to the specified path parameter.
func (r *Response) Location(path string) *Response {
	if path == "back" {
		path = r.ctx.request().Referer()
	}

	if path == "" {
		path = "/"
	}

	r.Set("Location", path)
	return r
}

// Redirect redirects to the URL derived from the specified path, with specified status.
func (r *Response) Redirect(path string, status ...int) *Response {
	statusCode := http.StatusFound
	if len(status) > 0 {
		statusCode = status[0]
	}

	if path == "back" {
		path = r.ctx.request().Referer()
	}
	if path == "" {
		path = "/"
	}

	r.Location(path)
	r.Status(statusCode)
	return r
}

// Type sets the Content-Type HTTP header to the MIME type as determined by the filename’s extension.
func (r *Response) Type(filename string) *Response {
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	r.Set("Content-Type", mimeType)
	return r
}

// Vary adds a field to the Vary header, if it doesn't already exist.
func (r *Response) Vary(field string) *Response {
	if field == "" {
		log.Println("field argument is required")
		return r
	}

	existingHeader := r.Get("Vary")
	if existingHeader == "*" || field == "*" {
		r.Set("Vary", "*")
		return r
	}

	fields := strings.Split(existingHeader, ",")
	for _, f := range fields {
		if strings.TrimSpace(f) == field {
			return r
		}
	}

	if existingHeader != "" {
		field = existingHeader + ", " + field
	}
	r.Set("Vary", field)
	return r
}

// Render renders a template with data and sends a text/html response.
func (r *Response) Render(name string, data interface{}) *Response {
	tmpl, ok := r.ctx.templates[name]
	if !ok {
		http.Error(r.ww, fmt.Sprintf("template %s not found", name), http.StatusInternalServerError)
		return r
	}

	r.Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.Execute(r.ww, data)
	if err != nil {
		http.Error(r.ww, err.Error(), http.StatusInternalServerError)
	}
	return r
}

func (r *Response) StatusCode() int {
	return r.ww._statusCode()
}
