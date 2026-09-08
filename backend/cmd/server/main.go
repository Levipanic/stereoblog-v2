package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}

	if err := gin.New().Run(cfg.ListenAddress()); err != nil {
		log.Fatal(err)
	}
}
