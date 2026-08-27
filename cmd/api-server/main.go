package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/luck/permission-center-go/internal/app"
	"github.com/luck/permission-center-go/internal/config"
	httpserver "github.com/luck/permission-center-go/internal/server/http"
)

func main() {
	configPath := os.Getenv("PERMISSION_CENTER_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/app.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}
	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()
	server := &http.Server{
		Addr: cfg.HTTP.Addr,
		Handler: httpserver.NewServer(
			httpserver.NewHandler(application.Permissions, application.Auth),
			application.Auth.Middleware,
		),
	}
	log.Printf("permission center HTTP server listening on %s", cfg.HTTP.Addr)
	log.Fatal(server.ListenAndServe())
}
