package coco

import (
	"html/template"
	"net/http"
)

type reqcontext struct {
	handlers  []Handler
	templates map[string]*template.Template
	req       *Request
	app       *App
	accepted  []string
}

func (c *reqcontext) coco() *App {
	return c.app
}

func (c *reqcontext) request() *http.Request {
	return c.req.r
}

// next calls the next handler in the chain if there is one.
// If there is no next handler, the request is terminated.
func (c *reqcontext) next(rw Response, req *Request) {
	if len(c.handlers) == 0 {
		http.NotFound(rw.w, c.request())
		return
	}

	// Take the first handler off the list and call it.
	h := c.handlers[0]
	c.handlers = c.handlers[1:]
	h(rw, req, c.next)
}
