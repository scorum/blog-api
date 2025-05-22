package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileStorage(t *testing.T) {
	defer cleanUp(t)

	storage := NewProfileStorage(dbWrite)
	require.NoError(t, storage.Create(leonarda))
}
