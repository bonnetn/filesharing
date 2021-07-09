package main

import (
	"context"
	"github.com/bonnetn/filesharing/endpoint"
	"github.com/bonnetn/filesharing/handler"
	"github.com/bonnetn/filesharing/server"
	"log"
)

func app(ctx context.Context) error {
	r := endpoint.ChannelRepository{}
	c := endpoint.ConnectionController{Repository: &r}
	h := handler.NewHandler(&c)
	return server.Run(ctx, h)
}

func main() {
	if err := app(context.Background()); err != nil {
		log.Fatalf("app crashed: %w", err)
	}
}
