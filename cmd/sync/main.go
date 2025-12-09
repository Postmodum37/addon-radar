package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("addon-radar sync starting...")

	apiKey := os.Getenv("CURSEFORGE_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: CURSEFORGE_API_KEY not set")
		os.Exit(1)
	}

	fmt.Println("API key found, sync would run here")
}
