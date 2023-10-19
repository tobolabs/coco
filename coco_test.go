package coco_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/tobolabs/coco/testutils"

	"github.com/tobolabs/coco"
)

func TestCocoApp(t *testing.T) {
	app := coco.NewApp()

	adminRouter := app.Router("/admin")

	if adminRouter.Path() != "/admin" {
		t.Errorf("Expected base to be /admin, got %s", adminRouter.Path())
	}

	if app.Path() != "/" {
		t.Errorf("Expected base to be /, got %s", app.Path())
	}

	app.Enable("trust proxy")
	if v := app.Enabled("trust proxy"); !v {
		t.Errorf("Expected enabled proxy to be true, got %v", v)
	}

	if v := app.Disabled("trust proxy"); v {
		t.Errorf("Expected disabled proxy to be false, got %v", v)
	}

	app.Disable("trust proxy")
	if v := app.Enabled("trust proxy"); v {
		t.Errorf("Expected enabled proxy to be false, got %v", v)
	}

	testRouteMethods(t, app)

	app.SetX("title", "My Site")
	if v := app.GetX("title"); v != "My Site" {
		t.Errorf("Expected app.GetX to be 'My Site', got %v", v)
	}

}

func testRouteMethods(t *testing.T, app *coco.App) {

	cases := make([]testutils.Mock, 0)

	app.All("zero", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("0")
	})

	allCases := []string{"GET", "POST", "DELETE", "PATCH", "PUT"}

	for _, method := range allCases {
		cases = append(cases, testutils.Mock{
			Request: testutils.Request{
				Method: method,
				Path:   "/zero",
			},
			Response: testutils.Response{
				Status: 200,
				Body:   "0",
			},
		})
	}

	app.Get("one", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("1")
	})

	cases = append(cases, testutils.Mock{
		Request: testutils.Request{
			Method: "GET",
			Path:   "/one",
		},
		Response: testutils.Response{
			Status: 200,
			Body:   "1",
		},
	})

	app.Post("two", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("2")
	})

	cases = append(cases, testutils.Mock{
		Request: testutils.Request{
			Method: "POST",
			Path:   "/two",
		},
		Response: testutils.Response{
			Status: 200,
			Body:   "2",
		},
	})

	app.Delete("three", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("3")
	})

	cases = append(cases, testutils.Mock{
		Request: testutils.Request{
			Method: "DELETE",
			Path:   "/three",
		},
		Response: testutils.Response{
			Status: 200,
			Body:   "3",
		},
	})

	app.Patch("four", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("4")
	})

	cases = append(cases, testutils.Mock{
		Request: testutils.Request{
			Method: "PATCH",
			Path:   "/four",
		},
		Response: testutils.Response{
			Status: 200,
			Body:   "4",
		},
	})

	app.Put("five", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("5")
	})

	cases = append(cases, testutils.Mock{
		Request: testutils.Request{
			Method: "PUT",
			Path:   "/five",
		},
		Response: testutils.Response{
			Status: 200,
			Body:   "5",
		},
	})

	for _, c := range cases {
		t.Run(fmt.Sprintf("--%s %s", c.Request.Method, c.Request.Path), func(t *testing.T) {
			rc := httptest.NewRecorder()
			req := httptest.NewRequest(c.Request.Method, c.Request.Path, nil)

			app.ServeHTTP(rc, req)

			if rc.Code != c.Response.Status {
				t.Errorf("Expected status code %d, got %d", c.Response.Status, rc.Code)
			}

			if rc.Body.String() != c.Response.Body {
				t.Errorf("Expected body to be '%s', got %s", c.Response.Body, rc.Body.String())

			}
		})
	}

}
