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

	// Application.All ✅
	app.All("/generic", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Generic")
	})

	// Application.Delete ✅
	app.Delete("/delete", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Delete")
	})

	// Application.Disable ✅
	app.Disable("x-powered-by")

	// Application.Disabled ✅
	fmt.Printf("Disabled: %v\n", app.Disabled("x-powered-by"))

	app.Get("/chekme", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Checkme")
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

	userRouter.Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
		log.Println("User Param Middleware")
		next(res, req)
	})

	userRouter.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello User")
	})

	userRouter.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		log.Println("User Middleware 1")
		next(res, req)
	})

	userRouter.Get("hello/:id", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send(fmt.Sprintf("Hello %s 👋", req.Params["id"]))
	})

	profileRouter := userRouter.Router("profile")

	profileRouter.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		log.Println("Profile Middleware 1")
		next(res, req)
	})

	profileRouter.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello User Profile")
	})

	userRouter.Get("/profile/settings", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		fmt.Println(req.Path)
		res.Send("Hello User")
	})

	socialRouter := userRouter.Router("social")

	socialRouter.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send("Hello User Social")

	})

	if err := app.Listen(":3003", context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}
