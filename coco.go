package coco

import (
	coreContext "context"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

// NextFunc is a function that is called to pass execution to the next handler
// in the chain.
type NextFunc func(res Response, r *Request)

// Handler Handle is a function that is called when a request is made to the Route.
type Handler func(res Response, req *Request, next NextFunc)

// ParamHandler is a function that is called when a parameter is found in the
// Route path.
type ParamHandler func(res Response, req *Request, next NextFunc, param string)

// App is the main type for the coco framework.
type App struct {
	base      *httprouter.Router
	handler   http.Handler
	basePath  string
	*Route    // default Route
	server    *http.Server
	templates map[string]*template.Template
	settings  map[string]interface{}
	once      sync.Once
}

// Settings returns the settings instance for the App.
func (a *App) Settings() map[string]interface{} {
	return a.settings
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
		handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Handler not configured", http.StatusInternalServerError)
		}),
		settings: defaultSettings(),
	}

	app.Route = app.newRoute(app.basePath, true, nil)
	return
}

// Listen starts an HTTP server and listens on the given address.
// Equivalent to:
//
// app.listen(3000, () => {})
func (a *App) Listen(addr string) error {
	a.server = &http.Server{
		Addr:    addr,
		Handler: a,
	}

	return a.server.ListenAndServe()
}

func (a *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	a.once.Do(func() {
		a.configureRoutes()
		a.handler = a.base
	})
	a.handler.ServeHTTP(w, req)
}

// Close stops the server gracefully and returns any encountered error.
func (a *App) Close() error {
	if a.server == nil {

		return nil
	}
	shutdownTimeout := 5 * time.Second
	shutdownCtx, cancel := coreContext.WithTimeout(coreContext.Background(), shutdownTimeout)
	defer cancel()

	err := a.server.Shutdown(shutdownCtx)
	if err != nil {
		if err == http.ErrServerClosed {

			return nil
		}
		return fmt.Errorf("error during server shutdown: %w", err)
	}

	return nil
}

// configureRoutes method attaches the routes to their relevant handlers and middleware
func (a *App) configureRoutes() {
	//a.printRoutes("")
	a.traverseAndConfigure(a.Route)
}

// TODO: fix this
func (a *App) traverseAndConfigure(r *Route) {
	for _, path := range r.paths {
		handlers := r.combineHandlers(path.handlers...)

		r.hr.Handle(path.method, path.name, func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
			request, e := newRequest(req, w, p, a)
			if e != nil {
				fmt.Printf("DEBUG: %v\n", e)
			}
			ctx := &context{
				handlers:  handlers,
				templates: r.app.templates,
				req:       request,
			}
			response := Response{ww: wrapWriter(w), ctx: ctx}
			execParamChain(ctx, p, r.paramHandlers)
			ctx.next(response, request)
		})
	}

	for _, child := range r.children {
		a.traverseAndConfigure(child)
	}
}

// Disable sets a setting to false.
func (a *App) Disable(key string) {
	a.settings[key] = false
}

// Enable sets a setting to true.
func (a *App) Enable(key string) {
	a.settings[key] = true
}

// SetX sets a custom setting with a key and value.
func (a *App) SetX(key string, value interface{}) {
	a.settings[key] = value
}

// GetX retrieves a custom setting by its key.
func (a *App) GetX(key string) interface{} {
	return a.settings[key]
}

// Disabled checks if a setting is false.
func (a *App) Disabled(key string) bool {
	value, ok := a.settings[key].(bool)
	return ok && !value
}

// Enabled checks if a setting is true.
func (a *App) Enabled(key string) bool {
	value, ok := a.settings[key].(bool)
	return ok && value
}
