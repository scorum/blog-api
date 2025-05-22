package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/btcsuite/btcutil"
	"github.com/scorum/scorum-go/sign"
	"gitlab.scorum.com/blog/api/rpc"
)

const (
	chain   = "d3c1f19a4947c296446583f988c43fd1a83818fabaf3454a0020198cb361ebd2"
	account = "roselle"
	wif     = ""
	salt    = "1111"
)

func main() {
	verifier := rpc.NewVerifier(chain)

	var params []*json.RawMessage
	paramsStr := `[]`

	if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
		log.Fatal(err)
	}

	digest, _ := verifier.SignedRequestDigest(account, salt, params)

	w, err := btcutil.DecodeWIF(wif)
	if err != nil {
		panic(err)
	}
	privKey := w.PrivKey

	sig := sign.SignBufferSha256(digest, privKey.ToECDSA())
	sigHex := hex.EncodeToString(sig)

	fmt.Printf("Account: %s\n", account)
	fmt.Printf("Salt: %s\n", salt)
	fmt.Println("----------")
	fmt.Printf("Signature: %s\n", sigHex)
}
