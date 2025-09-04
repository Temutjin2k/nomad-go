package ride

import (
	"context"
	"flag"
	"os"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/app"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

var (
	helpFlag   = flag.Bool("help", false, "Show help message")
	configPath = flag.String("config-path", "config.yaml", "Path to the config yaml file")
)

func Run() {
	flag.Parse()
	if *helpFlag {
		config.PrintHelp()
		return
	}

	ctx := context.Background()
	log := logger.InitLogger("", logger.LevelDebug)

	cfg, err := config.NewConfig(*configPath)
	if err != nil {
		log.Error(ctx, "failed to configure application", err)
		config.PrintHelp()
		return
	}

	// Printing configuration
	config.PrintConfig(cfg)

	if cfg.Mode != "" {
		log = logger.InitLogger(string(cfg.Mode), logger.LevelDebug)
	}

	// Creating application
	app, err := app.NewApplication(ctx, *cfg, log)
	if err != nil {
		log.Error(ctx, "failed to init application", err)
		os.Exit(1)
	}

	// Running the apllication
	if err = app.Run(ctx); err != nil {
		log.Error(ctx, "failed to run application", err)
		os.Exit(1)
	}
}
