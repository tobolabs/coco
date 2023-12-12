package coco

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestJSONError_Error(t *testing.T) {
	err := Error{
		Code:    400,
		Message: "Bad Request",
	}

	assert.Equal(t, "Bad Request", err.Error())
}

func TestBody_JSON(t *testing.T) {
	t.Run("it should unmarshal JSON successfully", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"key": "value"}`)), // Valid JSON
			},
		}

		var data map[string]string
		err := body.JSON(&data)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if data["key"] != "value" {
			t.Errorf("Expected key to be 'value', got '%s'", data["key"])
		}
	})

	t.Run("it should return error for nil destination", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"key": "value"}`)),
			},
		}

		err := body.JSON(nil)
		if err == nil {
			t.Fatalf("Expected error for nil destination, got nil")
		}

		var jsonErr JSONError
		if !errors.As(err, &jsonErr) || jsonErr.Status != http.StatusBadRequest {
			t.Fatalf("Expected JSONError with StatusBadRequest, got %v", err)
		}
	})

	t.Run("it should return error for unsupported media type", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Header: http.Header{
					"Content-Type": []string{"text/plain"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"key": "value"}`)),
			},
		}

		var data map[string]string
		err := body.JSON(&data)
		if err == nil {
			t.Fatalf("Expected error for unsupported media type, got nil")
		}

		var jsonErr JSONError
		if !errors.As(err, &jsonErr) || jsonErr.Status != http.StatusUnsupportedMediaType {
			t.Fatalf("Expected JSONError with StatusUnsupportedMediaType, got %v", err)
		}
	})

	t.Run("it should return error for invalid JSON", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"key": "value`)), // Invalid JSON
			},
		}

		var data map[string]string
		err := body.JSON(&data)
		if err == nil {
			t.Fatalf("Expected error for invalid JSON, got nil")
		}

		var jsonErr JSONError
		if !errors.As(err, &jsonErr) || jsonErr.Status != http.StatusBadRequest {
			t.Fatalf("Expected JSONError with StatusBadRequest, got %v", err)
		}
	})

	t.Run("it should return error when reading body fails", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(errReader{}),
			},
		}

		var data map[string]string
		err := body.JSON(&data)
		if err == nil {
			t.Fatalf("Expected error when reading body fails, got nil")
		}

		var jsonErr JSONError
		if !errors.As(err, &jsonErr) || jsonErr.Status != http.StatusBadRequest {
			t.Fatalf("Expected JSONError with StatusInternalServerError, got %v", err)
		}
	})
}

func TestBody_Text(t *testing.T) {
	t.Run("it should return body as string", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Body: io.NopCloser(bytes.NewBufferString("test body content")),
			},
		}

		text, err := body.Text()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if text != "test body content" {
			t.Errorf("Expected body to be 'test body content', got '%s'", text)
		}
	})

	t.Run("it should return error when closing body fails", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Body: &badCloser{Reader: bytes.NewBufferString("test body content")},
			},
		}

		_, err := body.Text()
		if err == nil {
			t.Fatalf("Expected error when closing body, got nil")
		}
		if !strings.Contains(err.Error(), "error closing request body") {
			t.Errorf("Expected error closing request body, got %v", err)
		}
	})

	t.Run("it should return error when reading body fails", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Body: io.NopCloser(errReader{}),
			},
		}

		_, err := body.Text()
		if err == nil {
			t.Fatalf("Expected error when reading body, got nil")
		}
		if !strings.Contains(err.Error(), "error reading text payload") {
			t.Errorf("Expected error reading text payload, got %v", err)
		}
	})
}

func TestBody_FormData(t *testing.T) {
	t.Run("it should handle FormData successfully", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest("POST", "/", strings.NewReader("key1=value1&key2=value2"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		request, _ := newRequest(req, w, nil, app)

		body := request.Body

		data, err := body.FormData()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(data) != 2 || data["key1"][0] != "value1" || data["key2"][0] != "value2" {
			t.Errorf("FormData did not parse correctly: got %+v", data)
		}
	})

	t.Run("it should return error for unsupported Content-Type", func(t *testing.T) {
		body := &Body{
			req: &http.Request{
				Header: http.Header{
					"Content-Type": []string{"text/plain"},
				},
				Body: io.NopCloser(strings.NewReader("key1=value1&key2=value2")),
			},
		}

		_, err := body.FormData()
		if err == nil {
			t.Fatalf("Expected error for unsupported Content-Type, got nil")
		}

		if !strings.Contains(err.Error(), "unsupported Content-Type: text/plain") {
			t.Errorf("Expected specific error message for unsupported Content-Type, got %v", err)
		}
	})

	t.Run("it should return error when reading body fails", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest("POST", "/", io.NopCloser(errReader{}))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		request, _ := newRequest(req, w, nil, app)

		body := request.Body
		_, err := body.FormData()
		if err == nil {
			t.Fatalf("Expected error when reading body, got nil")
		}

		if !strings.Contains(err.Error(), "failed to parse form data") {
			t.Errorf("Expected specific error message when reading body fails, got %v", err)
		}
	})

	t.Run("it should return error when parsing form fails", func(t *testing.T) {
		app := NewApp()

		req := httptest.NewRequest("POST", "/", strings.NewReader("%"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		request, _ := newRequest(req, w, nil, app)

		body := request.Body

		_, err := body.FormData()
		if err == nil {
			t.Fatalf("Expected error when parsing form, got nil")
		}

		if !strings.Contains(err.Error(), "failed to parse form data") {
			t.Errorf("Expected specific error message when parsing form fails, got %v", err)
		}
	})
}

func TestRequest_Cookie(t *testing.T) {
	t.Run("it should retrieve a cookie if it exists", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		req.AddCookie(&http.Cookie{Name: "session_token", Value: "abc123"})
		request, _ := newRequest(req, w, nil, app)

		cookie, ok := request.Cookie("session_token")

		// Assertion
		if !ok {
			t.Fatal("Expected cookie to exist")
		}
		if cookie != "abc123" {
			t.Errorf("Expected cookie value to be 'abc123', got '%s'", cookie)
		}
	})

	t.Run("it should return an error if the cookie does not exist", func(t *testing.T) {
		// Setup
		req := &http.Request{Header: http.Header{"Cookie": []string{"session_token=abc123"}}}
		request := &Request{r: req}

		// Execution
		_, ok := request.Cookie("nonexistent_cookie")
		if ok {
			t.Fatal("Expected error for nonexistent cookie, got nil")
		}
	})
}

func TestRequest_Get(t *testing.T) {
	t.Run("it should retrieve the param value if it exists", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest("GET", "/?name=test", nil)
		req.Header.Add("X-Custom-Header", "value123")
		w := httptest.NewRecorder()

		params := httprouter.Params{
			httprouter.Param{Key: "id", Value: "123"},
		}

		request, _ := newRequest(req, w, params, app)

		value := request.Get("X-Custom-Header")

		if value != "value123" {
			t.Errorf("Expected header value to be 'value123', got '%s'", value)
		}

		param := request.GetParam("id")

		if param != "123" {
			t.Errorf("Expected param value to be '123', got '%s'", param)
		}

		query := request.QueryParam("name")

		if query != "test" {
			t.Errorf("Expected query value to be 'test', got '%s'", query)
		}

	})
}

func TestRequest_Is(t *testing.T) {
	t.Run("it should return true if the content type matches", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Add("Content-Type", "application/json")
		w := httptest.NewRecorder()

		request, _ := newRequest(req, w, nil, app)

		isJSON := request.Is("application/json")

		// Assertion
		if !isJSON {
			t.Error("Expected content type to be JSON")
		}
	})

	t.Run("it should return false if the content type does not match", func(t *testing.T) {
		app := NewApp()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Add("X-Custom-Header", "value123")
		w := httptest.NewRecorder()

		params := httprouter.Params{
			httprouter.Param{Key: "id", Value: "123"},
		}

		request, _ := newRequest(req, w, params, app)

		isJSON := request.Is("application/json")

		if isJSON {
			t.Error("Expected content type not to be JSON")
		}
	})
}

func TestRequest_Range(t *testing.T) {
	t.Run("it should return nil if no Range header is present", func(t *testing.T) {
		req := &http.Request{Header: http.Header{}}
		request := &Request{r: req}

		ranges, err := request.Range(1000)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if ranges != nil {
			t.Errorf("Expected nil ranges, got %v", ranges)
		}
	})

	t.Run("it should return error for invalid range specifier", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"invalid=0-499"}}}
		request := &Request{r: req}

		_, err := request.Range(1000)
		if err == nil {
			t.Fatal("Expected an error for invalid range specifier, got nil")
		}
	})

	t.Run("it should return error for invalid range format", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"bytes=500"}}}
		request := &Request{r: req}

		_, err := request.Range(1000)
		if err == nil {
			t.Fatal("Expected an error for invalid range format, got nil")
		}
	})

	t.Run("it should return error for invalid range start value", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"bytes=abc-500"}}}
		request := &Request{r: req}

		_, err := request.Range(1000)
		if err == nil {
			t.Fatal("Expected an error for invalid range start value, got nil")
		}
	})

	t.Run("it should return error for invalid range end value", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"bytes=0-xyz"}}}
		request := &Request{r: req}

		_, err := request.Range(1000)
		if err == nil {
			t.Fatal("Expected an error for invalid range end value, got nil")
		}
	})

	t.Run("it should return error for unsatisfiable range", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"bytes=1000-2000"}}}
		request := &Request{r: req}

		_, err := request.Range(500)
		if err == nil {
			t.Fatal("Expected an error for unsatisfiable range, got nil")
		}
	})

	t.Run("it should return satisfiable range", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"bytes=0-499"}}}
		request := &Request{r: req}

		ranges, err := request.Range(1000)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(ranges) != 1 || ranges[0].Start != 0 || ranges[0].End != 499 {
			t.Errorf("Expected range [0-499], got %v", ranges)
		}
	})

	t.Run("it should handle multiple ranges, including unsatisfiable ones", func(t *testing.T) {
		req := &http.Request{Header: http.Header{"Range": []string{"bytes=0-499,1000-1500"}}}
		request := &Request{r: req}

		ranges, err := request.Range(1000)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(ranges) != 1 || ranges[0].Start != 0 || ranges[0].End != 499 {
			t.Errorf("Expected only satisfiable range [0-499], got %v", ranges)
		}
	})
}

func TestRequest_Context(t *testing.T) {
	t.Run("it should return the request's context", func(t *testing.T) {
		// Setup
		req := &http.Request{Header: http.Header{}}
		request := &Request{r: req}

		// Execution
		ctx := request.Context()

		// Assertion
		if ctx == nil {
			t.Fatal("Expected context to be non-nil")
		}
	})
}

func Test_newRequest(t *testing.T) {
	// Create a new App instance
	app := NewApp()

	validRequest := httptest.NewRequest("GET", "http://example.com/path", nil)
	validRequest.RemoteAddr = "192.0.2.1:1234"
	// Define test cases
	tests := []struct {
		name           string
		expectedResult func() *Request
		expectError    bool
	}{
		{
			name: "valid request",
			expectedResult: func() *Request {
				return &Request{
					BaseURL:     "/",
					HostName:    "example.com",
					Ip:          "192.0.2.1",
					Protocol:    "HTTP/1.1",
					Secure:      false,
					Xhr:         false,
					OriginalURL: &url.URL{Scheme: "http", Host: "example.com", Path: "/path"},
					Method:      "GET",
					Path:        "/path",
					Fresh:       false,
					Stale:       true,
					r:           validRequest,
					Body:        Body{req: validRequest},
				}
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			params := httprouter.Params{}

			got, err := newRequest(validRequest, w, params, app)
			if (err != nil) != tc.expectError {
				t.Fatalf("newRequest() error = %v, wantErr %v", err, tc.expectError)
			}
			if err != nil {
				return
			}

			expected := tc.expectedResult()

			// Compare the expected and got Request struct fields individually
			if got.BaseURL != expected.BaseURL ||
				got.HostName != expected.HostName ||
				got.Ip != expected.Ip ||
				got.Protocol != expected.Protocol ||
				got.Secure != expected.Secure ||
				got.Xhr != expected.Xhr ||
				got.Method != expected.Method ||
				got.Path != expected.Path ||
				got.Fresh != expected.Fresh ||
				got.Stale != expected.Stale {
				t.Errorf("newRequest() fields do not match the expected result")
			}

			// Compare the OriginalURL separately
			if got.OriginalURL.String() != expected.OriginalURL.String() {
				t.Errorf("newRequest() OriginalURL got = %v, want %v", got.OriginalURL, expected.OriginalURL)
			}
		})
	}
}

type errReader struct{}

func (e errReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

type badCloser struct {
	Reader *bytes.Buffer
}

func (bc *badCloser) Read(p []byte) (n int, err error) {
	return bc.Reader.Read(p)
}

func (bc *badCloser) Close() error {
	return errors.New("error closing request body")
}
