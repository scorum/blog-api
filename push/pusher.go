package push

import (
	"fmt"

	"errors"

	"github.com/appleboy/go-fcm"
	log "github.com/sirupsen/logrus"
)

var ErrTokenUnregistered = errors.New("token unregistered")

type Pusher struct {
	client *fcm.Client
}

type Push struct {
	Title       string
	Body        string
	ClickAction string
}

type Config struct {
	APIKey string `yaml:"api_key"`
}

func NewPusher(c Config) (*Pusher, error) {
	client, err := fcm.NewClient(c.APIKey)
	if err != nil {
		return nil, err
	}

	return &Pusher{
		client: client,
	}, nil
}

func (p *Pusher) SendWebPush(token string, push Push) error {
	message := &fcm.Message{
		Notification: &fcm.Notification{
			Title:       push.Title,
			Body:        push.Body,
			Icon:        "https://scorum.com/assets/public/favicon/apple-touch-icon.png",
			ClickAction: push.ClickAction,
		},
		To: token,
	}

	resp, err := p.client.Send(message)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	if len(resp.Results) == 0 {
		return fmt.Errorf("empty results response")
	}

	if resp.Results[0].Unregistered() {
		return ErrTokenUnregistered
	}

	log.Debugf("successfully send message to %s: %v", token, resp)
	return nil
}
