package rpc

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/scorum/scorum-go/sign"
	scorumtype "github.com/scorum/scorum-go/types"
	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
)

var (
	pubKey []byte
	trx    types.Transaction
)

func init() {
	pubKey = []byte("1dbe5db4f9c3da58e429673ee9265254f1e5cc5962ba")

	expired := time.Unix(0, 0)
	trx = types.Transaction{
		Expiration: &scorumtype.Time{
			Time: &expired,
		},
	}
}

func TestVerifyInvalid(t *testing.T) {
	verifier := NewVerifier(sign.ScorumChain.ID)
	trx.Signatures = []string{"1f023c608241bdd20b56c87a7a3ee3919393714a036cd15eaad71a7fdd3077f27d23dbe5db4f9c3da58e429673ee9265254f1e5cc5962ba0f4024e58b83076a6d5"}

	valid, err := verifier.VerifyTransaction(&trx, [][]byte{pubKey})
	require.NoError(t, err)
	require.False(t, valid)
}

func TestVerifyManyInvalidSignatures(t *testing.T) {
	verifier := NewVerifier(sign.ScorumChain.ID)
	signature := "1f023c608241bdd20b56c87a7a3ee3919393714a036cd15eaad71a7fdd3077f27d23dbe5db4f9c3da58e429673ee9265254f1e5cc5962ba0f4024e58b83076a6d5"
	trx.Signatures = []string{signature, signature, signature, signature}

	valid, err := verifier.VerifyTransaction(&trx, [][]byte{pubKey})
	require.NoError(t, err)
	require.False(t, valid)
}

func TestVerifyInvalidSignature(t *testing.T) {
	verifier := NewVerifier(sign.ScorumChain.ID)
	trx.Signatures = []string{"1234"}

	valid, err := verifier.VerifyTransaction(&trx, [][]byte{pubKey})
	require.NoError(t, err)
	require.False(t, valid)
}

func TestDigest(t *testing.T) {
	verifier := NewVerifier(sign.ScorumChain.ID)
	digest, err := verifier.TransactionDigest(&trx)

	require.NoError(t, err)
	require.Equal(t, "bffeed9538d70b0c005735a62c90d6b46c9d05f1e91db5a912cd26188bcbb618",
		hex.EncodeToString(digest))
}
