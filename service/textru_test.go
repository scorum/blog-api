package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTextruClient(t *testing.T) {
	t.Skip()
	client := &TextRUClient{
		key: textRUKey,
	}

	id, err := client.submitPostForCheck(testText, "")
	require.NoError(t, err)
	_, err = client.checkResults(id)
	require.NoError(t, err)
}
