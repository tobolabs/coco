package main

import (
	"context"
	"embed"
	"log"

	"github.com/noelukwa/coco"
)

var (
	//go:embed web/views
	views embed.FS

	//go:embed web/assets
	assets embed.FS
)

func main() {

	app := coco.NewApp()

	if err := app.LoadTemplates(views, nil); err != nil {
		log.Fatal(err)
	}

	app.Static(assets, "web/*file")

	app.Get("/", func(rw coco.Response, req *coco.Request, next coco.NextFunc) {
		rw.Render("views/home", nil)
	})

	if err := app.Listen(":8980", context.Background()); err != nil {
		log.Fatal(err)
	}
}
