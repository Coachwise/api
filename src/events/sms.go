package events

import (
	"encoding/json"
	"coachwise/src/logger"

	"coachwise/src/sms"

	"github.com/nats-io/nats.go"
)

// SubjectSMS carries SMSJob messages — outbound texts (OTP codes now) sent by a
// worker so the API request returns immediately.
const SubjectSMS = "events.sms"

type SMSJob struct {
	To string `json:"to"`
	// Exactly one of Code (OTP → verify/lookup template) or Text (free-text) is set.
	Code string `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}

// deliver routes a job to the OTP or free-text path.
func (j SMSJob) deliver() error {
	if j.Code != "" {
		return sms.SendOTP(j.To, j.Code)
	}
	return sms.Send(j.To, j.Text)
}

// EmitOTP queues a verification code for delivery (routed to the country's OTP
// provider). See emit for the bus/fallback behaviour.
func EmitOTP(to, code string) { emit(SMSJob{To: to, Code: code}) }

// EmitSMS queues a free-text message for delivery.
func EmitSMS(to, text string) { emit(SMSJob{To: to, Text: text}) }

// emit publishes a job to the bus (a worker sends it, keeping the request fast);
// when the bus is down it sends inline so OTPs still work in dev / degraded mode.
func emit(job SMSJob) {
	if c := getConn(); c != nil && c.IsConnected() {
		if b, err := json.Marshal(job); err == nil {
			if err := c.Publish(SubjectSMS, b); err == nil {
				return
			}
		}
	}
	if err := job.deliver(); err != nil {
		logger.Errorf("events: sms fallback send: %v", err)
	}
}

// StartSMSConsumer sends queued texts. Queue group → scale with more workers.
func StartSMSConsumer() {
	c := getConn()
	if c == nil {
		logger.Info("events: sms consumer not started (bus disabled)")
		return
	}
	_, err := c.QueueSubscribe(SubjectSMS, "sms-workers", func(m *nats.Msg) {
		var job SMSJob
		if err := json.Unmarshal(m.Data, &job); err != nil {
			return
		}
		if err := job.deliver(); err != nil {
			logger.Errorf("events: send sms: %v", err)
		}
	})
	if err != nil {
		logger.Errorf("events: sms subscribe failed: %v", err)
		return
	}
	logger.Info("events: sms consumer subscribed")
}
