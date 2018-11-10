package slack

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/helper/jsonutil"
	"gitlab.morningconsult.com/mci/go-elasticsearch-alerts/command/alert"
)

const (
	defaultChannel  string = "#error-alerts"
	defaultUsername string = "go-alerts"
	defaultEmoji    string = ":robot_face:"
)

// Ensure Helper adheres to the alert.AlertHandler interface
var _ alert.AlertMethod = (*SlackAlertMethod)(nil)

type SlackAlertMethodConfig struct {
	WebhookURL string
	Client     *http.Client
	Channel    string
	Username   string
	Text       string
	Emoji      string
}

type SlackAlertMethod struct {
	webhookURL string
	client     *http.Client
	channel    string
	username   string
	text       string
	emoji      string
}

type Payload struct {
	Channel     string        `json:"channel"`
	Username    string        `json:"username,omitempty"`
	Text        string        `json:"text,omitempty"`
	Emoji       string        `json:"icon_emoji,omitempty"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

func NewSlackAlertMethod(config *SlackAlertMethodConfig) *SlackAlertMethod {
	if config.Client == nil {
		config.Client = cleanhttp.DefaultClient()
	}

	if config.Channel == "" {
		config.Channel = defaultChannel
	}

	if config.Username == "" {
		config.Username = defaultUsername
	}

	if config.Emoji == "" {
		config.Emoji = defaultEmoji
	}

	return &SlackAlertMethod{
		channel:    config.Channel,
		webhookURL: config.WebhookURL,
		client:     config.Client,
		text:       config.Text,
		emoji:      config.Emoji,
	}
}

func (s *SlackAlertMethod) Write(ctx context.Context, records []*alert.Record) error {
	if records == nil || len(records) < 1 {
		return nil
	}
	return s.post(ctx, s.BuildPayload(records))
}

func (s *SlackAlertMethod) BuildPayload(records []*alert.Record) *Payload {
	payload := &Payload{
		Channel:  s.channel,
		Username: s.username,
		Text:     s.text,
		Emoji:    s.emoji,
	}

	for _, record := range records {
		att := NewAttachment(&AttachmentConfig{
			Fallback: record.Title,
			Pretext:  record.Title,
			Text:     record.Text,
		})

		for _, field := range record.Fields {
			f := &Field{
				Title: field.Key,
				Value: fmt.Sprintf("%d", field.Count),
				Short: true,
			}
			att.Fields = append(att.Fields, f)
		}
		payload.Attachments = append(payload.Attachments, att)
	}
	return payload
}

func (s *SlackAlertMethod) post(ctx context.Context, payload *Payload) error {
	data, err := jsonutil.EncodeJSON(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(data))
	req.Header.Add("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making HTTP request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 || resp.StatusCode != 201 || resp.StatusCode != 202 {
		return fmt.Errorf("received non-200 status code: %s", resp.Status)
	}

	return err
}