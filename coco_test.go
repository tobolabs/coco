package coco_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/tobolabs/coco/v2"
)

func TestNewApp(t *testing.T) {
	app := coco.NewApp()
	if app == nil {
		t.Fatal("NewApp() should create a new app instance, got nil")
	}

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

func TestCocoApp_RoutingAndSettings(t *testing.T) {
	app := coco.NewApp()
	setupAppRoutes(app)
	server := httptest.NewServer(app)
	defer server.Close()

	testRoutes := []struct {
		method         string
		path           string
		expectedBody   string
		expectedStatus int
	}{
		{"GET", "/one", "1", 200},
		{"POST", "/two", "2", 200},
		{"DELETE", "/three", "3", 200},
		{"PATCH", "/four", "4", 200},
		{"PUT", "/five", "5", 200},
		{"ALL", "/chow", "choww", 200},
		{"OPTIONS", "/six", "6", 200},
		{"GET", "/test/123", "123", 200},
	}

	for _, test := range testRoutes {
		testRouteMethod(t, server.URL, test.method, test.path, test.expectedBody, test.expectedStatus)
	}

	// Test settings modifications
	app.SetSetting("test", "test")
	if app.GetSetting("test") != "test" {
		t.Errorf("Expected setting 'test' to be 'test', but got %s", app.GetSetting("test"))
	}

	app.SetSetting("someBool", true)
	if !app.IsSettingEnabled("someBool") {
		t.Errorf("Expected 'someBool' setting to be enabled, but it is not")
	}

	app.SetSetting("someBool", false)
	if !app.IsSettingDisabled("someBool") {
		t.Errorf("Expected 'someBool' setting to be disabled, but it is not")
	}
}

func testRouteMethod(t *testing.T, host, method, path, expectedBody string, expectedStatus int) {
	t.Helper()
	t.Run(method+" "+path, func(t *testing.T) {
		url := host + path
		var resp *http.Response
		var err error

		switch method {
		case "ALL":
			methods := []string{"GET", "POST", "DELETE", "PATCH", "PUT"}
			for _, m := range methods {
				req, _ := http.NewRequest(m, url, nil)
				resp, err = http.DefaultClient.Do(req)
				checkResponse(t, m, resp, err, expectedBody, expectedStatus)
			}
		default:
			req, _ := http.NewRequest(method, url, nil)
			resp, err = http.DefaultClient.Do(req)
			checkResponse(t, method, resp, err, expectedBody, expectedStatus)
		}
	})
}

func checkResponse(t *testing.T, method string, resp *http.Response, err error, expectedBody string, expectedStatus int) {
	if err != nil {
		t.Fatalf("Error making %s request: %v", method, err)
	}
	defer resp.Body.Close()

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

func setupAppRoutes(app *coco.App) {
	app.All("/chow", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("choww")
	}).Get("/zero", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("0")
	}).Get("/one", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("1")
	}).Post("/two", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("2")
	}).Delete("/three", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("3")
	}).Patch("/four", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("4")
	}).Put("/five", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("5")
	}).Options("/six", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("6")
	}).Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
		res.Send(param)
	}).Get("/test/:id", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("!!")
	}).Get("/echo", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Echo test")
	})
}

func TestApp_Settings(t *testing.T) {
	app := coco.NewApp()
	app.SetSetting("x-powered-by", false)
	if !app.IsSettingDisabled("x-powered-by") {
		t.Errorf("Expected 'x-powered-by' to be disabled")
	}

	app.SetSetting("x-powered-by", true)
	if !app.IsSettingEnabled("x-powered-by") {
		t.Errorf("Expected 'x-powered-by' to be enabled")
	}
}
