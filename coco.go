package coco

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// NextFunc is a function that is called to pass execution to the next handler
// in the chain.
// Equivalent to: next() in express
// See: https://expressjs.com/en/guide/writing-middleware.html
type NextFunc func(res Response, r *Request)

// Handler Handle is a function that is called when a request is made to the Route.
// Equivalent to:  (req, res) => { ... } in express
// See: https://expressjs.com/en/guide/routing.html
type Handler func(res Response, req *Request, next NextFunc)

type ParamHandler func(res Response, req *Request, next NextFunc, param string)

// App is the main type for the coco framework.
// It is the equivalent of what is returned in the express() function in express.
// See: https://expressjs.com/en/4x/api.html#express
type App struct {
	base     *httprouter.Router
	basePath string
	*Route   // default Route

	templates map[string]*template.Template
	settings  map[string]interface{}
}

func defaultSettings() map[string]interface{} {
	return map[string]interface{}{
		"x-powered-by":     true,
		"env":              "development",
		"etag":             "weak",
		"trust proxy":      false,
		"subdomain offset": 2,
	}
}

// NewApp creates a new App instance with a default Route at the root path "/"
// and a default settings instance with default values.
// Equivalent to:
//
// const app = express()
func NewApp() (app *App) {

	app = &App{
		basePath: "",
		base:     httprouter.New(),
		settings: defaultSettings(),
	}

	app.Route = app.newRoute(app.basePath, true, nil)
	return
}

// Listen starts an HTTP server and listens on the given address. It returns an
// error if the server fails to start, or if the context is cancelled.
// Equivalent to:
//
// app.listen(3000, () => {})
func (a *App) Listen(addr string, ctx context.Context) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a,
	}

	// Configure routes before starting the server
	a.configureRoutes()

	return server.ListenAndServe()
}

// configureRoutes method attaches the routes to their relevant handlers and middleware
func (a *App) configureRoutes() {
	a.traverseAndConfigure(a.Route)
}

func (a *App) traverseAndConfigure(r *Route) {
	if len(r.paths) > 0 {
		for idx := range r.paths {
			path := &r.paths[idx]
			handlers := r.combineHandlers(path.handlers...)

			r.hr.Handle(path.method, path.name, func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
				request, e := newRequest(req, w, p, a)
				if e != nil {
					fmt.Printf("DEBUG: %v\n", e)
				}
				accepts := parseAccept(req.Header.Get("Accept"))
				ctx := &reqcontext{
					handlers:  handlers,
					templates: r.app.templates,
					req:       request,
					accepted:  accepts,
				}
				response := Response{w, ctx, 0}
				execParamChain(ctx, p, r.paramHandlers)
				ctx.next(response, request)
			})
		}
	}

	for _, child := range r.children {
		a.traverseAndConfigure(child)
	}
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.base.ServeHTTP(w, r)
}

func (a *App) Disable(key string) {
	a.settings[key] = false
}

func (a *App) Enable(key string) {
	a.settings[key] = true
}

func (a *App) SetX(key string, value interface{}) {
	a.settings[key] = value
}

func (a *App) GetX(key string) interface{} {
	return a.settings[key]
}

func (a *App) Disabled(key string) bool {
	value, ok := a.settings[key].(bool)
	return ok && !value
}

func (a *App) Enabled(key string) bool {
	value, ok := a.settings[key].(bool)
	return ok && value
}
