package coco

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func Test_context_coco(t *testing.T) {
	app := &App{}
	rc := &context{app: app}

	if got := rc.coco(); got != app {
		t.Errorf("rcontext.coco() = %v, want %v", got, app)
	}
}

func Test_context_next_noHandlers(t *testing.T) {
	rc := &context{handlers: []Handler{}}
	rw := Response{ww: wrapWriter(httptest.NewRecorder())}
	req := &Request{r: httptest.NewRequest("GET", "/", nil)}

	rc.next(rw, req)

	if len(rc.handlers) != 0 {
		t.Error("Expected no handlers to be removed from the slice")
	}
	
}

func Test_context_next_oneHandler(t *testing.T) {
	called := false
	handler := func(rw Response, req *Request, next NextFunc) {
		called = true
	}

	rc := &context{handlers: []Handler{handler}}
	rw := Response{ww: wrapWriter(httptest.NewRecorder())}
	req := &Request{r: httptest.NewRequest("GET", "/", nil)}

	rc.next(rw, req)

	if !called {
		t.Error("Expected the handler to be called")
	}

	if len(rc.handlers) != 0 {
		t.Error("Expected the handler to be removed from the slice")
	}
}

func Test_context_next_multipleHandlers(t *testing.T) {
	var callOrder []int
	handler1 := func(rw Response, req *Request, next NextFunc) {
		callOrder = append(callOrder, 1)
		next(rw, req)
	}
	handler2 := func(rw Response, req *Request, next NextFunc) {
		callOrder = append(callOrder, 2)
	}

	rc := &context{handlers: []Handler{handler1, handler2}}
	rw := Response{ww: wrapWriter(httptest.NewRecorder())}
	req := &Request{r: httptest.NewRequest("GET", "/", nil)}

	rc.next(rw, req)

	if !reflect.DeepEqual(callOrder, []int{1, 2}) {
		t.Errorf("Handlers were called in the wrong order: got %v, want %v", callOrder, []int{1, 2})
	}

	if len(rc.handlers) != 0 {
		t.Error("Expected all handlers to be removed from the slice")
	}
}
