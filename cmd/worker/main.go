// Command worker is the background consumer service. It subscribes to the event
// bus (NATS) and processes jobs — today persisting notifications; push / email /
// SMS consumers can be added here later. Run as its own process (scale out by
// running more instances; the queue group splits the work).
package main

import (
	"coachwise/src/alert"
	"coachwise/src/config"
	"coachwise/src/events"
	"coachwise/src/sms"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	alert.Init(config.Config.Discord.AlertWebhook, envName())
	events.Connect(config.Config.Nats.URL)
	sms.Init()
	events.StartNotificationConsumer()
	events.StartSMSConsumer()
	events.StartAlertConsumer()
	log.Println("worker: running (Ctrl+C to stop)")

	// Block until interrupted.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("worker: shutting down")
}

// envName labels alerts so a failure on a dev box is never mistaken for one in
// production.
func envName() string {
	if config.Config.Debug {
		return "dev"
	}
	return "production"
}
