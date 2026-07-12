package main

import (
	"coachwise/src/app"
	"coachwise/src/app/ws"
	"coachwise/src/config"
	"coachwise/src/events"
	"coachwise/src/payments"
	"coachwise/src/sms"
	"time"

	database "github.com/socious-io/pkg_database"
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
	events.Connect(config.Config.Nats.URL)
	payments.Init()
	sms.Init()
	ws.Start()

	app.Serve()
}
