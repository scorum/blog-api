package main

import (
	"time"

	"gitlab.scorum.com/blog/api/broadcast/types"

	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcutil"
	"github.com/scorum/scorum-go/sign"
	t "github.com/scorum/scorum-go/types"
	"gitlab.scorum.com/blog/api/rpc"
)

const (
	chain   = "d3c1f19a4947c296446583f988c43fd1a83818fabaf3454a0020198cb361ebd2"
	account = "roselle"
	wif     = ""
)

func main() {
	// operation can be changed
	op := types.RemoveDraftOperation{
		Account: account,
		ID:      "someid123",
	}

	expires := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
	tx := types.Transaction{
		RefBlockNum:    31231,
		RefBlockPrefix: 434234234,
		Expiration:     &t.Time{Time: &expires},
		Operations: types.Operations{
			&op,
		},
	}

	verifier := rpc.NewVerifier(chain)
	digest, _ := verifier.TransactionDigest(&tx)

	w, err := btcutil.DecodeWIF(wif)
	if err != nil {
		panic(err)
	}
	privKey := w.PrivKey

	sig := sign.SignBufferSha256(digest, privKey.ToECDSA())
	sigHex := hex.EncodeToString(sig)

	tx.Signatures = []string{sigHex}

	fmt.Printf("Account: %s\n", account)
	fmt.Printf("Operation: %s\n", op.Type())
	fmt.Println("----------")
	fmt.Printf("RefBlockNum: %d\n", tx.RefBlockNum)
	fmt.Printf("RefBlockPrefix: %d\n", tx.RefBlockPrefix)
	fmt.Printf("Expiration: %s\n", tx.Expiration)
	fmt.Printf("Signatures: %s\n", tx.Signatures)
}
