package verify

/*
 #cgo LDFLAGS: ${SRCDIR}/c-secp256k1/.libs/libsecp256k1.a -lsecp256k1 -lgmp
 #include <stdlib.h>
 #include "signing.h"
*/
import "C"
import (
	"bytes"
	"encoding/hex"
	"unsafe"

	"github.com/pkg/errors"
)

var context *C.secp256k1_context

func init() {
	context = C.secp256k1_context_create(C.SECP256K1_CONTEXT_VERIFY | C.SECP256K1_CONTEXT_SIGN)
}

// VerifyAny checks whether the given signatures is signed with any of the given public keys
func VerifyAny(pubKeys [][]byte, signature string, digest []byte) (bool, error) {
	if len(signature) != 130 {
		return false, nil
	}

	pubKeyFound, err := extractPublicKeys(signature, digest)
	if err != nil {
		return false, err
	}

	for _, key := range pubKeys {
		if bytes.Equal(key, pubKeyFound) {
			return true, nil
		}
	}
	return false, nil
}

func extractPublicKeys(signature string, digest []byte) ([]byte, error) {
	cDigest := C.CBytes(digest)
	defer C.free(cDigest)

	// Make sure to free memory.
	cSigs := make([]unsafe.Pointer, 0, 1)
	defer func() {
		for _, cSig := range cSigs {
			C.free(cSig)
		}
	}()

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode signature hex")
	}

	recoverParameter := sig[0] - 27 - 4
	sig = sig[1:]

	cSig := C.CBytes(sig)
	cSigs = append(cSigs, cSig)

	var publicKey [33]byte

	// validate recovery parameter
	if recoverParameter < 0 || recoverParameter > 4 {
		return nil, nil
	}

	code := C.verify_recoverable_signature(
		context,
		(*C.uchar)(cDigest),
		(*C.uchar)(cSig),
		(C.int)(recoverParameter),
		(*C.uchar)(&publicKey[0]),
	)
	if code == 1 {
		return publicKey[:], nil
	}

	return []byte{}, nil
}
