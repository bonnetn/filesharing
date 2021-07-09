package main

import (
	"context"
	"github.com/bonnetn/filesharing/internal"
	"log"
)

func app(ctx context.Context) error {
	var (
		repository = internal.NewChannelRepository()
		get        = internal.NewGetOperation(&repository)
		post       = internal.NewPostOperation(&repository)
		h          = internal.NewHandler(&get, &post)
	)
	return internal.RunServer(ctx, h)
}

func main() {
	if err := app(context.Background()); err != nil {
		log.Fatalf("app crashed: %w", err)
	}
}
