package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/tobolabs/coco/v2"
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

	staticFiles, err := fs.Sub(assets, "web/assets")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}

	// Serve static assets on /static; e.g. /static/css/main.css maps to web/assets/css/main.css
	app.Static(staticFiles, "/static")

	app.Get("/", func(rw coco.Response, req *coco.Request, next coco.NextFunc) {
		rw.Render("views/home", nil)
	})

	if err := app.Listen(":8980"); err != nil {
		log.Fatal(err)
	}
}
