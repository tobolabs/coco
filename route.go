package coco

import (
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

type routePath struct {
	name     string
	handlers []Handler
	method   string
}

type route struct {
	base   string
	hr     *httprouter.Router
	parent *route

	middleware    []Handler
	paths         []routePath
	paramHandlers map[string]ParamHandler
	app           *App

	cachedMiddleware []Handler
	rootNode         bool
	children         map[string]*route
}

// func (r *Route) printRoutes(prefix string) {
// 	fmt.Printf("%s%s with (%d) paths, children: (%d)  \n", prefix, r.base, len(r.paths), len(r.children))

// 	for _, child := range r.children {
// 		//fmt.Printf("%s%s\n", prefix, child.base)
// 		child.printRoutes(prefix)
// 	}
// }

// Router is equivalent of app.route(path), returns a new instance of Route


func (r *route) combineHandlers(handlers ...Handler) []Handler {
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

func (a *App) newRoute(path string, isRoot bool, parent *route) *route {
	var r route
	cleanPath := a.makePath(path)

	if isRoot {
		r = route{
			base:     cleanPath,
			hr:       a.router,
			app:      a,
			rootNode: true,
			parent:   nil,
			children: make(map[string]*route),
		}
		a.route = &r
	} else {
		combinedPath := filepath.Join(parent.base, cleanPath)
		r = route{
			base:     combinedPath,
			hr:       a.router,
			app:      a,
			parent:   parent,
			children: make(map[string]*route),
		}
		parent.children[combinedPath] = &r
	}

	return &r
}

func (r *route) getFullPath(path string) string {
	var builder strings.Builder

	builder.WriteString(strings.TrimSuffix(r.base, "/"))
	if len(path) > 0 && path[0] != '/' {
		builder.WriteRune('/')
	}
	builder.WriteString(path)

	return builder.String()
}

func (r *route) fetchMiddleware() []Handler {
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

func (r *route) handle(httpMethod string, path string, handlers []Handler) {
	newPath := routePath{
		name:     r.getFullPath(path),
		handlers: handlers,
		method:   httpMethod,
	}

	if r.paths == nil {
		r.paths = make([]routePath, 0)
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

func (r *route) Path() string {
	return r.base
}

func (r *route) NewRouter(path string) *route {
	return r.app.newRoute(path, false, r)
}

func (r *route) Use(middleware ...Handler) *route {
	r.middleware = append(r.middleware, middleware...)
	return r
}

func (r *route) Get(path string, handlers ...Handler) *route {
	r.handle("GET", path, handlers)
	return r
}

func (r *route) Post(path string, handlers ...Handler) *route {
	r.handle("POST", path, handlers)
	return r
}

func (r *route) Put(path string, handlers ...Handler) *route {
	r.handle("PUT", path, handlers)
	return r
}

func (r *route) Delete(path string, handlers ...Handler) *route {
	r.handle("DELETE", path, handlers)
	return r
}

func (r *route) Patch(path string, handlers ...Handler) *route {
	r.handle("PATCH", path, handlers)
	return r
}

func (r *route) Options(path string, handlers ...Handler) *route {
	r.handle("OPTIONS", path, handlers)
	return r
}

func (r *route) Head(path string, handlers ...Handler) *route {
	r.handle("HEAD", path, handlers)
	return r
}

func (r *route) All(path string, handlers ...Handler) *route {
	for _, v := range methods {
		r.handle(v, path, handlers)
	}
	return r
}

func (r *route) Static(root fs.FS, path string) *route {
	strippedPath := "/" + strings.Trim(path, "/")
	fileServer := http.FileServer(http.FS(root))

	r.hr.GET(strippedPath+"/*filepath", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		req.URL.Path = ps.ByName("filepath")
		fileServer.ServeHTTP(w, req)
	})

	return r
}

func (r *route) Param(param string, handler ParamHandler) *route {

	if r.paramHandlers == nil {
		r.paramHandlers = make(map[string]ParamHandler)
	}
	r.paramHandlers[param] = handler

	return r
}
