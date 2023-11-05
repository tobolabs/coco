package coco

import (
	"fmt"
	"io/fs"
	"net/http"
	fp "path"
	"path/filepath"
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

// Path is a type that represents a path in the application.
type Path struct {
	name     string
	handlers []Handler
	method   string
}

// Route is a type that represents a route in the application.
// It is equivalent to an express.Router instance.
type Route struct {
	base   string
	hr     *httprouter.Router
	parent *Route
	// Middleware
	middleware    []Handler
	paths         []Path
	paramHandlers map[string]ParamHandler
	app           *App

	cachedMiddleware []Handler
	rootNode         bool
	children         map[string]*Route
}

func (r *Route) printRoutes(prefix string) {
	fmt.Printf("%s%s with (%d) paths, children: (%d)  \n", prefix, r.base, len(r.paths), len(r.children))

	for _, child := range r.children {
		//fmt.Printf("%s%s\n", prefix, child.base)
		child.printRoutes(prefix)
	}
}

// Router is equivalent of app.route(path), returns a new instance of Route
func (r *Route) Router(path string) *Route {
	return r.app.newRoute(path, false, r)
}

// Use adds middleware to the route.
func (r *Route) Use(middleware ...Handler) *Route {
	r.middleware = append(r.middleware, middleware...)
	return r
}

func (r *Route) combineHandlers(handlers ...Handler) []Handler {
	middlewares := make([]Handler, 0)
	middlewares = append(middlewares, r.fetchMiddleware()...)
	return append(middlewares, handlers...)
}

func (a *App) makePath(p string) string {
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
	cleanPath := a.makePath(path)

	if isRoot {
		r = Route{
			base:     cleanPath,
			hr:       a.base,
			app:      a,
			rootNode: true,
			parent:   nil,
			children: make(map[string]*Route),
		}
		a.Route = &r
	} else {
		combinedPath := filepath.Join(parent.base, cleanPath)
		r = Route{
			base:     combinedPath,
			hr:       a.base,
			app:      a,
			parent:   parent,
			children: make(map[string]*Route),
		}
		parent.children[combinedPath] = &r
	}

	return &r
}

func (r *Route) getFullPath(path string) string {
	var builder strings.Builder

	builder.WriteString(strings.TrimSuffix(r.base, "/"))
	if len(path) > 0 && path[0] != '/' {
		builder.WriteRune('/')
	}
	builder.WriteString(path)

	return builder.String()
}

func (r *Route) fetchMiddleware() []Handler {
	if r.rootNode {
		return r.middleware
	}

	if r.cachedMiddleware == nil {
		var middleware []Handler
		for current := r; current != nil; current = current.parent {
			middleware = append(current.middleware, middleware...)
		}
		r.cachedMiddleware = middleware
	}

	return r.cachedMiddleware
}

func (r *Route) handle(httpMethod string, path string, handlers []Handler) {
	newPath := Path{
		name:     r.getFullPath(path),
		handlers: handlers,
		method:   httpMethod,
	}

	if r.paths == nil {
		r.paths = make([]Path, 0)
	}
	r.paths = append(r.paths, newPath)
}

func execParamChain(ctx *context, params httprouter.Params, handlers map[string]ParamHandler) {
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

func (r *Route) Static(root fs.FS, path string) {
	strippedPath := "/" + strings.Trim(path, "/")
	fileServer := http.FileServer(http.FS(root))

	r.hr.GET(strippedPath+"/*filepath", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		req.URL.Path = ps.ByName("filepath")
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
