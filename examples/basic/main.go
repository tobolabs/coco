package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/tobolabs/coco"
)

func main() {
	app := coco.NewApp()

	app.Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
		fmt.Printf("Param Middleware: %s\n", param)
		next(res, req)
	})

	app.Post("/post", func(res coco.Response, req *coco.Request, next coco.NextFunc) {

		data, err := req.Body.FormData()
		if err != nil {
			res.Send(err.Error())
			return
		}
		res.JSON(data)
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
	app.SetSetting("x-powered-by", false)

	// Application.Disabled ✅
	fmt.Printf("Disabled: %v\n", app.IsSettingDisabled("x-powered-by"))

	app.Get("/chekme", func(res coco.Response, req *coco.Request, next coco.NextFunc) {

		res.Cookie(&http.Cookie{
			Name:  "greeting",
			Value: "Ser",
		})
		res.Send("Checkme")
	})

	app.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {

		isJson := req.Is("json/*")
		fmt.Printf("isJson: %v\n", isJson)

		isText := req.Is("text/*")
		fmt.Printf("isText: %v\n", isText)

		isHtml := req.Is("html")
		fmt.Printf("isHtml: %v\n", isHtml)

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

	userRouter := app.NewRouter("users")

	userRouter.Param("id", func(res coco.Response, req *coco.Request, next coco.NextFunc, param string) {
		log.Println("User Param Middleware")
		next(res, req)
	})

	userRouter.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		fmt.Printf("req.BaseUrl : %s\n", req.BaseURL)
		res.Send("Hello User")
	})

	userRouter.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		log.Println("User Middleware 1")
		next(res, req)
	})

	userRouter.Get("hello/:id", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		res.Send(fmt.Sprintf("Hello %s 👋", req.Params["id"]))
	})

	profileRouter := userRouter.NewRouter("profile")

	profileRouter.Use(func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		log.Println("Profile Middleware 1")
		next(res, req)
	})

	profileRouter.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		fmt.Printf("req.BaseUrl : %s\n", req.BaseURL)
		res.Send("Hello User Profile")
	})

	userRouter.Get("/profile/settings", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		fmt.Println(req.Path)
		fmt.Printf("req.BaseUrl : %s\n", req.BaseURL)
		res.Send("Hello User")
	})

	socialRouter := userRouter.NewRouter("social")

	socialRouter.Get("/", func(res coco.Response, req *coco.Request, next coco.NextFunc) {
		fmt.Printf("req.BaseUrl : %s\n", req.BaseURL)
		res.Send("Hello User Social")

	})

	srv := &http.Server{
		Addr:    ":3003",
		Handler: app,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
	//
	//if err := app.Listen(":3003"); err != nil {
	//	log.Fatal(err.Error())
	//}
}
