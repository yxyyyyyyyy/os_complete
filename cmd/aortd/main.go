package main

import (
	"flag"
	"log"
	"net/http"

	"aort-r/internal/api"
	"aort-r/internal/config"
)

func main() {
	configPath := flag.String("config", "configs/dev.yaml", "path to AORT-R config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("aortd listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, api.NewServer(cfg)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
