package coco_test

import (
	"bytes"
	"github.com/tobolabs/coco"
	"github.com/tobolabs/coco/testutils"
	"net/http/httptest"
	"testing"
)

func TestRequest(t *testing.T) {
	app := coco.NewApp()

	t.Run("it should contain session cookie", func(t *testing.T) {
		req := testutils.Request{
			Method: "GET",
			Path:   "/test/hello",
			Headers: map[string]string{
				"Cookie": "session=123",
			},
			Body: "",
		}
		app.Get("/test/hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
			if v, ok := req.Cookies["session"]; !ok || v != "123" {
				t.Errorf("Expected session cookie to be 123, got %s", v)
			}
		})

		Do(t, app, req)
	})

	t.Run("it should parse Accept header", func(t *testing.T) {
		req := testutils.Request{
			Method: "GET",
			Path:   "/test/hello",
			Headers: map[string]string{
				"Accept": "text/*,application/json",
			},
			Body: "",
		}
		app.Get("/test/hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
			if v := req.Accepts("html"); v != "html" {
				t.Errorf("Expected Accepts to be html, got %s", v)
			}

			if v := req.Accepts("text/html"); v != "text/html" {
				t.Errorf("Expected Accepts to be text/html, got %s", v)
			}

			if v := req.Accepts("json"); v != "json" {
				t.Errorf("Expected Accepts to be json, got %s", v)
			}

			if v := req.Accepts("application/json"); v != "application/json" {
				t.Errorf("Expected Accepts to be application/json, got %s", v)
			}
		})

		Do(t, app, req)
	})

	t.Run("it should parse weighted Accept header", func(t *testing.T) {
		req := testutils.Request{
			Method: "GET",
			Path:   "/test/hello",
			Headers: map[string]string{
				"Accept": "text/*;q=0.3,application/json;q=0.5",
			},
			Body: "",
		}
		app.Get("/test/hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
			if v := req.Accepts("html"); v != "html" {
				t.Errorf("Expected Accepts to be html, got %s", v)
			}

			if v := req.Accepts("text/html"); v != "text/html" {
				t.Errorf("Expected Accepts to be text/html, got %s", v)
			}

			if v := req.Accepts("json"); v != "json" {
				t.Errorf("Expected Accepts to be json, got %s", v)
			}

			if v := req.Accepts("application/json"); v != "application/json" {
				t.Errorf("Expected Accepts to be application/json, got %s", v)
			}

			if v := req.Accepts("html", "json"); v != "json" {
				t.Errorf("Expected Accepts to be json, got %s", v)
			}
		})

		Do(t, app, req)
	})
}

func Do(t *testing.T, app *coco.App, r testutils.Request) {
	t.Helper()
	rc := httptest.NewRecorder()

	req := httptest.NewRequest(r.Method, r.Path, bytes.NewBufferString(r.Body))
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	//req.Header.Set("Cookie", "session=123")
	//req.Header.Set("X-Requested-With", "XMLHttpRequest")
	//req.Header.Set("Accept", "application/json")
	//req.Header.Set("Accept-Charset", "utf-8")
	//req.Header.Set("Accept-Encoding", "gzip")
	//req.Header.Set("Accept-Language", "en")
	//req.Header.Set("Host", "localhost")
	//req.Header.Set("Referer", "http://localhost/test/hello")
	//// create a fresh request
	//req.Header.Set("If-Modified-Since", time.Now().Add(-time.Hour).Format(time.RFC1123))

	app.ServeHTTP(rc, req)

}
