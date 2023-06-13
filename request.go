package coco

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-http-utils/fresh"
	"github.com/golang/gddo/httputil/header"
	"github.com/julienschmidt/httprouter"
)

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

type Request struct {
	r *http.Request

	accepted []string

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
	OriginalURL string

	// Cookies contains the cookies sent by the request.
	Cookies map[string]string

	// Body contains the body of the request.
	Body

	// Query contains the parsed query string from the URL.
	Query map[string]string

	// Params contains the Route parameters.
	Params map[string]string

	// SignedCookies contains the signed cookies sent by the request.
	// see - https://expressjs.com/en/4x/api.html#req.signedCookies
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

	// Url contains the parsed URL of the request.
	Url *url.URL
}

func newRequest(r *http.Request, w http.ResponseWriter, params httprouter.Params) Request {

	isXhr := func() bool {
		xrw := r.Header.Get("X-Requested-With")
		if xrw == "XMLHttpRequest" || xrw == strings.ToLower("XMLHttpRequest") {
			return true
		}
		return false
	}

	req := Request{
		BaseURL:     filepath.Dir(r.URL.Path),
		HostName:    r.Host,
		Ip:          r.RemoteAddr,
		Protocol:    r.Proto,
		Secure:      r.TLS != nil,
		Xhr:         isXhr(),
		OriginalURL: r.URL.Path,
		Cookies:     make(map[string]string),
		Query:       make(map[string]string),
		Params:      make(map[string]string),
		Method:      r.Method,
		Body:        Body{r},
		Url:         r.URL,
		r:           r,
		Path:        r.URL.Path,
		Fresh:       checkFreshness(r, w),
	}

	for _, cookie := range r.Cookies() {
		req.Cookies[cookie.Name] = cookie.Value
	}

	for key, value := range r.URL.Query() {
		req.Query[key] = value[0]
	}

	for _, param := range params {
		req.Params[param.Key] = param.Value
	}
	return req
}

type Body struct {
	req *http.Request
}

// JSON marshals the request body into the given interface.
// It returns an error if the request body is not a valid JSON or if the
// given interface is not a pointer.
func (body *Body) JSON(dest interface{}) error {

	if body.req.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(body.req.Header, "Content-Type")
		if value != "application/json" {
			return Error{http.StatusUnsupportedMediaType, "unsupported media type, expected 'application/json'"}
		}
	}

	bdy, err := io.ReadAll(body.req.Body)
	if err != nil {
		return errors.New("error reading json payload")
	}
	return json.Unmarshal(bdy, dest)
}

// Text returns the request body as a string.
func (body *Body) Text() (string, error) {
	bdy, err := io.ReadAll(body.req.Body)
	if err != nil {
		return "", errors.New("error reading text payload")
	}

	return string(bdy), nil
}

// FormData returns the body form data, expects request sent with `x-www-form-urlencoded` header
func (body *Body) FormData() (map[string]interface{}, error) {
	if err := body.req.ParseForm(); err != nil {
		return nil, err
	}

	data := make(map[string]interface{})

	for key, value := range body.req.Form {
		data[key] = value[0]
	}

	return data, nil
}

func checkFreshness(req *http.Request, w http.ResponseWriter) bool {
	if req.Method != "GET" && req.Method != "HEAD" {
		return false
	}
	return fresh.IsFresh(req.Header, w.Header())
}

// Accepts returns the value of the incoming request’s “Accept” HTTP header field
// the first match is returned.
// TODO: implement
func (req *Request) Accepts(mime ...string) string {

	return ""
}

// AcceptsCharsets returns true if the incoming request’s “Accept-Charset” HTTP header field
// includes the given charset.
func (req *Request) AcceptsCharsets(charset string) bool {
	return false
}

// AcceptsEncodings returns true if the incoming request’s “Accept-Encoding” HTTP header field
// includes the given encoding.
func (req *Request) AcceptsEncodings(encoding string) bool {
	return false
}

// AcceptsLanguages returns true if the incoming request’s “Accept-Language” HTTP header field
// includes the given language.
func (req *Request) AcceptsLanguages(lang string) bool {
	return false
}

// Get returns the value of param `name` when present or `defaultValue`.
func (req *Request) Get(name string, defaultValue string) string {
	if value, ok := req.Params[name]; ok {
		return value
	}
	return defaultValue
}

// Is returns true if the incoming request’s “Content-Type” HTTP header field
// matches the given mime type.
func (req *Request) Is(mime string) bool {
	return false
}

// Range returns the first range found in the request’s “Range” header field.
// If the “Range” header field is not present or the range is unsatisfiable,
// nil is returned.
// func (req *Request) Range(size int64) *header.Range {
// 	return nil
// }

// TODO: implement
// parseAccept parses the Accept header field and returns a slice of strings accepted.
func parseAccept(header string) []string {

	accepts := strings.Split(header, ",")
	accepted := make([]string, 0, len(accepts))

	// iterate over the accepts and parse them, then add them to the accepted slice
	// symbols are not supported, so we can just split on ';'
	for _, accept := range accepts {
		accept = strings.Split(accept, ";")[0]

		// if the accept is a wildcard, we can just return it
		if accept == "*" {
			return []string{"*"}
		}

		// if the accept is a valid mime type, add it to the accepted slice
		if media, _, err := mime.ParseMediaType(accept); err == nil {

			// if the media type is a wildcard, we can just return it
			if strings.Contains(media, "/*") {
				return []string{media}
			}

			// if the media type is a valid mime type, add it to the accepted slice
			if _, _, err := mime.ParseMediaType(media); err == nil {
				accepted = append(accepted, media)
			}
		}
	}
	return accepted
}

// TODO: implement
func convertToMimeTypes(mimeTypes []string) []string {
	mimes := make([]string, 0, len(mimeTypes))
	for _, mimeType := range mimeTypes {
		if strings.Contains(mimeType, "/") {
			mimes = append(mimes, mimeType)
		} else {
			mimes = append(mimes, mime.TypeByExtension(mimeType))
		}
	}
	return mimes
}
