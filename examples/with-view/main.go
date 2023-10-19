package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/tobolabs/coco"
)

var (
	//go:embed views
	views embed.FS
)

func main() {
	// create a new app and set the global prefix
	app := coco.NewApp()

	if err := app.LoadTemplates(views, nil); err != nil {
		fmt.Printf("error loading templates: %v", err)
	}

	// named param
	app.Get("/", func(rw coco.Response, r *coco.Request, next coco.NextFunc) {
		if err := rw.Render("index", nil); err != nil {
			fmt.Printf("error rendering template: %v", err)
		}
	})

	app.Get("/health", func(rw coco.Response, req *coco.Request, next coco.NextFunc) {
		rw.JSON(map[string]string{"status": "ok"})
	})

	app.Get("/about", func(rw coco.Response, req *coco.Request, next coco.NextFunc) {
		rw.Render("about", nil)
	})

	app.Get("/dash", func(rw coco.Response, req *coco.Request, next coco.NextFunc) {
		rw.Render("dash/index", nil)
	})

	if err := app.Listen(":8040", context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("closed")
}
