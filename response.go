package coco

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Response struct {
	w      http.ResponseWriter
	ctx    *reqcontext
	status int
}

func (r *Response) written() bool {
	return r.status != 0
}

// Append sets the specified value to the HTTP response header field.
// If the header is not already set, it creates the header with the specified value.
func (r *Response) Append(key, value string) *Response {
	r.w.Header().Add(key, value)
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
		r.w.Header().Set("Content-Type", mimeType)
	}

	r.w.Header().Set("Content-Disposition", contentDisposition)
	return r
}

// Cookie sets cookie name to value
func (r *Response) Cookie(cookie *http.Cookie) *Response {
	http.SetCookie(r.w, cookie)
	return r
}

// SignedCookie SecureCookie sets a signed cookie
func (r *Response) SignedCookie(cookie *http.Cookie, secret string) *Response {

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(cookie.Value))
	mac.Write([]byte(cookie.Name))

	sig := string(mac.Sum(nil))
	cookie.Value = fmt.Sprintf("%s.%s", sig, cookie.Value)

	http.SetCookie(r.w, cookie)
	return r
}

// ClearCookie clears the cookie by setting the MaxAge to -1
func (r *Response) ClearCookie(cookie *http.Cookie) *Response {
	cookie.MaxAge = -1
	return r.Cookie(cookie)
}

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
	errDirPath      = errors.New("specified path is a directory")
	errDotfilesDeny = errors.New("serving dotfiles is not allowed")
)

func (r *Response) Download(filepath, filename string, options *DownloadOption, cb func(error)) {
	if options == nil {
		options = defaultDownloadOptions
	}

	// check dotfiles option and update the filename accordingly
	if options.Dotfiles == "deny" && strings.HasPrefix(filename, ".") {
		cb(errDotfilesDeny)
		return
	}

	r.SendFile(filepath, filename, options, cb)
}

// SendFile transfers the file at the given path.
// Sets the Content-Type response HTTP header field based on the filename’s extension.
func (r *Response) SendFile(filepath, filename string, options *DownloadOption, cb func(error)) {
	if cb == nil {
		cb = func(error) {
		}
	}

	fi, err := os.Stat(filepath)
	if err != nil {
		cb(err)
		return
	}

	if fi.IsDir() {
		cb(errDirPath)
		return
	}

	file, err := os.Open(filepath)
	if err != nil {
		cb(err)
		return
	}
	defer file.Close()

	// Construct Cache-Control header
	cacheControlValues := make([]string, 0)
	if options.CacheControl {
		if options.MaxAge > 0 {
			cacheControlValues = append(cacheControlValues, fmt.Sprintf("public, max-age=%d", options.MaxAge))
		} else {
			cacheControlValues = append(cacheControlValues, "public, max-age=0")
		}
		if options.Immutable {
			cacheControlValues = append(cacheControlValues, "immutable")
		}
		r.w.Header().Set("Cache-Control", strings.Join(cacheControlValues, ", "))
	}

	if options.LastModified {
		r.w.Header().Set("Last-Modified", fi.ModTime().Format(time.RFC1123))
	}
	if options.AcceptRanges {
		r.w.Header().Set("Accept-Ranges", "bytes")
	}
	for key, value := range options.Headers {
		r.w.Header().Set(key, value)
	}

	r.w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	http.ServeContent(r.w, r.ctx.request(), filename, fi.ModTime(), file)

	cb(nil)

}

// JSON sends a JSON response with the given payload.
func (r *Response) JSON(v interface{}) *Response {

	r.setContentType("application/json")

	jsn, err := json.Marshal(v)
	if err != nil {
		http.Error(r.w, err.Error(), http.StatusInternalServerError)
		return r
	}

	return r.Send(jsn)
}

func (r *Response) setContentType(contentType string) {
	if cty := r.w.Header().Get("Content-Type"); cty == "" {
		r.w.Header().Set("Content-Type", contentType)
	}
}

// Send sends the HTTP response.
// The body parameter can be a string, a []byte, or any other value.
func (r *Response) Send(body interface{}) *Response {
	r.setContentType("application/octet-stream")

	var data []byte
	var err error

	switch chunk := body.(type) {
	case string:
		r.setContentType("text/html")
		data = []byte(chunk)
	case []byte:
		data = chunk
	default:
		r.setContentType("application/json")
		data, err = json.Marshal(chunk)
		if err != nil {
			http.Error(r.w, err.Error(), http.StatusInternalServerError)
			return r
		}
	}

	if _, err := r.w.Write(data); err != nil {
		http.Error(r.w, err.Error(), http.StatusInternalServerError)
		return r
	}

	r.status = http.StatusOK
	return r
}

// TODO: implement this method
func (r *Response) Set(key string, val ...string) {

}

func (r *Response) Get(key string) string {
	return r.w.Header().Get(key)
}

func (r *Response) Location(path string) {
	var location string

	if path == "back" {
		location = r.ctx.request().Referer()
	}

	if location == "" {
		location = "/"
	}

	r.w.Header().Set("Location", url.QueryEscape(location))
}

func (r *Response) Redirect(path string, status ...int) {
	// if len(status) > 0 {
	// 	r.WriteHeader(status[0])
	// }
	http.Redirect(r.w, r.ctx.request(), path, http.StatusFound)
}

func (r *Response) SendStatus(statusCode int) {
	r.status = statusCode
}

func (r *Response) Type(filename string) {
	r.w.Header().Set("Content-Type", filepath.Ext(filename))
}

func (r *Response) Vary(field string) {
	r.w.Header().Add("Vary", field)
}

// Status sets the HTTP status for the response.
func (r *Response) Status(code int) *Response {
	if r.written() {
		return r
	}
	r.status = code
	return r
}

func (r *Response) Render(name string, data interface{}) error {

	temp := r.ctx.templates[name]
	if temp == nil {
		return fmt.Errorf("template %s not found", name)
	}
	return temp.Execute(r.w, data)
}
