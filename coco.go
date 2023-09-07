package coco

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

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
	settings  *settings
}

type settings struct {
	xPoweredBy      bool
	env             string
	etag            string
	trustProxy      bool
	subDomainOffset int

	custom map[string]interface{}
}

var defaultKeys = map[string]interface{}{
	"x-powered-by":     true,
	"etag":             "weak",
	"trust proxy":      false,
	"subdomain offset": 2,
	"env":              "development",
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
	a.configureRoutes()
	go func() {
		<-ctx.Done()
		fmt.Println("shutting down server")
		server.Shutdown(ctx)
	}()

	return server.ListenAndServe()
}

// configureRoutes method attaches the routes to their relevant handlers and middleware
func (a *App) configureRoutes() {
	a.printRoutes("root")
	var routes []*Route
	var transverse func(r *Route)
	transverse = func(r *Route) {
		routes = append(routes, r)
		for _, child := range r.children {
			transverse(child)
		}
	}
	transverse(a.Route)
	
	for _, route := range routes {
		for idx := range route.paths {
			path := &route.paths[idx]
			handlers := route.combineHandlers(path.handlers...)
			route.hr.Handle(path.method, route.getfullPath(path.name), func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
				request := newRequest(req, w, p)
				accepts := parseAccept(req.Header.Get("Accept"))
				ctx := &reqcontext{
					handlers:  handlers,
					templates: route.app.templates,
					req:       &request,
					accepted:  accepts,
				}

				response := Response{w, ctx, 0}
				execParamChain(ctx, p, route.paramHandlers)
				ctx.next(response, &request)
			})

		}

	}
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.base.ServeHTTP(w, r)
}

func (a *App) getBool(key string) bool {

	//check the default keys for any key with a boolean value
	if _, ok := defaultKeys[key]; ok {
		switch key {
		case "x-powered-by":
			return a.settings.xPoweredBy
		case "trust proxy":
			return a.settings.trustProxy

		}
	} else if v, ok := a.settings.custom[key]; ok {
		vOk, e := strconv.ParseBool(fmt.Sprintf("%v", v))
		if e != nil {
			return false
		}
		return vOk
	}

	return false
}

func (a *App) SetX(key string, val interface{}) {
	switch key {
	case "x-powered-by":
		a.settings.xPoweredBy = val.(bool)
	case "etag":
		a.settings.etag = val.(string)
	case "trust proxy":
		a.settings.trustProxy = val.(bool)
	case "subdomain offset":
		a.settings.subDomainOffset = val.(int)
	case "env":
		a.settings.env = val.(string)
	default:
		a.settings.custom[key] = val
	}
}

func (a *App) Disable(key string) {
	if v, ok := a.settings.custom[key]; ok {
		a.settings.custom[key] = !v.(bool)
		return
	} else if _, ok := defaultKeys[key]; ok {
		switch key {
		case "x-powered-by":
			a.settings.xPoweredBy = false
		case "etag":
			a.settings.etag = "none"
		case "trust proxy":
			a.settings.trustProxy = false
		}
	}
}

func (a *App) Enable(key string) {
	if v, ok := a.settings.custom[key]; ok {
		a.settings.custom[key] = v.(bool)
		return
	} else if _, ok := defaultKeys[key]; ok {
		switch key {
		case "x-powered-by":
			a.settings.xPoweredBy = true
		case "etag":
			a.settings.etag = "weak"
		case "trust proxy":
			fmt.Println("enabled trust proxy")
			a.settings.trustProxy = true
		}
	}
}

func (a *App) GetX(key string) interface{} {

	if v, ok := a.settings.custom[key]; ok {
		return v
	} else if _, ok := defaultKeys[key]; ok {
		switch key {
		case "x-powered-by":
			return a.settings.xPoweredBy
		case "etag":
			return a.settings.etag
		case "trust proxy":
			return a.settings.trustProxy
		case "subdomain offset":
			return a.settings.subDomainOffset
		case "env":
			return a.settings.env

		}
	}
	return nil
}

func (a *App) Disabled(key string) bool {
	return !a.getBool(key)
}

func (a *App) Enabled(key string) bool {
	return a.getBool(key)
}

func defaultSettings() *settings {
	return &settings{
		xPoweredBy:      true,
		env:             "development",
		etag:            "weak",
		trustProxy:      false,
		subDomainOffset: 2,
		custom:          make(map[string]interface{}),
	}
}
