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
	base      *httprouter.Router
	basePath  string
	*Route    // default Route
	routes    map[string]*Route
	templates map[string]*template.Template
	*settings
}

// NewApp creates a new App instance with a default Route at the root path "/"
// and a default settings instance with default values.
// Equivalent to:
//
// const app = express()
func NewApp() (app *App) {
	app = &App{
		basePath: "",
		routes:   make(map[string]*Route),
		base:     httprouter.New(),
	}

	app.Route = app.newRoute(app.basePath)
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
	go func() {
		<-ctx.Done()
		fmt.Println("shutting down server")
		server.Shutdown(ctx)
	}()

	return server.ListenAndServe()
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.base.ServeHTTP(w, r)
}
