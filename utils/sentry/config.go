package sentry

import "fmt"

type Config struct {
	Key string
	Secret string
	Project string
}

func (config Config) GetDSN() string {
	return fmt.Sprintf("https://%s:%s@sentry.io/%s", config.Key, config.Secret, config.Project)
}
