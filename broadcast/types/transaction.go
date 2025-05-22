package types

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/scorum/scorum-go/encoding/transaction"
	"github.com/scorum/scorum-go/types"
)

type Transaction struct {
	RefBlockNum    uint16      `json:"ref_block_num"`
	RefBlockPrefix uint32      `json:"ref_block_prefix"`
	Expiration     *types.Time `json:"expiration" validate:"required"`
	Operations     Operations  `json:"operations" validate:"required,eq=1"`
	Signatures     []string    `json:"signatures" validate:"required,gt=0,dive,required"`
}

// Serialize transaction to a byte array
func (tx *Transaction) Serialize() ([]byte, error) {
	if tx.Expiration == nil {
		return nil, errors.New("expiration should not be nil")
	}

	var b bytes.Buffer
	encoder := transaction.NewEncoder(&b)

	enc := transaction.NewRollingEncoder(encoder)

	enc.Encode(tx.RefBlockNum)
	enc.Encode(tx.RefBlockPrefix)
	enc.Encode(tx.Expiration)

	enc.EncodeUVarint(uint64(len(tx.Operations)))
	for _, op := range tx.Operations {
		enc.Encode(op)
	}

	// extensions are not supported yet.
	enc.EncodeUVarint(0)

	if err := enc.Err(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
