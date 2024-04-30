package coco

import (
	coreCtx "context"
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

// Handler Handle is a function that is called when a request is made to the route.
type Handler func(res Response, req *Request, next NextFunc)

// ParamHandler is a function that is called when a parameter is found in the route path
type ParamHandler func(res Response, req *Request, next NextFunc, param string)

// App is the main type for the coco framework.
type App struct {
	router      *httprouter.Router
	httpHandler http.Handler
	basePath    string
	*route
	httpServer    *http.Server
	templates     map[string]*template.Template
	settings      map[string]interface{}
	once          sync.Once
	settingsMutex sync.RWMutex
}

// Settings returns the settings instance for the App.
func (a *App) Settings() map[string]interface{} {
	a.settingsMutex.RLock()
	defer a.settingsMutex.RUnlock()
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
func NewApp() (app *App) {

	app = &App{
		basePath: "",
		router:   httprouter.New(),
		httpHandler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Handler not configured", http.StatusInternalServerError)
		}),
		settings: defaultSettings(),
	}

	app.route = app.newRoute(app.basePath, true, nil)
	return
}

// Listen starts an HTTP server and listens on the given address.
// addr should be in format :PORT ie :8000
func (a *App) Listen(addr string) error {
	a.httpServer = &http.Server{
		Addr:    addr,
		Handler: a,
	}
	return a.httpServer.ListenAndServe()
}

func (a *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	a.once.Do(func() {
		a.configureRoutes()
		a.httpHandler = a.router
	})
	a.httpHandler.ServeHTTP(w, req)
}

// Close stops the server gracefully and returns any encountered error.
func (a *App) Close() error {
	if a.httpServer == nil {
		return nil
	}
	shutdownTimeout := 5 * time.Second
	shutdownCtx, cancel := coreCtx.WithTimeout(coreCtx.Background(), shutdownTimeout)
	defer cancel()
	err := a.httpServer.Shutdown(shutdownCtx)
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
	a.traverseAndConfigure(a.route)
}

func (a *App) traverseAndConfigure(r *route) {
	for _, path := range r.paths {
		handlers := r.combineHandlers(path.handlers...)
		r.hr.Handle(path.method, path.name, func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
			request, err := newRequest(req, w, p, a)
			if err != nil {
				fmt.Printf("DEBUG: %v\n", err)
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

// SetSetting sets a custom setting with a key and value.
func (a *App) SetSetting(key string, value interface{}) {
	a.settingsMutex.Lock()
	defer a.settingsMutex.Unlock()
	a.settings[key] = value
}

// GetSetting retrieves a custom setting by its key.
func (a *App) GetSetting(key string) interface{} {
	a.settingsMutex.RLock()
	defer a.settingsMutex.RUnlock()
	return a.settings[key]
}

// IsSettingEnabled checks if a setting is true.
func (a *App) IsSettingEnabled(key string) bool {
	a.settingsMutex.RLock()
	defer a.settingsMutex.RUnlock()
	value, ok := a.settings[key].(bool)
	return ok && value
}

// IsSettingDisabled checks if a setting is false.
func (a *App) IsSettingDisabled(key string) bool {
	a.settingsMutex.RLock()
	defer a.settingsMutex.RUnlock()
	value, ok := a.settings[key].(bool)
	return ok && !value
}
