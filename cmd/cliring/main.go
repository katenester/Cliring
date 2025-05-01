package main

import (
	"cliring/internal/app"
	"github.com/sirupsen/logrus"
)

// @title           MERMAID-DIAGRAM API
// @version         1.0
// @description     Api server for conducting the power of attorney approval process

// @host localhost:3000
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	logrus.SetFormatter(new(logrus.JSONFormatter))

	app.Run()
}

func Close() error {
	return nil
}
