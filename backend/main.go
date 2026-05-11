package main

import (
	"fmt"
	"log"

	"github.com/philippgehrig/asimuth-automation/backend/config"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting asimut-automation on port %s", cfg.Port)
	fmt.Println("Server not yet implemented")
}
