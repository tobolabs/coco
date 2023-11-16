package coco

import (
	coreContext "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-http-utils/fresh"
	"github.com/julienschmidt/httprouter"
)

// Error is an error type that is returned when an error occurs while handling a request.
type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

// Range represents a byte range.
type Range struct {
	Start int64
	End   int64
}

type Request struct {
	r *http.Request

	BaseURL string

	// HostName contains the hostname derived from the Host HTTP header.
	HostName string

	// Ip contains the remote IP address of the request.
	Ip string

	// Ips contains the remote IP addresses from the X-Forwarded-For header.
	Ips []string

	// Protocol contains the request protocol string: "http" or "https"
	Protocol string

	// Secure is a boolean that is true if the request protocol is "https"
	Secure bool

	// Subdomains is a slice of subdomain strings.
	Subdomains []string

	// Xhr is a boolean that is true if the request's X-Requested-With header
	// field is "XMLHttpRequest".
	Xhr bool

	// OriginalURL is the original URL requested by the client.
	OriginalURL *url.URL

	// Cookies contains the cookies sent by the request.
	Cookies map[string]string

	// Body contains the body of the request.
	Body

	// Query contains the parsed query string from the URL.
	Query map[string]string

	// Params contains the Route parameters.
	Params map[string]string

	// SignedCookies contains the signed cookies sent by the request.
	SignedCookies map[string]string

	// Stale is a boolean that is true if the request is stale, false otherwise.
	Stale bool

	// Fresh is a boolean that is true if the request is fresh, false otherwise.
	Fresh bool

	// Method contains a string corresponding to the HTTP method of the request:
	// GET, POST, PUT, and so on.
	Method string

	// Path contains a string corresponding to the path of the request.
	Path string
}

type Body struct {
	req *http.Request
}

func newRequest(r *http.Request, w http.ResponseWriter, params httprouter.Params, app *App) (*Request, error) {
	hostName, err := parseHostName(r.Host)
	if err != nil {
		return nil, err
	}

	ip, err := parseIP(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	xhr := isXhr(r.Header.Get("X-Requested-With"))

	domainOffset := app.settings["subdomain offset"].(int)

	req := &Request{
		BaseURL:     filepath.Dir(r.URL.Path),
		HostName:    hostName,
		Ip:          ip,
		Ips:         []string{},
		Protocol:    r.Proto,
		Secure:      r.TLS != nil,
		Xhr:         xhr,
		OriginalURL: r.URL,
		Cookies:     parseCookies(r.Cookies()),
		Query:       parseQuery(r.URL.Query()),
		Params:      parseParams(params),
		Method:      r.Method,
		Body:        Body{r},
		r:           r,
		Path:        r.URL.Path,
		Stale:       !checkFreshness(r, w),
		Fresh:       checkFreshness(r, w),
		Subdomains:  parseSubdomains(hostName, domainOffset),
	}

	if app.IsTrustProxyEnabled() {
		req.Ips = parseXForwardedFor(r.Header.Get("X-Forwarded-For"))
	}

	return req, nil
}

// parseSubdomains parses the subdomains of the request hostname based on a subdomain offset.
func parseSubdomains(host string, subdomainOffset int) []string {
	parts := strings.Split(host, ".")

	if len(parts) <= subdomainOffset {
		return nil
	}

	return parts[:len(parts)-subdomainOffset]
}

// parseXForwardedFor parses the X-Forwarded-For header to extract IP addresses
func parseXForwardedFor(header string) []string {
	parts := strings.Split(header, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func parseHostName(host string) (string, error) {
	if idx := strings.Index(host, ":"); idx != -1 {
		return host[:idx], nil
	}
	return host, nil
}

func parseIP(remoteAddr string) (string, error) {
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx], nil
	}
	return remoteAddr, nil
}

func isXhr(xRequestedWith string) bool {
	return strings.EqualFold(xRequestedWith, "XMLHttpRequest")
}

func parseCookies(cookies []*http.Cookie) map[string]string {
	cookieMap := make(map[string]string)
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}
	return cookieMap
}

func parseQuery(query url.Values) map[string]string {
	queryMap := make(map[string]string)
	for key, values := range query {
		queryMap[key] = values[0] // Assuming there's at least one value
	}
	return queryMap
}

func parseParams(params httprouter.Params) map[string]string {
	paramMap := make(map[string]string)
	for _, param := range params {
		paramMap[param.Key] = param.Value
	}
	return paramMap
}

func checkFreshness(req *http.Request, w http.ResponseWriter) bool {
	if req.Method != "GET" && req.Method != "HEAD" {
		return false
	}
	return fresh.IsFresh(req.Header, w.Header())
}

// JSONError is an error type that is returned when the request body is not a valid JSON.
type JSONError struct {
	Status  int
	Message string
}

func (e JSONError) Error() string {
	return fmt.Sprintf("JSON error (Status %d): %s", e.Status, e.Message)
}

// JSON marshals the request body into the given interface.
// It returns an error if the request body is not a valid JSON or if the
// given interface is not a pointer.
func (body *Body) JSON(dest interface{}) error {
	if dest == nil {
		return JSONError{http.StatusBadRequest, "Destination interface is nil"}
	}

	contentType := body.req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return JSONError{http.StatusUnsupportedMediaType, "Unsupported media type, expected 'application/json'"}
	}

	bdy, err := io.ReadAll(body.req.Body)
	if err != nil {
		return JSONError{http.StatusBadRequest, "Error reading JSON payload: " + err.Error()}
	}

	if err := body.req.Body.Close(); err != nil {
		return JSONError{http.StatusInternalServerError, "Error closing request body: " + err.Error()}
	}

	if err := json.Unmarshal(bdy, dest); err != nil {
		return JSONError{http.StatusBadRequest, "Error unmarshalling JSON: " + err.Error()}
	}

	return nil
}

// Text returns the request body as a string.
func (body *Body) Text() (string, error) {
	b, err := io.ReadAll(body.req.Body)
	if err != nil {
		return "", fmt.Errorf("error reading text payload: %w", err)
	}

	if err := body.req.Body.Close(); err != nil {
		return "", fmt.Errorf("error closing request body: %w", err)
	}

	return string(b), nil
}

// FormData returns the body form data, expects request sent with `x-www-form-urlencoded` header or
func (body *Body) FormData() (map[string][]string, error) {
	if body.req.Body == nil {
		return nil, errors.New("request body is nil")
	}
	defer body.req.Body.Close()

	contentType, _, err := mime.ParseMediaType(body.req.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Content-Type header: %w", err)
	}

	if contentType != "application/x-www-form-urlencoded" {
		return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
	}

	if err := body.req.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form data: %w", err)
	}

	data := make(map[string][]string)
	for key, values := range body.req.Form {
		data[key] = make([]string, len(values))
		copy(data[key], values)
	}

	return data, nil
}

func (a *App) IsTrustProxyEnabled() bool {
	return a.settings["trust proxy"].(bool)
}

func (req *Request) Cookie(name string) (value string, exists bool) {
	value, exists = req.Cookies[name]
	return value, exists
}

// GetParam returns the value of param `name` when present or `defaultValue`.
func (req *Request) GetParam(name string) string {
	if value, ok := req.Params[name]; ok {
		return value
	}
	return ""
}

// Get returns the value of specified HTTP request header field (case-insensitive match)
func (req *Request) Get(key string) string {
	if value := req.r.Header.Get(key); value != "" {
		return value
	}
	return ""
}

// Set sets the value of specified HTTP request header field (case-insensitive match)
func (req *Request) Set(key string, value string) {
	req.r.Header.Set(key, value)
}

// QueryParam returns the value of query parameter `name` when present or `defaultValue`.
func (req *Request) QueryParam(name string) string {
	if value, ok := req.Query[name]; ok {
		return value
	}
	return ""
}

// Is returns true if the incoming request’s “Content-Type” HTTP header field
// matches the given mime type.
func (req *Request) Is(mime string) bool {
	contentType := req.r.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	switch mime {
	case "json":
		mime = "application/json"
	case "html":
		mime = "text/html"
	case "xml":
		mime = "application/xml"
	case "text":
		mime = "text/plain"
	}

	mimeParts := strings.Split(mime, "/")
	ctParts := strings.Split(contentType, "/")

	if mimeParts[1] == "*" {
		return strings.EqualFold(mimeParts[0], ctParts[0])
	}

	return strings.EqualFold(mime, contentType)
}

// Range returns the first range found in the request’s “Range” header field.
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Range
func (req *Request) Range(size int64) ([]Range, error) {
	rangeHeader := req.r.Header.Get("Range")
	if rangeHeader == "" {
		return nil, nil // No Range header is not an error, just no range requested
	}

	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("invalid range specifier")
	}

	rangesStr := strings.Split(rangeHeader[6:], ",")
	var ranges []Range
	for _, rStr := range rangesStr {
		rStr = strings.TrimSpace(rStr)
		if rStr == "" {
			continue
		}

		rangeParts := strings.SplitN(rStr, "-", 2)
		if len(rangeParts) != 2 {
			return nil, fmt.Errorf("invalid range format")
		}

		startStr, endStr := strings.TrimSpace(rangeParts[0]), strings.TrimSpace(rangeParts[1])
		var start, end int64
		var err error

		if startStr != "" {
			start, err = strconv.ParseInt(startStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid range start value")
			}
		}

		if endStr != "" {
			end, err = strconv.ParseInt(endStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid range end value")
			}
		}

		if startStr == "" {
			start = size - end
			end = size - 1
		} else if endStr == "" {
			end = size - 1
		}

		if start > end || start < 0 || end >= size {
			continue
		}

		ranges = append(ranges, Range{Start: start, End: end})
	}

	if len(ranges) == 0 {
		return nil, fmt.Errorf("no satisfiable range found")
	}

	return ranges, nil
}

func (req *Request) Context() coreContext.Context {
	return req.r.Context()
}

func (req *Request) SetContext(ctx coreContext.Context) {
	req.r = req.r.WithContext(ctx)
}

//// Accepts checks if the specified mine types are acceptable, based on the request’s Accept HTTP header field.
//// The method returns the best match, or if none of the specified mine types is acceptable, returns "".
//func (req *Request) Accepts(mime ...string) string {
//
//	if len(mime) == 0 {
//		return ""
//	}
//
//	return ""
//}

//// AcceptsCharsets returns true if the incoming request’s “Accept-Charset” HTTP header field
//// includes the given charset.
//func (req *Request) AcceptsCharsets(charset string) bool {
//	return false
//}
//
//// AcceptsEncodings returns true if the incoming request’s “Accept-Encoding” HTTP header field
//// includes the given encoding.
//func (req *Request) AcceptsEncodings(encoding string) bool {
//	return false
//}

//// AcceptsLanguages returns true if the incoming request’s “Accept-Language” HTTP header field
//// includes the given language.
//func (req *Request) AcceptsLanguages(lang string) bool {
//	return false
//}
