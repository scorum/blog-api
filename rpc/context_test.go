package rpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	protocol "github.com/scorum/scorum-go/transport"
	"github.com/stretchr/testify/require"
)

func TestRouter_ParseInvalidRPCRequest(t *testing.T) {
	t.Run("InvalidRPCRequest", func(t *testing.T) {

		req := httptest.NewRequest("POST", "/", bytes.NewBufferString("{}"))
		w := httptest.NewRecorder()
		ctx := NewContext(req, w)

		require.False(t, ctx.Parse())

		resp := w.Result()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		require.NotEmpty(t, body)

		var rpcResp protocol.RPCResponse
		require.NoError(t, json.Unmarshal(body, &rpcResp))
		require.Equal(t, InvalidRequestCode, rpcResp.Error.Code)
	})
	t.Run("InvalidRPCMethod", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString(
			`{"method":"call1","params":["blockchain_history_api","get_ops_in_block",[127,0]],"id":10}`,
		))
		w := httptest.NewRecorder()
		ctx := NewContext(req, w)

		require.False(t, ctx.Parse())
		resp := w.Result()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		require.NotEmpty(t, body)

		var rpcResp protocol.RPCResponse
		require.NoError(t, json.Unmarshal(body, &rpcResp))
		require.Equal(t, InvalidRequestCode, rpcResp.Error.Code)
	})
}
