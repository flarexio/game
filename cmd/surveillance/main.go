package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/surveillance"
)

const (
	Version string = "0.0.0"
)

func main() {
	app := &cli.App{
		Name: "surveillance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Usage:   "Specifies the working directory for the Surveillance service.",
				EnvVars: []string{"SURVEILLANCE_PATH"},
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

		path = homeDir + "/.flarex/surveillance"
	}

	f, err := os.Open(path + "/config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg *surveillance.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	natsURL := cli.String("nats")
	natsCreds := path + "/user.creds"

	nc, err := nats.Connect(natsURL,
		nats.Name("surveillance"),
		nats.UserCredentials(natsCreds),
	)
	if err != nil {
		return err
	}
	defer nc.Drain()

	svc := surveillance.NewService(cfg, nc)
	svc = surveillance.LoggingMiddleware(log)(svc)

	srv, err := micro.AddService(nc, micro.Config{
		Name:    "surveillance",
		Version: Version,
	})
	defer srv.Stop()

	if err != nil {
		return err
	}

	group := srv.AddGroup("peers")
	group.AddEndpoint("iceservers", surveillance.ICEServersHandler(svc))
	group.AddEndpoint("negotiation", surveillance.AcceptPeerHandler(svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit // Wait for a termination signal

	log.Info("graceful shutdown", zap.String("singal", sign.String()))
	return nil
}
