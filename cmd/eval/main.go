package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/eval"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func main() {
	configPath := flag.String("config", "", "path to config.yaml")
	limit := flag.Int("limit", 200, "max tasks to score")
	userID := flag.String("user", "", "optional user id to scope tasks")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	report, err := eval.Run(context.Background(), st, *limit, *userID)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
}
