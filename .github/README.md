
# coco

coco is a lightweight, flexible web framework for Go that provides a simple yet powerful API for building web applications.

[![Go Report Card](https://goreportcard.com/badge/github.com/tobolabs/coco?style=flat-square)](https://goreportcard.com/report/github.com/tobolabs/coco)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/tobolabs/coco)
[![Release](https://github.com/looterz/grimd/actions/workflows/release.yaml/badge.svg)](https://github.com/tobolabs/coco/releases)

## Getting Started

To get started with coco, you need to have Go installed on your machine. In your go module run:

```bash
go get github.com/tobolabs/coco
```

## Features 🚀

- 🛣️ Dynamic and static routing with `httprouter`.
- 📦 Middleware support for flexible request handling.
- 📑 Template rendering with custom layout configurations.
- 🛠️ Simple API for managing HTTP headers, cookies, and responses.
- 🔄 Graceful shutdown for maintaining service integrity.
- 🔄 JSON and form data handling.

## Basic Example

```go
package main

import (
    "github.com/tobolabs/coco"
    "net/http"
)

func main() {
    app := coco.NewApp()
    app.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
        res.Send("Welcome to coco!")
    })
    app.Listen(":8080")
}

```

## API Overview

### Routing

Define routes easily with methods corresponding to HTTP verbs:

```go
app.Get("/users", getUsers)
app.Post("/users", createUser)
app.Put("/users/:id", updateUser)
app.Delete("/users/:id", deleteUser)

```

### Parameter Routing

```go
 app.Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
  // runs for any route with a param named :id
  next(res, req)
 })
```

### Middleware

```go
app.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
    // Log, authenticate, etc.
    next(res, req)
})

```

### Dynamic URL Parameters

```go
app.Get("/greet/:name", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
    name := req.GetParam("name")
    res.Send("Hello, " + name + "!")
})
```

### Settings and Custom Configuration

```go
app.SetSetting("x-powered-by", false)
isEnabled := app.IsSettingEnabled("x-powered-by")
```

### Responses

```go
app.Get("/data", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
    data := map[string]interface{}{"key": "value"}
    res.JSON(data)
})
```

### Templates

```go
var (
 //go:embed views
 views embed.FS
)

app.LoadTemplates(views, nil)
app.Get("/", func(rw coco.Response, r *coco.Request, next coco.NextFunc) {
  rw.Render("index", map[string]interface{}{"title": "Home"})
 })
```

### Static Files

```go
app.Static(http.Dir("public"), "/static")
```

## Acknowledgments

coco is inspired by [Express](https://expressjs.com/), a popular web framework for Node.js.
At the core of coco is httprouter, a fast HTTP router by [Julien Schmidt](https://github.com/julienschmidt).

## Author

- [Uche Ukwa](https://github.com/noelukwa)
