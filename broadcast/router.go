package broadcast

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/apis/network_broadcast"
	log "github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/rpc"
	"gopkg.in/go-playground/validator.v9"
)

type TransactionHandler func(op types.Operation) *rpc.Error

// TransactionRouter routes transaction requests to the corresponding handler
type TransactionRouter struct {
	Blockchain *scorumgo.Client
	Verifier   rpc.Verifier
	routes     map[types.OpType]TransactionHandler
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// NewTransactionRouter creates new TransactionRouter
func NewTransactionRouter(blockchain *scorumgo.Client, verifier rpc.Verifier) *TransactionRouter {
	return &TransactionRouter{
		Blockchain: blockchain,
		Verifier:   verifier,
		routes:     make(map[types.OpType]TransactionHandler),
	}
}

// Register handler of the given operation type
func (router *TransactionRouter) Register(op types.OpType, trxHandler TransactionHandler) {
	router.routes[op] = trxHandler
}

// Route validates and routes the given transaction to the corresponding handler
func (router *TransactionRouter) Route(ctx *rpc.Context) {
	var trx types.Transaction

	if err := ctx.Param(0, &trx); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	if err := validate.Struct(trx); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, errors.Wrap(err, "transaction is invalid").Error())
		return
	}

	ops := trx.Operations
	if len(ops) != 1 {
		ctx.WriteError(rpc.InvalidParameterCode, "only one operation per transaction supported")
		return
	}

	op := ops[0]

	if _, ok := op.(*types.UnknownOperation); ok {
		ctx.WriteError(rpc.InvalidParameterCode, fmt.Sprintf("%s operation is unknown", op.Type()))
		return
	}

	keys, err := rpc.GetSignPubKeys(router.Blockchain, op.GetAccount())
	if err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	valid, err := router.Verifier.VerifyTransaction(&trx, keys)
	if err != nil {
		ctx.WriteError(rpc.InvalidRequestCode, err.Error())
		return
	}

	if !valid {
		ctx.WriteError(rpc.InvalidParameterCode, "transaction is not valid")
		return
	}

	handler, ok := router.routes[op.Type()]
	if !ok {
		log.Fatalf("%s handler is not registered", op.Type())
	}

	if err := validate.Struct(op); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, fmt.Sprintf("invalid request: %s", err))
		return
	}

	// invoke operation handler
	if err := handler(op); err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}

	// write broadcast ok
	ctx.WriteResult(network_broadcast.BroadcastResponse{
		ID:       strconv.FormatUint(ctx.ID, 10),
		Expired:  false,
		BlockNum: 0,
		TrxNum:   0,
	})
}
