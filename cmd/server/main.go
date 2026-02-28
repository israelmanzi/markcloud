package main

import (
	"log"
	"net/http"
	"os"

	"github.com/israelmanzi/markcloud/internal/store"
	"github.com/israelmanzi/markcloud/internal/web"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbPath := envOr("DB_PATH", "markcloud.db")
	apiKey := mustEnv("API_KEY")
	deploySecret := mustEnv("DEPLOY_SECRET")
	addr := envOr("ADDR", ":8080")
	templatesDir := envOr("TEMPLATES_DIR", "templates")

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	srv := web.NewServer(web.Config{
		Store:        s,
		APIKey:       apiKey,
		DeploySecret: deploySecret,
		TemplatesDir: templatesDir,
	})

	log.Printf("markcloud listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s not set", key)
	}
	return v
}
