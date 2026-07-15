package main

import (
	"coachwise/src/alert"
	"coachwise/src/app"
	"coachwise/src/app/ws"
	"coachwise/src/config"
	"coachwise/src/events"
	"coachwise/src/payments"
	"coachwise/src/sms"
	"coachwise/src/storage"
	"time"

	"coachwise/src/database"
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

	// The API PUBLISHES events; the `cmd/worker` service consumes them. It also
	// runs the realtime hub: it subscribes to refetch signals off the bus and
	// pushes them to connected websockets.
	// The sink is initialised here too, not just in the worker: when the bus is
	// down EmitAlert delivers inline from this process.
	alert.Init(config.Config.Discord.AlertWebhook, envName())
	events.Connect(config.Config.Nats.URL)
	payments.Init()
	sms.Init()
	storage.Init()
	ws.Start()

	app.Serve()
}

// envName labels alerts so a panic from a dev box is never mistaken for one from
// production.
func envName() string {
	if config.Config.Debug {
		return "dev"
	}
	return "production"
}
