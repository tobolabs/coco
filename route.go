package coco

import (
	"fmt"
	"io/fs"
	"net/http"
	fp "path"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type allowedMethod int

const (
	GET allowedMethod = 1 << iota
	POST
	PUT
	DELETE
	PATCH
	OPTIONS
	HEAD
	ALL
)

var methods = map[allowedMethod]string{
	GET:     "GET",
	POST:    "POST",
	PUT:     "PUT",
	DELETE:  "DELETE",
	PATCH:   "PATCH",
	OPTIONS: "OPTIONS",
	HEAD:    "HEAD",
}

type Route struct {
	base   string
	hr     *httprouter.Router
	parent bool
	// Middleware
	middleware []Handler

	app *App
}

func (r *Route) Router(path string) *Route {
	return r.app.newRoute(path)
}

func (r *Route) Use(middleware ...Handler) *Route {
	r.middleware = append(r.middleware, middleware...)
	return r
}

func (r *Route) combineHandlers(handlers ...Handler) []Handler {
	middlewares := make([]Handler, 0)
	middlewares = append(middlewares, r.app.middleware...)
	if !r.parent {
		middlewares = append(middlewares, r.middleware...)
	}
	return append(middlewares, handlers...)
}

func (a *App) pathify(p string) string {

	clean := fp.Clean(p)
	if clean[0] != '/' {
		clean = "/" + clean
	}

	return a.basePath + clean
}

func (a *App) newRoute(path string) *Route {
	var r Route
	if path == "" {
		path = "/"
		r.parent = true
	}
	path = a.pathify(path)

	if r, ok := a.routes[path]; ok {
		return r
	} else {
		a.routes[path] = r
	}
	r.base = path
	r.hr = a.base
	r.app = a
	return &r
}

func (r *Route) getfullPath(path string) string {

	raw := strings.Trim(path, "/")

	if len(raw) > 0 && raw[0] == ':' {
		return r.base + "/" + raw
	}

	if strings.HasSuffix(r.base, "/") {
		return fmt.Sprintf("%s%s", r.base, raw)
	}
	return fmt.Sprintf("%s/%s", r.base, raw)
}

func (r *Route) handle(httpMethod string, path string, handlers []Handler) {
	handlers = r.combineHandlers(handlers...)
	r.hr.Handle(httpMethod, r.getfullPath(path), func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {

		request := newRequest(req, w, p)

		accepts := parseAccept(req.Header.Get("Accept"))

		ctx := &reqcontext{
			handlers:  handlers,
			templates: r.app.templates,
			req:       &request,
			accepted:  accepts,
		}
		response := Response{w, ctx, 0}

		ctx.next(response, &request)
	})
}

func (r *Route) Get(path string, handlers ...Handler) *Route {
	r.handle("GET", path, handlers)
	return r
}

func (r *Route) Post(path string, handlers ...Handler) {
	r.handle("POST", path, handlers)
}

func (r *Route) Put(path string, handlers ...Handler) {
	r.handle("PUT", path, handlers)
}

func (r *Route) Delete(path string, handlers ...Handler) *Route {
	r.handle("DELETE", path, handlers)
	return r
}

func (r *Route) Patch(path string, handlers ...Handler) {
	r.handle("PATCH", path, handlers)
}

func (r *Route) Options(path string, handlers ...Handler) {
	r.handle("OPTIONS", path, handlers)
}

func (r *Route) Head(path string, handlers ...Handler) {
	r.handle("HEAD", path, handlers)
}

func (r *Route) All(path string, handlers ...Handler) *Route {
	for _, v := range methods {
		r.handle(v, path, handlers)
	}
	return r
}

func (r *Route) Group(path string, handlers ...Handler) *Route {
	return &Route{
		base:       r.base + path,
		hr:         r.hr,
		middleware: r.combineHandlers(handlers...),
	}
}

func (r *Route) Static(root fs.FS, path string) {
	fileServer := http.FileServer(http.FS(root))

	r.hr.GET(r.base+path, func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		fileServer.ServeHTTP(w, req)
	})
}

func (r *Route) Param(param string, handler ParamHandler) {
	panic("not implemented")
}
