package rpc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scorum/scorum-go"
	protocol "github.com/scorum/scorum-go/transport"
	protocolHttp "github.com/scorum/scorum-go/transport/http"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const MaxRequestSize = 25000000
const NodeHTTPS = "https://testnet.scorum.com"
const ChainID = "d3c1f19a4947c296446583f988c43fd1a83818fabaf3454a0020198cb361ebd2"

var router *Router
var client *scorumgo.Client

func init() {
	transport := protocolHttp.NewTransport(NodeHTTPS)
	client = scorumgo.NewClient(transport)
	router = NewRouter(client, NewVerifier(ChainID), MaxRequestSize)
}

func TestRouter_ExistingRoute(t *testing.T) {
	route := Route{"api", "method"}

	invoked := false
	router.Register(route, func(ctx *Context) {
		invoked = true
	})
	router.route(route, &Context{log: logrus.NewEntry(logrus.StandardLogger())})
	require.True(t, invoked)
}

func TestRouter_NotFoundRoute(t *testing.T) {
	route := Route{"api", "method"}

	invoked := false
	router.Register(route, func(ctx *Context) {
		invoked = true
	})
	router.route(Route{"api", "dummy"}, NewContext(nil, httptest.NewRecorder()))
	require.False(t, invoked)
}

func TestRouter_MaxRequestSize(t *testing.T) {
	router := NewRouter(client, NewVerifier(ChainID), 1000)
	route := Route{"api", "dummy"}

	invoked := false
	router.Register(route, func(ctx *Context) {
		invoked = true
	})

	body := []byte(`{
		"id": 3,
		"method": "call",
		"params": ["api","dummy",[]]
	}`)

	w := httptest.NewRecorder()

	// Next 2 tests use one instance of invoked. Bear in mind.
	t.Run("invalid_size", func(t *testing.T) {
		router.maxBodySize = 1
		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))

		router.Handle(w, req)
		require.False(t, invoked)
	})
	t.Run("valid_size", func(t *testing.T) {
		router.maxBodySize = 1000
		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))

		router.Handle(w, req)
		require.True(t, invoked)
	})
}

func TestRouter_HTTPMethod(t *testing.T) {
	t.Run("GetMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		router.Handle(w, req)
		resp := w.Result()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		require.NotEmpty(t, body)

		var rpcResp protocol.RPCResponse
		require.NoError(t, json.Unmarshal(body, &rpcResp))
		require.Equal(t, InvalidRequestCode, rpcResp.Error.Code)
		require.Equal(t, "only POST http method is supported", rpcResp.Error.Message)
	})
	t.Run("OptionsMethod", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/", nil)
		w := httptest.NewRecorder()

		router.Handle(w, req)
		resp := w.Result()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		require.Equal(t, 0, w.Body.Len())

		require.Equal(t, resp.Header.Get("Access-Control-Allow-Origin"), "*")
	})
}

func TestGetSignPubKey(t *testing.T) {
	expected := "0366b11f2f616e44c59bcf082a3e00e77e6b9c0057161a62af3fc16176eb6ba104"

	transport := protocolHttp.NewTransport(NodeHTTPS)
	client := scorumgo.NewClient(transport)

	t.Run("existing posting key", func(t *testing.T) {
		keys, err := GetSignPubKeys(client, "azucena")
		require.NoError(t, err)
		require.Len(t, keys, 3)

		// azucena has owner, active and posting keys equal
		for _, key := range keys {
			require.Equal(t, expected, hex.EncodeToString(key))
		}
	})

	t.Run("not existing account", func(t *testing.T) {
		_, err := GetSignPubKeys(client, "blaazucena") //not existing account
		require.Error(t, err)
		require.Equal(t, "account not found", err.Error())
	})

}
