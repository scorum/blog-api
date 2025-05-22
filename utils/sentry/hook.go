package sentry

import (
	log "github.com/sirupsen/logrus"
	"github.com/getsentry/raven-go"
	"github.com/davecgh/go-spew/spew"
)

type Hook struct{
}

func (*Hook) Levels() []log.Level {
	return []log.Level{log.ErrorLevel, log.FatalLevel}
}

func (hook *Hook) Fire(entry *log.Entry) error {
	tags := make(map[string]string)
	for k, v := range entry.Data {
		tags[k] = spew.Sdump(v)
	}

	packet := raven.NewPacket(entry.Message)

	currentStacktrace := raven.NewStacktrace(6, 3, []string{})
	if currentStacktrace != nil {
		packet.Interfaces = append(packet.Interfaces, currentStacktrace)
	}

	raven.Capture(packet, tags)

	return nil
}
