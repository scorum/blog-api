package broadcast

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/sign"
	protocol "github.com/scorum/scorum-go/transport"
	"github.com/scorum/scorum-go/transport/http"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/rpc"
)

const nodeHTTPS = "https://testnet.scorum.com"

func TestInvalidSignature(t *testing.T) {
	transport := http.NewTransport(nodeHTTPS)
	client := scorumgo.NewClient(transport)

	router := NewTransactionRouter(client, rpc.NewVerifier(""))

	testdata, _ := os.Open("testdata/invalid_signature.json")
	req := httptest.NewRequest("POST", "/", testdata)
	w := httptest.NewRecorder()

	ctx := rpc.NewContext(req, w)
	require.True(t, ctx.Parse())

	// act
	called := false
	router.Register(types.FollowOpType, func(_ types.Operation) *rpc.Error {
		called = true
		return nil
	})
	router.Route(ctx)

	// assert
	require.False(t, called)

	body, _ := ioutil.ReadAll(w.Result().Body)
	require.NotEmpty(t, body)
	t.Log(string(body))

	var rpcResp protocol.RPCResponse
	require.NoError(t, json.Unmarshal(body, &rpcResp))

	require.Equal(t, uint64(3), rpcResp.ID)
	require.NotNil(t, rpcResp.Error)
	require.Equal(t, rpc.InvalidParameterCode, rpcResp.Error.Code)
	require.Equal(t, "transaction is not valid", rpcResp.Error.Message)
}

func TestValidSignature(t *testing.T) {
	transport := http.NewTransport(nodeHTTPS)
	client := scorumgo.NewClient(transport)

	router := NewTransactionRouter(client, rpc.NewVerifier(sign.TestChain.ID))

	testdata, _ := os.Open("testdata/valid_signature.json")
	req := httptest.NewRequest("POST", "/", testdata)
	w := httptest.NewRecorder()

	ctx := rpc.NewContext(req, w)
	require.True(t, ctx.Parse())

	// act
	called := false
	router.Register(types.FollowOpType, func(_ types.Operation) *rpc.Error {
		called = true
		return nil
	})
	router.Route(ctx)

	// true
	require.True(t, called)

	body, _ := ioutil.ReadAll(w.Result().Body)
	require.NotEmpty(t, body)
	t.Log(string(body))

	var rpcResp protocol.RPCResponse
	require.NoError(t, json.Unmarshal(body, &rpcResp))

	require.Equal(t, uint64(3), rpcResp.ID)
	require.Nil(t, rpcResp.Error)
	require.NotNil(t, rpcResp.Result)
}
