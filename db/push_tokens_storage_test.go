package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPushRegistrationStorage(t *testing.T) {
	defer cleanUp(t)

	const (
		token1 = "token1"
		token2 = "token2"
	)

	require.NoError(t, NewProfileStorage(dbWrite).Create(leonarda))

	storage := NewPushTokensStorage(dbWrite)

	tokens, err := storage.GetTokensByAccount(leonarda)
	require.NoError(t, err)
	require.Empty(t, tokens)

	require.NoError(t, storage.Add(leonarda, token1))
	require.NoError(t, storage.Add(leonarda, token2))

	tokens, err = storage.GetTokensByAccount(leonarda)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	require.Contains(t, tokens, token1)
	require.Contains(t, tokens, token2)

	require.NoError(t, storage.Delete(leonarda, token1))
	require.NoError(t, storage.Delete(leonarda, token2))

	tokens, err = storage.GetTokensByAccount(leonarda)
	require.NoError(t, err)

	require.Empty(t, tokens)
}
