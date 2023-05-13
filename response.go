package coco

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

	if filename != "" {
		r.w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		ext := filepath.Ext(filename)
		r.w.Header().Set("Content-Type", ext)
	} else {
		r.w.Header().Set("Content-Disposition", "attachment")
	}
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

	// MaxAge sets the max-age property of the Cache-Control header in milliseconds.
	MaxAge int

	// LastModified sets the Last-Modified header to the file modification time.
	LastModified bool

	// Headers containing HTTP headers to serve with the file.
	// The header Content-Disposition will be overriden by the filename argument
	Headers map[string]string

	// Dotfiles enables  serving of dotfiles. Possible values are “allow”, “deny”, “ignore”
	Dotfiles bool

	// AcceptRanges enable or disable accepting ranged requests
	AcceptRanges bool

	// CacheControl enable or disable setting Cache-Control response header
	CacheControl bool

	// Immutable enable or disable the immutable directive in the Cache-Control response header.
	// If enabled, the maxAge option should also be specified to enable caching.
	// The immutable directive will prevent supported clients from making conditional
	// requests during the life of the maxAge option to check if the file has changed
	Immutable bool
}

func (r *Response) Download(filepath, filename string, options *DownloadOption, cb func(error)) {

	// TODO: improve this method to accept headers and other options
	fi, err := os.Stat(filename)
	if err == nil && !fi.IsDir() {
		if file, err := os.Open(filename); err == nil {
			defer file.Close()
			http.ServeContent(r.w, r.ctx.request(), filename, fi.ModTime(), file)
			return
		}
	}
	// fi, err := os.Stat(filename)
	// if err == nil && !fi.IsDir() {
	// 	if file, err := os.Open(filename); err == nil {
	// 		defer file.Close()
	// 		http.ServeContent(r.w, r.ctx.Request(), filename, fi.ModTime(), file)
	// 		return
	// 	}
	// }

	cb(err)
}

// JSON sets the Content-Type as “application/json” and sends a JSON response.
func (r *Response) JSON(v interface{}) *Response {
	if cty := r.w.Header().Get("Content-Type"); cty == "" {
		r.w.Header().Set("Content-Type", "application/json")
	}

	jsn, _ := json.Marshal(v)
	fmt.Printf("Type of jsn: %T\n", jsn)
	return r.Send(jsn)
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

func (r *Response) Send(body interface{}) *Response {

	cType := r.w.Header().Get("Content-Type")
	switch chunk := body.(type) {
	case string:
		if cType == "" {
			r.w.Header().Set("Content-Type", "text/html")
		}
		r.w.Write([]byte(chunk))
	case []uint8:
		if cType == "" {
			r.w.Header().Set("Content-Type", "application/octet-stream")
		}
		r.w.Write(chunk)
	default:
		if cType == "" {
			r.w.Header().Set("Content-Type", "application/json")
		}

	}
	if !r.written() {
		r.status = http.StatusOK
	}
	return r
}

func (r *Response) SendFile(filename string, cb func(error)) {
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
