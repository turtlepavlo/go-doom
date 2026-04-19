package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/converters"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/wad"
)

func main() {
	var iwadPath string
	flag.StringVar(&iwadPath, "iwad", "", "path to IWAD/PWAD file")
	flag.Parse()

	iwadPath = strings.TrimSpace(iwadPath)
	if iwadPath == "" {
		fmt.Fprintln(os.Stderr, "usage: doom -iwad <path-to-wad>")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reader := wad.NewBinaryReader()
	converter := converters.NewWADConverter()
	useCase, err := loadiwad.New(reader, converter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap load IWAD use case: %v\n", err)
		os.Exit(1)
	}

	archive, err := useCase.Execute(ctx, iwadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load WAD: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %s with %d lumps and %d map markers\n", archive.Header.Kind, len(archive.Lumps), len(archive.Maps))
	for i, gameMap := range archive.Maps {
		if i >= 5 {
			fmt.Printf("... and %d more maps\n", len(archive.Maps)-5)
			break
		}
		fmt.Printf("  %s: %d lumps\n", gameMap.Name, len(gameMap.Lumps))
	}
}
