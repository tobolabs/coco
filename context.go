package coco

import (
	"html/template"
	"net/http"
)

type context struct {
	handlers  []Handler
	templates map[string]*template.Template
	req       *Request
	app       *App
}

func (c *context) coco() *App {
	return c.app
}

func (c *context) request() *http.Request {
	return c.req.r
}

// next calls the next handler in the chain if there is one.
// If there is no next handler, the request is terminated.
func (c *context) next(rw Response, req *Request) {
	if len(c.handlers) == 0 {
		http.NotFound(rw.ww, req.r)
		return
	}
	h := c.handlers[0]
	c.handlers = c.handlers[1:]
	h(rw, req, c.next)
}
