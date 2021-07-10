package main

import (
	"context"
	"fmt"
	"github.com/bonnetn/filesharing/internal"
	"log"
)

func app(ctx context.Context) error {
	agents, err := internal.FetchUserAgentDatabase()
	if err != nil {
		return fmt.Errorf("could not fetch user agent database: %w", err)
	}

	var (
		repository = internal.NewPendingFileshareRepository()
		get        = internal.NewGetOperation(repository)
		post       = internal.NewCreateOperation(repository)
		handler    = internal.NewHandler(get, post, agents)
	)
	return internal.RunServer(ctx, handler)
}

func main() {
	if err := app(context.Background()); err != nil {
		log.Fatalf("app crashed: %w", err)
	}
}
