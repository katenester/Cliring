package app

import (
	"cliring/config"
	"cliring/pkg/postgres"
	"context"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

// Run - Building dependencies and logic
func Run() {
	// Download variables env
	if err := godotenv.Load(); err != nil {
		logrus.Fatalf("error initalization db password(file env) %s", err.Error())
	}
	cfg, err := config.New()
	if err != nil {
		logrus.Fatalf("error load env %s", err.Error())
	}

	ctx := context.Background()

	db := postgres.New(cfg)
	if err = db.Open(ctx); err != nil {
		logrus.Fatalf("error open db %s", err.Error())
	}

	// Dependency injection for architecture application
	repos := repository.NewRepository(db)
	services := service.NewService(repos)
	handlers := transport.NewHandler(services)
	srv := new(transport.Server)
	go func() {
		if err := srv.Run(cfg.HTTPPort, handlers.InitRoutes()); err != nil {
			logrus.Fatalf("error occured while running http server %s", err.Error())
		}
	}()

	logrus.Print("todo server started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Println("shutting down server...")
	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Fatalf("error occured while shutting down server %s", err.Error())
	}
	if err := db.Close(ctx); err != nil {
		logrus.Fatalf("error occured while closing db %s", err.Error())
	}
}
