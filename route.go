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

type Path struct {
	name     string
	handlers []Handler
	method   string
}

type Route struct {
	base   string
	hr     *httprouter.Router
	parent *Route
	// Middleware
	middleware    []Handler
	paths         []Path
	paramHandlers map[string]ParamHandler
	app           *App

	// gotta flex my data structures knowledge, so allow me cook ✋🏽
	cachedMiddleware []Handler
	rootNode         bool
	children         map[string]*Route
}

func (r *Route) printRoutes(prefix string) {
	fmt.Printf("%s%s with (%d) children  ", prefix, r.base, len(r.paths))
	fmt.Printf("children: %d\n", len(r.children))
	for _, child := range r.children {
		//fmt.Printf("%s%s\n", prefix, child.base)
		child.printRoutes(prefix + r.base + "->")
	}
}

// Router is equivalent of app.route(path), returns a new instance of route
func (r *Route) Router(path string) *Route {
	fmt.Printf("Router: is creating a child %s\n", r.base)
	return r.app.newRoute(path, false, r)
}

func (r *Route) Use(middleware ...Handler) *Route {
	r.middleware = append(r.middleware, middleware...)
	return r
}

func (r *Route) combineHandlers(handlers ...Handler) []Handler {
	middlewares := make([]Handler, 0)
	middlewares = append(middlewares, r.fetchMiddleware()...)
	return append(middlewares, handlers...)
}

func (a *App) pathify(p string) string {
	if p == "" {
		p = "/"
	}
	clean := fp.Clean(p)
	if clean[0] != '/' {
		clean = "/" + clean
	}

	return a.basePath + clean
}

func (a *App) newRoute(path string, isRoot bool, parent *Route) *Route {
	var r Route
	path = a.pathify(path)

	if isRoot {
		r = Route{
			base:     path,
			hr:       a.base,
			app:      a,
			rootNode: true,
			parent:   nil,
			children: make(map[string]*Route),
		}
		a.Route = &r
	} else {
		if parent.parent != nil {
			path = parent.base + path
		}
		r = Route{
			base:     path,
			hr:       a.base,
			app:      a,
			parent:   parent,
			children: make(map[string]*Route),
		}
		parent.children[path] = &r
	}

	return &r
}

func (r *Route) getfullPath(path string) string {
	raw := strings.Trim(path, "/")
	var builder strings.Builder

	builder.WriteString(strings.TrimSuffix(r.base, "/"))
	if len(raw) > 0 {
		builder.WriteRune('/')
		builder.WriteString(raw)
	}

	return builder.String()
}

func (r *Route) fetchMiddleware() []Handler {

	if len(r.cachedMiddleware) > 0 {
		return r.cachedMiddleware
	}

	middleware := append([]Handler{}, r.middleware...)

	current := r.parent
	for current != nil {
		if len(current.cachedMiddleware) > 0 {
			middleware = append(middleware, current.cachedMiddleware...)
			break
		}
		middleware = append(middleware, current.middleware...)
		current = current.parent
	}

	r.cachedMiddleware = middleware

	return middleware
}

func (r *Route) handle(httpMethod string, path string, handlers []Handler) {
	newPath := Path{
		name:     r.getfullPath(path),
		handlers: handlers,
		method:   httpMethod,
	}

	if r.paths == nil {
		r.paths = make([]Path, 0)
	}
	r.paths = append(r.paths, newPath)
}

func execParamChain(ctx *reqcontext, params httprouter.Params, handlers map[string]ParamHandler) {
	if len(handlers) == 0 {
		return
	}
	pending := make([]Handler, 0)
	for _, p := range params {
		if h, ok := handlers[p.Key]; ok {
			fn := func(rw Response, req *Request, next NextFunc) {
				h(rw, req, next, p.Value)
			}
			pending = append(pending, fn)
		}
	}

	ctx.handlers = append(pending, ctx.handlers...)

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

// Param calls the given handler when the route param matches the given param.
// The handler is passed the value of the param.
func (r *Route) Param(param string, handler ParamHandler) {
	if r.paramHandlers == nil {
		r.paramHandlers = make(map[string]ParamHandler)
	}
	r.paramHandlers[param] = handler
}

func (r *Route) Path() string {
	return r.base
}
