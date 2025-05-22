package rpc

import (
	"fmt"
	"net/http"
	"strings"

	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
	"github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/types"
)

type Route struct {
	API    string
	Method string
}

type APIHandler func(ctx *Context)
type SignedAPIHandler func(ctx *Context, account string, params []*json.RawMessage)

type Router struct {
	blockchain  *scorumgo.Client
	verifier    Verifier
	routes      map[Route]APIHandler
	maxBodySize int64
}

func NewRouter(blockchain *scorumgo.Client, verifier Verifier, maxBodySize int64) *Router {
	return &Router{
		blockchain:  blockchain,
		verifier:    verifier,
		routes:      make(map[Route]APIHandler),
		maxBodySize: maxBodySize,
	}
}

func (router *Router) Register(r Route, h APIHandler) {
	router.routes[r] = h
}

func (router *Router) route(r Route, ctx *Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.New(fmt.Sprintf("unknown panic: %s", r))
			}

			ctx.log.Error(err)
			if ctx.flushed {
				ctx.log.Warningf("Has panic but context flushed: %s", err)
				return
			}
			ctx.WriteError(InternalErrorCode, err.Error())
		}
	}()

	route, ok := router.routes[r]
	if !ok {
		ctx.WriteError(RouteNotRegisteredCode, fmt.Sprintf("route: %s not registered", r))
		return
	}

	/*
		ctx.log.Debug("processing started")
		defer func(start time.Time) {
			ctx.log.WithFields(log.Fields{
				"elapsed": time.Since(start),
				"route":   fmt.Sprintf("%+v", r),
			}).Debug("processed")
		}(time.Now())
	*/
	route(ctx)
}

func (router *Router) Handle(writer http.ResponseWriter, request *http.Request) {
	// limit request size
	request.Body = http.MaxBytesReader(writer, request.Body, router.maxBodySize)

	// create context
	ctx := NewContext(request, writer)

	// enable CORS(enabled all for all)
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	ctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	switch request.Method {
	case "POST":
		if ok := ctx.Parse(); !ok {
			return
		}
		// process the request
		router.route(Route{
			API:    ctx.API(),
			Method: ctx.Method(),
		}, ctx)

	case "OPTIONS":
		ctx.WriteResult(nil)

	default:
		ctx.WriteError(InvalidRequestCode, "only POST http method is supported")
	}
}

func (router *Router) SignedAPI(handler SignedAPIHandler) APIHandler {
	return func(ctx *Context) {
		var account string
		if err := ctx.Param(0, &account); err != nil {
			ctx.WriteError(InvalidParameterCode, err.Error())
			return
		}

		var salt string
		if err := ctx.Param(1, &salt); err != nil {
			ctx.WriteError(InvalidParameterCode, err.Error())
			return
		}

		var signature string
		if err := ctx.Param(2, &signature); err != nil {
			ctx.WriteError(InvalidParameterCode, err.Error())
			return
		}

		var params []*json.RawMessage
		if err := ctx.Param(3, &params); err != nil {
			ctx.WriteError(InvalidParameterCode, err.Error())
			return
		}

		keys, err := GetSignPubKeys(router.blockchain, account)
		if err != nil {
			ctx.WriteError(InvalidParameterCode, err.Error())
			return
		}

		valid, err := router.verifier.VerifySignedRequest(account, salt, signature, params, keys)
		if err != nil {
			ctx.WriteError(InvalidRequestCode, err.Error())
			return
		}

		if !valid {
			ctx.WriteError(InvalidParameterCode, "signature is not valid")
			return
		}

		handler(ctx, account, params)
	}
}

func GetSignPubKeys(blockchain *scorumgo.Client, name string) ([][]byte, error) {
	const prefix = "SCR"

	extractKeys := func(keys types.StringInt64Map) ([][]byte, error) {
		var out [][]byte
		for key := range keys {
			if strings.Index(key, prefix) != 0 {
				return nil, fmt.Errorf("%s is not a valid key", key)
			}
			key = key[len(prefix):]
			keyWithChecksum := base58.Decode(key)
			out = append(out, keyWithChecksum[:len(keyWithChecksum)-4])
		}
		return out, nil
	}

	accounts, err := blockchain.Database.GetAccounts(name)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("account not found")
	}

	var out [][]byte
	account := accounts[0]

	//owner
	keys, err := extractKeys(account.Owner.KeyAuths)
	if err != nil {
		return nil, errors.Wrap(err, "extract owner keys failed")
	}
	out = append(out, keys...)

	//active
	keys, err = extractKeys(account.Active.KeyAuths)
	if err != nil {
		return nil, errors.Wrap(err, "extract active keys failed")
	}
	out = append(out, keys...)

	//posting
	keys, err = extractKeys(account.Posting.KeyAuths)
	if err != nil {
		return nil, errors.Wrap(err, "extract posting keys failed")
	}
	out = append(out, keys...)

	if len(out) == 0 {
		return nil, errors.New("sing keys not found")
	}

	return out, nil
}
