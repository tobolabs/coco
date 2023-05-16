package main

import (
	"context"
	"fmt"
	"log"

	"github.com/noelukwa/coco"
)

func main() {
	app := coco.NewApp()

	app.Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
		log.Println("Param Middleware")
		next(res, req)
	})

	app.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		log.Println("Middleware 0")
		next(res, req)
	})

	app.Get("hello/:id", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send(fmt.Sprintf("Hello %s 👋", req.Params["id"]))
	})

	app.Post("hello/:id", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send(fmt.Sprintf("Hello %s 👋", req.Params["id"]))
	})

	app.Get("hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello World")
	})

	app.Post("hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello World Post")
	})

	userRouter := app.Router("users")

	userRouter.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		log.Println("Middleware 1")
		next(res, req)
	})

	userRouter.Get("hello", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello User")
	})

	userRouter.Get("/profile/settings", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		fmt.Println(req.Path)
		res.Send("Hello User")
	})

	if err := app.Listen(":3003", context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}
