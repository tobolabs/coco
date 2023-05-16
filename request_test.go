package coco_test

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/noelukwa/coco"
	"github.com/noelukwa/coco/testutils"
)

func TestRequest(t *testing.T) {
	app := coco.NewApp()

	router := app.Router("/test")

	router.All("/hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		base := req.BaseURL
		if base != "/test" {
			t.Errorf("Expected base url to be /test, got %s", base)
		}

		if req.Method == "POST" {

			var body = struct {
				Name string `json:"name"`
			}{}
			err := req.Body.JSON(&body)
			if err != nil {
				t.Errorf("Expected no error, got %s", err.Error())
			}

			if body.Name != "Noel" {
				t.Errorf("Expected name to be Noel, got %s", body.Name)
			}
		}

		if req.Method == "PUT" {
			var body string
			body, err := req.Body.Text()
			if err != nil {
				t.Errorf("Expected no error, got %s", err.Error())
			}

			if body != "Noel" {
				t.Errorf("Expected name to be Noel, got %s", body)
			}
		}

		if req.Method == "PATCH" {
			var body map[string]interface{}
			body, err := req.Body.FormData()
			if err != nil {
				t.Errorf("Expected no error, got %s", err.Error())
			}

			if body["name"] != "Noel" {
				t.Errorf("Expected name to be Noel, got %s", body)
			}
		}

		cookies := req.Cookies
		if cookies["session"] != "123" {
			t.Errorf("Expected session cookie to be 123, got %s", cookies["session"])
		}

		if req.Method == "GET" || req.Method == "HEAD" {
			isFresh := req.Fresh
			if !isFresh {
				t.Errorf("Expected request to be fresh")
			}
		}

		hostName := req.HostName
		if hostName != "example.com" {
			t.Errorf("Expected host name to be localhost, got %s", hostName)
		}

		ip := req.Ip
		if ip != "192.0.2.1:1234" {
			t.Errorf("Expected ip to be 127.0.0.1 got %s", ip)
		}

		ips := req.Ips
		if len(ips) != 0 {
			t.Errorf("Expected ips to be zero")
		}

		protocol := req.Protocol
		if protocol != "HTTP/1.1" {
			t.Errorf("Expected protocol to be HTTP/1.1 got %s", protocol)
		}

		secure := req.Secure
		if secure {
			t.Errorf("Expected request to be insecure")
		}

		stale := req.Stale
		if stale {
			t.Errorf("Expected request to be fresh")
		}

		subdomains := req.Subdomains
		if len(subdomains) != 0 {
			t.Errorf("Expected subdomains to be empty, got %s", subdomains)
		}

		query := req.Query
		if query["name"] != "Noel" {
			t.Errorf("Expected query name to be Noel, got %s", query["name"])
		}

		xhr := req.Xhr
		if !xhr {
			t.Errorf("Expected request to be xhr")
		}

		originalUrl := req.OriginalURL
		if originalUrl != "/test/hello" {
			t.Errorf("Expected original url to be /test/hello, got %s", originalUrl)
		}

		path := req.Path
		if path != "/hello" {
			t.Errorf("Expected path to be /hello, got %s", path)
		}

		xcookies := req.SignedCookies
		if xcookies["session"] != "123" {
			t.Errorf("Expected session cookie to be 123, got %s", xcookies["session"])
		}

		//accepts := req.Accepts("application/json")
		//if !accepts {
		//	t.Errorf("Expected request to accept application/json")
		//}

		acceptsCharsets := req.AcceptsCharsets("utf-8")
		if !acceptsCharsets {
			t.Errorf("Expected request to accept utf-8")
		}

		acceptsEncodings := req.AcceptsEncodings("gzip")
		if !acceptsEncodings {
			t.Errorf("Expected request to accept gzip")
		}

		acceptsLanguages := req.AcceptsLanguages("en")
		if !acceptsLanguages {
			t.Errorf("Expected request to accept en")
		}
	})

	cases := []testutils.Mock{
		{
			Request: testutils.Request{
				Method: "POST",
				Path:   "/test/hello",
				Body:   `{"name": "Noel"}`,
			},
			Response: testutils.Response{
				Status: 200,
				Body:   "",
			},
		},
		{
			Request: testutils.Request{
				Method: "PUT",
				Path:   "/test/hello",
				Body:   "Noel",
			},
			Response: testutils.Response{
				Status: 200,
				Body:   "",
			},
		},
		{
			Request: testutils.Request{
				Method: "PATCH",
				Path:   "/test/hello",
				Body:   "name=Noel",
			},
			Response: testutils.Response{
				Status: 200,
				Body:   "",
			},
		},
	}

	for _, c := range cases {
		Do(t, app, c)
	}

}

func Do(t *testing.T, app *coco.App, c testutils.Mock) {
	rc := httptest.NewRecorder()

	req := httptest.NewRequest(c.Request.Method, fmt.Sprintf("%s?name=Noel", c.Request.Path), bytes.NewBuffer([]byte(c.Request.Body)))
	req.Header.Set("Cookie", "session=123")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Host", "localhost")
	req.Header.Set("Referer", "http://localhost/test/hello")
	// create a fresh request
	req.Header.Set("If-Modified-Since", time.Now().Add(-time.Hour).Format(time.RFC1123))

	app.ServeHTTP(rc, req)

	t.Run(fmt.Sprintf("%s", c.Request.Method), func(t *testing.T) {
		if rc.Code != c.Response.Status {
			t.Errorf("Expected status code %d, got %d", c.Response.Status, rc.Code)
		}

		if rc.Body.String() != c.Response.Body {
			t.Errorf("Expected response body %s, got %s", c.Response.Body, rc.Body.String())
		}
	})

}
