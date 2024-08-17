package main

import (
	"coachwise/src/app"
	"coachwise/src/config"
	"coachwise/src/database"
	"time"
)

func main() {
	config.Init("config.yml")
	database.Connect(&database.ConnectOption{
		URL:         config.Config.Database.URL,
		SqlDir:      config.Config.Database.SqlDir,
		MaxRequests: 5,
		Interval:    30 * time.Second,
		Timeout:     5 * time.Second,
	})

	app.Serve()
}
