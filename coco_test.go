package coco_test

import (
	"github.com/tobolabs/coco"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	app := coco.NewApp()
	if app == nil {
		t.Fatal("NewApp() should create a new app instance, got nil")
	}

	// Since defaultSettings is unexported, we need to define the expected settings here.
	expectedSettings := map[string]interface{}{
		"x-powered-by":     true,
		"env":              "development",
		"etag":             "weak",
		"trust proxy":      false,
		"subdomain offset": 2,
	}

	if !reflect.DeepEqual(app.Settings(), expectedSettings) {
		t.Errorf("Expected default settings %v, got %v", expectedSettings, app.Settings())
	}
}

func TestAppListen(t *testing.T) {
	app := coco.NewApp()

	// Use a goroutine to listen in a non-blocking way and a channel to communicate the result.
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Listen(":3000")
	}()

	// Give the server a moment to start
	time.Sleep(time.Millisecond * 100)

	// We expect the server to be listening, so there should be no error yet.
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Did not expect ListenAndServe to return an error, got: %v", err)
		}
	default:
		// No error, as expected.
	}

	// Clean up: Close the server.
	if err := app.Close(); err != nil {
		t.Errorf("Failed to close the server: %v", err)
	}
}

func TestCocoApp(t *testing.T) {
	app := coco.NewApp()

	app.All("/chow", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("choww")
	})
	app.Get("/zero", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("0")
	})
	app.Get("/one", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("1")
	})
	app.Post("/two", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("2")
	})
	app.Delete("/three", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("3")
	})
	app.Patch("/four", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("4")
	})
	app.Put("/five", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("5")
	})

	app.Options("/six", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("6")
	})

	app.Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
		res.Send(param)
	})

	app.Get("/test/:id", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("!!")
	})

	app.Get("/echo", func(res coco.Response, req *coco.Request, next coco.NextFunc) {

		app.SetX("test", "test")
		if app.GetX("test") != "test" {
			t.Errorf("Expected app.GetX('test') to be 'test', but got %s", app.GetX("test"))
		}

		app.Enable("someBool")
		if !app.Enabled("someBool") {
			t.Errorf("Expected app.Enabled('someBool') to be true, but got false")
		}

		app.Disable("someBool")
		if !app.Disabled("someBool") {
			t.Errorf("Expected app.Disabled('someBool') to be true, but got true")
		}
	})

	server := httptest.NewServer(app)
	defer server.Close()

	testRouteMethod(t, server.URL, "GET", "/one", "1", 200)
	testRouteMethod(t, server.URL, "POST", "/two", "2", 200)
	testRouteMethod(t, server.URL, "DELETE", "/three", "3", 200)
	testRouteMethod(t, server.URL, "PATCH", "/four", "4", 200)
	testRouteMethod(t, server.URL, "PUT", "/five", "5", 200)

	testRouteMethod(t, server.URL, "ALL", "/chow", "choww", 200)
	testRouteMethod(t, server.URL, "OPTIONS", "/six", "6", 200)

	testRouteMethod(t, server.URL, "GET", "/test/123", "123", 200)
}

func testRouteMethod(t *testing.T, host, method, path, expectedBody string, expectedStatus int) {
	t.Helper()
	t.Run(method+" "+path, func(t *testing.T) {
		url := host + path

		var resp *http.Response
		var err error

		if method == "ALL" {
			for _, m := range []string{"GET", "POST", "DELETE", "PATCH", "PUT"} {
				req, _ := http.NewRequest(m, url, nil)
				resp, err = http.DefaultClient.Do(req)

				if err != nil {
					t.Fatalf("Error making %s request: %v", m, err)
				}

				if resp.StatusCode != expectedStatus {
					t.Errorf("Expected status code %d, but got %d", expectedStatus, resp.StatusCode)
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Error reading response body: %v", err)
				}

				actualBody := string(body)
				if actualBody != expectedBody {
					t.Errorf("Expected response body '%s', but got '%s'", expectedBody, actualBody)
				}
			}
		} else {
			switch method {
			case "GET":
				resp, err = http.Get(url)
			case "POST":
				resp, err = http.Post(url, "application/json", nil)
			case "DELETE":
				req, _ := http.NewRequest("DELETE", url, nil)
				resp, err = http.DefaultClient.Do(req)
			case "PATCH":
				req, _ := http.NewRequest("PATCH", url, nil)
				resp, err = http.DefaultClient.Do(req)
			case "PUT":
				req, _ := http.NewRequest("PUT", url, nil)
				resp, err = http.DefaultClient.Do(req)

			case "OPTIONS":
				req, _ := http.NewRequest("OPTIONS", url, nil)
				resp, err = http.DefaultClient.Do(req)

			default:

				t.Fatalf("Unsupported HTTP method: %s", method)
			}

			if err != nil {
				t.Fatalf("Error making %s request: %v", method, err)
			}

			if resp.StatusCode != expectedStatus {
				t.Errorf("Expected status code %d, but got %d", expectedStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Error reading response body: %v", err)
			}

			actualBody := string(body)
			if actualBody != expectedBody {
				t.Errorf("Expected response body '%s', but got '%s'", expectedBody, actualBody)
			}
		}
	})
}

func TestApp_Disable(t *testing.T) {
	app := coco.NewApp()

	app.Disable("x-powered-by")
	if app.Enabled("x-powered-by") {
		t.Errorf("Expected app.Enabled('x-powered-by') to be false, but got true")
	}
}

func TestApp_Enable(t *testing.T) {
	app := coco.NewApp()

	app.Enable("x-powered-by")
	if app.Disabled("x-powered-by") {
		t.Errorf("Expected app.Disabled('x-powered-by') to be false, but got true")
	}
}

func TestApp_SetX(t *testing.T) {
	app := coco.NewApp()

	app.SetX("test", "test")
	if app.GetX("test") != "test" {
		t.Errorf("Expected app.GetX('test') to be 'test', but got %s", app.GetX("test"))
	}
}

func TestApp_GetX(t *testing.T) {
	app := coco.NewApp()

	app.SetX("test", "test")
	if app.GetX("test") != "test" {
		t.Errorf("Expected app.GetX('test') to be 'test', but got %s", app.GetX("test"))
	}
}

func TestApp_Disabled(t *testing.T) {
	app := coco.NewApp()

	app.Disable("x-powered-by")
	if app.Enabled("x-powered-by") {
		t.Errorf("Expected app.Enabled('x-powered-by') to be false, but got true")
	}

	if !app.Disabled("x-powered-by") {
		t.Errorf("Expected app.Disabled('x-powered-by') to be true, but got false")
	}
}

func TestApp_Enabled(t *testing.T) {
	app := coco.NewApp()

	app.Enable("x-powered-by")
	if app.Disabled("x-powered-by") {
		t.Errorf("Expected app.Disabled('x-powered-by') to be false, but got true")
	}

	if !app.Enabled("x-powered-by") {
		t.Errorf("Expected app.Enabled('x-powered-by') to be true, but got false")
	}
}
