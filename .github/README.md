**coco** is a lightweight web framework inspired by [express js](https://github.com/expressjs/express) and designed to simplify development in go and make it more enjoyable.

## Features

- **Routing**: Easily define and manage HTTP routes using an expressive routing mechanism.
- **Middleware**: Create and apply middleware functions to process requests and responses at various stages of your application.
- **Template Rendering**: Coco supports HTML template rendering, allowing you to generate dynamic HTML pages effortlessly.
- **Customizable Settings**: Configure settings for your application, such as trust proxy, environment mode, and more.
- **Error Handling**: Handle errors and exceptions in a clean and structured manner.
- **Request and Response**: Access detailed information about incoming requests and control outgoing responses.
- **JSON and Form Data Parsing**: Simplify JSON and form data parsing with built-in functionality.
- **HTTP Range Handling**: Conveniently process HTTP ranges for serving partial content.
- **Accept Header Parsing**: Determine the best response format based on the incoming request's Accept header.

## Quick Start ✨

```go
package main

import (
   "github.com/tobolabs/coco"
   "log"
)

func main() {
   app := coco.NewApp()

   app.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
      next(res, req)
      log.Printf("%s %s %d", req.Method, req.Path, res.StatusCode)
   })
   

   app.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
      res.Send("hello, coco!")
   })
   
   bookRoute := app.Router("/books")
   
    bookRoute.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
        res.Send("books")
    })
   
    app.Get("/user", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
       username := req.Query["username"]
       res.JSON(map[string]string{"message": "Hello, " + username})
    })
   
   if err := app.Listen(":8080"); err != nil {
      log.Fatalf("Server failed to start: %v", err)
   }
}
```

## **Installation**

   To use coco in your Go project, you need to install it using Go modules:

   ```shell
   go get github.com/tobolabs/coco
   ```

**See [examples](../examples) for more.**

## Configuration

coco allows you to configure settings for your application. Common configurations include enabling trust proxy, specifying the environment mode, and more. You can use these settings to customize your application behavior.

```go
app.Enable("trust proxy")
app.SetX("env", "production")
```

## Error Handling

coco provides a structured way to handle errors, including error codes and descriptive error messages. Errors are returned as `JSONError` and can be handled uniformly in your application.

```go
err := app.Listen(":8080")
if err != nil {
    log.Fatalf("Server failed to start: %v", err)
}
```

## Contributing

coco is an open-source project, and we welcome contributions from the community. If you'd like to contribute, please check the [Contribution Guidelines](CONTRIBUTING.md).

## License

coco is licensed under the [MIT License](../LICENSE).

## Getting Help

If you encounter issues or have questions about coco, please check the [GitHub Issues](https://github.com/tobolabs/coco/issues) for known problems or open a new issue.

## Roadmap

Check the [GitHub repository](https://github.com/tobolabs/coco) for the latest updates and the roadmap for Coco.

## Acknowledgments

coco is inspired by [Express](https://expressjs.com/), a popular web framework for Node.js.
At the core of Coco is httprouter, a fast HTTP router for Go by [Julien Schmidt](https://github.com/julienschmidt).

## Author

- [Noel Ukwa](https://github.com/noelukwa)

Happy coding with 🌴🚀

[License](../LICENSE)
