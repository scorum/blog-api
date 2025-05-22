package mailer

import (
	"github.com/nsqio/go-nsq"
	"gitlab.scorum.com/blog/api/db"
)

const (
	PlagiarismTopicName = "plagiarism-notifications"
)

type Client struct {
	producer *nsq.Producer
}

func NewClient(nsqdAddress string) (*Client, error) {
	producer, err := nsq.NewProducer(nsqdAddress, nsq.NewConfig())
	if err != nil {
		return nil, err
	}

	return &Client{producer: producer}, producer.Ping()
}

func (s *Client) SendPlagiarismEmail(meta db.PlagiarismRelatedNotificationMeta) error {
	if s.producer != nil {
		return s.producer.Publish(PlagiarismTopicName, meta.ToJson())
	}
	return nil
}
