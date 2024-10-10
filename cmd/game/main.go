package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/game"
)

const (
	Version string = "0.0.0"
)

func main() {
	app := &cli.App{
		Name: "game",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Usage:   "Specifies the working directory for the Game service.",
				EnvVars: []string{"GAME_PATH"},
			},
			&cli.StringFlag{
				Name:    "nats",
				EnvVars: []string{"NATS_URL"},
				Value:   "wss://nats.flarex.io",
			},
		},
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond)
	defer cancel()

	<-ctx.Done()
}

func run(cli *cli.Context) error {
	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	defer log.Sync()

	zap.ReplaceGlobals(log)

	path := cli.String("path")
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		path = homeDir + "/.flarex/game"
	}

	f, err := os.Open(path + "/config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg *game.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	natsURL := cli.String("nats")
	natsCreds := path + "/user.creds"

	nc, err := nats.Connect(natsURL,
		nats.Name("game"),
		nats.UserCredentials(natsCreds),
	)
	if err != nil {
		return err
	}
	defer nc.Drain()

	svc, err := game.NewService(cfg, nc)
	if err != nil {
		return err
	}

	svc = game.LoggingMiddleware(log)(svc)
	defer svc.Close()

	srv, err := micro.AddService(nc, micro.Config{
		Name:    "game",
		Version: Version,
	})
	defer srv.Stop()

	if err != nil {
		return err
	}

	group := srv.AddGroup("peers")
	group.AddEndpoint("iceservers", game.ICEServersHandler(svc))
	group.AddEndpoint("negotiation", game.AcceptPeerHandler(svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit // Wait for a termination signal

	log.Info("graceful shutdown", zap.String("singal", sign.String()))
	return nil
}
