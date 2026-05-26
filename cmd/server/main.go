// Package main Chat API
//
// Real-time chat application API with groups, channels, direct messages, and voice calls.
//
//	Schemes: http, https
//	BasePath: /api/v1
//	Version: 1.0.0
//
//	SecurityDefinitions:
//	  BearerAuth:
//	    type: apiKey
//	    name: Authorization
//	    in: header
//	    description: "Enter JWT token with 'Bearer ' prefix"
//
//	Contact: API Support <prozorenko24@gmail.com>
//
//	License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
// swagger:meta
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"zchat/config"
	_ "zchat/docs"
	"zchat/internal/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("config error: %s", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("app run error: %s", err)
	}
}
