package main

import (
	"coachwise/src/alert"
	"coachwise/src/app"
	"coachwise/src/app/ws"
	"coachwise/src/config"
	"coachwise/src/events"
	"coachwise/src/metrics"
	"coachwise/src/payments"
	"coachwise/src/sms"
	"coachwise/src/storage"
	"coachwise/src/utils"
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
	utils.InitDiscord(config.Config.Discord.Proxy)
	alert.Init(config.Config.Discord.AlertWebhook, envName())
	events.InitSupport(config.Config.Discord.SupportWebhook)
	events.Connect(config.Config.Nats.URL)
	payments.Init()
	sms.Init()
	storage.Init()
	ws.Start()

	// Flush a Prometheus-text snapshot of all metrics to the configured file
	// every minute (in place, no history). Empty file disables it.
	metrics.StartSnapshotWriter(config.Config.Metrics.File, time.Minute)

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
