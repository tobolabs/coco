# Coco Web Framework for Go

Coco is a lightweight web framework for Go designed to simplify web application development in go and make it more enjoyable.

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

## Getting Started

1. **Installation**:
   To use Coco in your Go project, you need to install it using Go modules:

   ```shell
   go get github.com/tobolabs/coco
   ```

2. **Creating a Coco App**:
   Import the `coco` package and create a Coco application:

   ```go
   package main
   
   import (
       "context"
       "github.com/tobolabs/coco"
   )

   func main() {
       app := coco.NewApp()
       
       // Define routes and handlers here
       
       // Start the server
       app.Listen(":8080", context.Background())
   }
   ```

3. **Routing**:
   Define your routes and handlers using the `Get`, `Post`, and other methods provided by Coco. Here's an example:

   ```go
   app.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
       res.Send("Hello, Coco!")
   })
   ```

4. **Middleware**:
   Apply middleware functions to your routes to add pre or post-processing logic to your requests. Middleware functions are executed in the order they are defined.

   ```go
   app.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
       // Add middleware logic here
       next(res, req)
   })
   ```

5. **Request and Response Handling**:
   Access request and response information within your handlers using the `Request` and `Response` objects. For example, get query parameters, set headers, or send responses:

   ```go
   app.Get("/user", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
       username := req.Query["username"]
       res.JSON(map[string]string{"message": "Hello, " + username})
   })
   ```

6. **Template Rendering**:
   Coco supports HTML template rendering. You can render dynamic HTML pages using Go templates:

   ```go
   app.Get("/about", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
       data := struct {
           Title   string
           Content string
       }{
           Title:   "About Us",
           Content: "We are Coco, a lightweight web framework for Go.",
       }
       res.Render("about", data)
   })
   ```
   
**See [examples](examples) for more.**

## Configuration

Coco allows you to configure settings for your application. Common configurations include enabling trust proxy, specifying the environment mode, and more. You can use these settings to customize your application behavior.

```go
app.Enable("trust proxy")
app.SetX("env", "production")
```

## Error Handling

Coco provides a structured way to handle errors, including error codes and descriptive error messages. Errors are returned as `JSONError` and can be handled uniformly in your application.

```go
err := app.Listen(":8080")
if err != nil {
    log.Fatalf("Server failed to start: %v", err)
}
```

## Contributing

Coco is an open-source project, and we welcome contributions from the community. If you'd like to contribute, please check the [Contribution Guidelines](CONTRIBUTING.md).

## License

Coco is licensed under the [MIT License](LICENSE).

## Getting Help

If you encounter issues or have questions about Coco, please check the [GitHub Issues](https://github.com/tobolabs/coco/issues) for known problems or open a new issue.

## Roadmap

Check the [GitHub repository](https://github.com/tobolabs/coco) for the latest updates and the roadmap for Coco.

## Acknowledgments

Coco is inspired by [Express](https://expressjs.com/), a popular web framework for Node.js.
At the core of Coco is httprouter, a fast HTTP router for Go by [Julien Schmidt](https://github.com/julienschmidt).
## Authors

- [Noel Ukwa](https://github.com/noelukwa)

Happy coding with Coco! 🌴🚀

[GitHub Repository](https://github.com/tobolabs/coco)

[License](LICENSE)

**Coco Web Framework for Go**