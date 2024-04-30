package coco_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/tobolabs/coco"
)

func TestRoute_Use(t *testing.T) {
	app := coco.NewApp()

	middlewareCalled := false
	middleware := func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		middlewareCalled = true
		next(res, req)
	}

	app.Use(middleware)

	app.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.SendStatus(http.StatusOK)
	})

	srv := httptest.NewServer(app)
	defer srv.Close()

	response, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("Could not make GET request: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected response status to be %d, got %d", http.StatusOK, response.StatusCode)
	}

	if !middlewareCalled {
		t.Errorf("Middleware was not called")
	}
}

func TestRoute_Router(t *testing.T) {
	app := coco.NewApp()
	subRouter := app.NewRouter("/sub")

	if subRouter.Path() != "/sub" {
		t.Errorf("Expected subRouter path to be '/sub', got '%s'", subRouter.Path())
	}
}

func TestRoute_Head(t *testing.T) {
	app := coco.NewApp()

	handlerCalled := false
	app.Head("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		handlerCalled = true
	})

	srv := httptest.NewServer(app)
	defer srv.Close()

	req, err := http.NewRequest("HEAD", srv.URL+"/", nil)
	if err != nil {
		t.Fatalf("Could not make HEAD request: %v", err)
	}

	w := httptest.NewRecorder()

	app.ServeHTTP(w, req)

	if !handlerCalled {
		t.Errorf("HEAD handler was not called")
	}
}

func TestRoute_Path(t *testing.T) {
	app := coco.NewApp()
	route := app.NewRouter("/test")

	if route.Path() != "/test" {
		t.Errorf("Expected route path to be '/test', got '%s'", route.Path())
	}
}

func TestRoute_Static(t *testing.T) {
	// Create a new App instance
	app := coco.NewApp()

	// Create an in-memory file system to simulate static files
	fs := fstest.MapFS{
		"hello.txt": &fstest.MapFile{
			Data: []byte("Hello, world!"),
		},
	}

	app.Static(fs, "/static")

	app.Get("/foo", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.SendStatus(http.StatusOK)
	})

	srv := httptest.NewServer(app)
	defer srv.Close()

	// Define test cases
	tests := []struct {
		path     string
		expected string
		status   int
	}{
		{"/static/hello.txt", "Hello, world!", http.StatusOK},
		{"/static/nonexistent.txt", "", http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("Failed to make GET request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Errorf("Expected status code %d, got %d", tc.status, resp.StatusCode)
			}

			if tc.status == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				if string(body) != tc.expected {
					t.Errorf("Expected body to be %q, got %q", tc.expected, string(body))
				}
			}
		})
	}
}
