package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/scorum/scorum-go/encoding/transaction"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/verify"
)

type Verifier interface {
	VerifyTransaction(tx *types.Transaction, pubKeys [][]byte) (bool, error)
	TransactionDigest(tx *types.Transaction) ([]byte, error)

	VerifySignedRequest(account, salt, signature string, params []*json.RawMessage, pubKeys [][]byte) (bool, error)
	SignedRequestDigest(account, salt string, params []*json.RawMessage) ([]byte, error)
}

func NewVerifier(chain string) Verifier {
	return &verifier{chain: chain}
}

type verifier struct {
	chain string
}

func (v *verifier) VerifyTransaction(tx *types.Transaction, pubKeys [][]byte) (bool, error) {
	digest, err := v.TransactionDigest(tx)
	if err != nil {
		return false, err
	}

	// transactions only with one signature are supported
	if len(tx.Signatures) != 1 {
		return false, nil
	}

	return verify.VerifyAny(pubKeys, tx.Signatures[0], digest)
}

// TransactionDigest calculates digest of the given transaction
func (v *verifier) TransactionDigest(tx *types.Transaction) ([]byte, error) {
	var msgBuffer bytes.Buffer

	// Write the chain ID.
	rawChainID, err := hex.DecodeString(v.chain)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode chain ID: %v", v.chain)
	}

	if _, err := msgBuffer.Write(rawChainID); err != nil {
		return nil, errors.Wrap(err, "failed to write chain ID")
	}

	// Write the serialized transaction.
	rawTx, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	if _, err := msgBuffer.Write(rawTx); err != nil {
		return nil, errors.Wrap(err, "failed to write serialized transaction")
	}

	// Compute the digest.
	digest := sha256.Sum256(msgBuffer.Bytes())
	return digest[:], nil
}

func (v *verifier) VerifySignedRequest(account, salt, signature string, params []*json.RawMessage, pubKeys [][]byte) (bool, error) {
	digest, err := v.SignedRequestDigest(account, salt, params)
	if err != nil {
		return false, err
	}

	return verify.VerifyAny(pubKeys, signature, digest)
}

func (v *verifier) SignedRequestDigest(account, salt string, params []*json.RawMessage) ([]byte, error) {
	var msgBuffer bytes.Buffer

	// Write the chain ID.
	rawChainID, err := hex.DecodeString(v.chain)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode chain ID: %v", v.chain)
	}

	if _, err := msgBuffer.Write(rawChainID); err != nil {
		return nil, errors.Wrap(err, "failed to write chain ID")
	}

	// Write the serialized transaction.
	var raw bytes.Buffer
	encoder := transaction.NewEncoder(&raw)

	enc := transaction.NewRollingEncoder(encoder)

	enc.Encode(account)
	enc.Encode(salt)

	for _, param := range params {
		h := hex.EncodeToString([]byte(*param))
		enc.Encode(h)
	}

	if err := enc.Err(); err != nil {
		return nil, err
	}

	if _, err := msgBuffer.Write(raw.Bytes()); err != nil {
		return nil, errors.Wrap(err, "failed to write serialized transaction")
	}

	// Compute the digest.
	digest := sha256.Sum256(msgBuffer.Bytes())
	return digest[:], nil
}
