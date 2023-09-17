package coco

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
// The body can be a string, a []byte, or any other value.
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

// Set sets the specified value to the HTTP response header field.
// If the header is not already set, it creates the header with the specified value.
func (r *Response) Set(key string, val ...string) {
	if len(val) == 0 {
		return
	}

	key = strings.ToLower(key)

	if key == "content-type" && !strings.Contains(strings.ToLower(val[0]), "charset") {
		if strings.HasPrefix(val[0], "text/") || strings.Contains(val[0], "application/json") {
			val[0] += "; charset=utf-8"
		}

	}

	if len(val) > 1 {
		r.w.Header().Set(key, strings.Join(val, ", "))
	} else {
		r.w.Header().Set(key, val[0])
	}
}

// SendStatus sends the HTTP response status code.
func (r *Response) SendStatus(statusCode int) *Response {
	body := http.StatusText(statusCode)
	if body == "" {
		body = strconv.Itoa(statusCode)
	}

	r.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	r.status = statusCode
	r.w.WriteHeader(statusCode)
	r.w.Write([]byte(body))

	return r
}

// Status sets the HTTP status for the response.
func (r *Response) Status(code int) *Response {
	if r.written() {
		return r
	}
	r.status = code
	r.w.WriteHeader(code)
	return r
}

// Get returns the HTTP response header specified by field.
func (r *Response) Get(key string) string {
	return r.w.Header().Get(key)
}

// Location sets the response Location HTTP header to the specified path parameter.
// If the provided path is relative, it is instead resolved relative to the path of the current request.
func (r *Response) Location(path string) string {
	var location string

	if path == "back" {
		location = r.ctx.request().Referer()
	}

	if location == "" {
		location = path
	}

	parsedURL, err := url.Parse(location)
	if err != nil {
		// Log the error, replace with your logger
		log.Println("Invalid URL: ", err)
		location = "/"
	}
	r.w.Header().Set("Location", parsedURL.String())

	return location
}

// Redirect redirects to the URL derived from the specified path, with specified status.
// If status is not specified, status defaults to '302 Found'.
func (r *Response) Redirect(path string, status ...int) {
	statusCode := http.StatusFound // Default to 302
	if len(status) > 0 {
		statusCode = status[0]
	}

	redirectURL := r.Location(path)
	http.Redirect(r.w, r.ctx.request(), redirectURL, statusCode)
}

// Type sets the Content-Type HTTP header to the MIME type as determined by the filename’s extension.
func (r *Response) Type(filename string) {
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	r.w.Header().Set("Content-Type", mimeType)
}

// VaryFieldNameRegex checks for valid field names for the Vary header as defined by RFC 7231.
var VaryFieldNameRegex = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)

// Vary add field to Vary header, if it doesn't already exist.
func (r *Response) Vary(field string) {

	if field == "" {
		log.Println("field argument is required")
		return
	}

	existingHeader := r.w.Header().Get("Vary")

	fields := strings.Split(field, ",")
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}

	existingFields := strings.Split(existingHeader, ",")
	for i, field := range existingFields {
		existingFields[i] = strings.TrimSpace(field)
	}

	for _, f := range fields {
		if !VaryFieldNameRegex.MatchString(f) {
			log.Println("field argument contains an invalid header name")
			return
		}
	}

	if existingHeader == "*" || strings.Contains(field, "*") {
		r.w.Header().Set("Vary", "*")
		return
	}

	// Utilizing a map to efficiently check if a field already exists
	fieldMap := make(map[string]bool)
	for _, f := range fields {
		fieldMap[strings.ToLower(f)] = true
	}

	for _, ef := range existingFields {
		if !fieldMap[strings.ToLower(ef)] {
			fields = append(fields, ef)
		}
	}

	r.w.Header().Set("Vary", strings.Join(fields, ", "))
}

// Render renders a template with data and sends a text/html response.
// name is the name of the template to render.
func (r *Response) Render(name string, data interface{}) error {

	temp := r.ctx.templates[name]
	if temp == nil {
		return fmt.Errorf("template %s not found", name)
	}
	return temp.Execute(r.w, data)
}
