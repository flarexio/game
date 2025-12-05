package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/game"
	"github.com/flarexio/game/nvstream"
)

const (
	Version string = "0.0.0"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err.Error())
	}

	path := filepath.Join(homeDir, ".flarex", "game")

	nvstreamCmd := &cli.Command{
		Name:        "nvstream",
		Description: "NVIDIA GameStream compatible client for game streaming.",
		Commands: []*cli.Command{
			{
				Name:        "pair",
				Description: "Pair with NVIDIA GameStream server.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Usage:   "Specifies the working directory for the Game service.",
						Sources: cli.EnvVars("GAME_PATH"),
						Value:   path,
					},
					&cli.StringFlag{
						Name:  "host",
						Usage: "The hostname or IP address of the GameStream server.",
						Value: "localhost",
					},
				},
				Action: pair,
			},
		},
	}

	cmd := &cli.Command{
		Name:        "game",
		Description: "Edge Gaming services for real-time game streaming and remote game controller access to edge computer.",
		Commands:    []*cli.Command{nvstreamCmd},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Usage:   "Specifies the working directory for the Game service.",
				Sources: cli.EnvVars("GAME_PATH"),
				Value:   path,
			},
			&cli.StringFlag{
				Name:    "nats",
				Sources: cli.EnvVars("NATS_URL"),
				Value:   "wss://nats.flarex.io",
			},
		},
		Action: run,
	}

	ctx := context.Background()
	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	defer log.Sync()

	zap.ReplaceGlobals(log)

	path := cmd.String("path")

	f, err := os.Open(filepath.Join(path, "config.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg *game.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	cfg.Path = path

	natsURL := cmd.String("nats")
	natsCreds := filepath.Join(path, "user.creds")

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

func pair(ctx context.Context, cmd *cli.Command) error {
	host := cmd.String("host")
	path := cmd.String("path")

	http, err := nvstream.NewHTTP("MyGameClient", host, path)
	if err != nil {
		return err
	}

	client := nvstream.NewPairingManager(http)

	// Client 產生 PIN
	pin := fmt.Sprintf("%04d", rand.Intn(10000))

	fmt.Println("===========================================")
	fmt.Printf("配對 PIN 碼: %s\n", pin)
	fmt.Println("===========================================")
	fmt.Println("步驟：")
	fmt.Println("1. 記住這個 PIN 碼")
	fmt.Println("2. 5 秒後會自動開始配對")
	fmt.Println("3. Sunshine 會彈出配對視窗，請輸入 PIN 碼")
	fmt.Println("===========================================")

	// 等待 5 秒讓使用者準備
	fmt.Println("5 秒後開始配對流程...")

	time.Sleep(5 * time.Second)

	fmt.Println("開始配對...")

	// 執行配對
	state := client.Pair(pin)

	if state != nvstream.PairStatePaired {
		fmt.Printf("配對失敗，狀態碼: %d\n", state)
		return nil
	}

	fmt.Println("配對成功！")

	return nil
}
