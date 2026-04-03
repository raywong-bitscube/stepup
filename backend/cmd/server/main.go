package main

import (
	"log"

	"github.com/raywong-bitscube/stepup/backend/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
