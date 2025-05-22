package push

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const apiKey = "AAAAYAyTbt4:APA91bHoJVU40vFQMtuqQlR78BfqQjl2moVEUPUTqI2hOhUcQyMGTNcF6F5jNusbVDFKyzaXumV2Gl3J2F8UneFBcO1nstcu48UorbBQnq7oL6fuI5LuHxHTFKdamRJogZ6Pbczq5ryYSclqbNvLAvsWpAFODprRiA"

func TestPusherTokenValid(t *testing.T) {
	t.Skip()
	pusher, err := NewPusher(Config{
		APIKey: apiKey,
	})
	require.NoError(t, err)

	const token = "ct_wJ38DZEc:APA91bE_srdJfqUzj-7g3fxgPUScNYLsgBERGCFepECZE6xjGNcpSNaqZbvePs3Gio8zR7EnfBNaigXRH0LZC5mHmqWIiwMwSonpwpgDRqeCwelTDLT_2WsVfKotrU6oUIUF7lZYAyHK-z1fe9RfZ_pXK7ckgUX3jQ"

	require.NoError(t, pusher.SendWebPush(token, Push{
		Title:       "test",
		Body:        "body",
		ClickAction: "https://ya.ru",
	}))
}

func TestPusherTokenUnregistered(t *testing.T) {
	pusher, err := NewPusher(Config{
		APIKey: apiKey,
	})
	require.NoError(t, err)

	const token = "ct_wJ38DZEc:APA91bE_srdJfqUz"

	require.Equal(t, ErrTokenUnregistered, pusher.SendWebPush(token, Push{
		Title:       "test",
		Body:        "body",
		ClickAction: "https://ya.ru",
	}))
}
