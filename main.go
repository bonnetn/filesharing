package main

import (
	"context"
	"github.com/bonnetn/filesharing/internal"
	"log"
)

func app(ctx context.Context) error {
	var (
		repository = internal.NewPendingFileshareRepository()
		get        = internal.NewGetOperation(repository)
		post       = internal.NewCreateOperation(repository)
		handler    = internal.NewHandler(get, post)
	)
	return internal.RunServer(ctx, handler)
}

func main() {
	if err := app(context.Background()); err != nil {
		log.Fatalf("app crashed: %w", err)
	}
}
