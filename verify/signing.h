#ifndef GOSTEEMRPC_SIGNING_H
#define GOSTEEMRPC_SIGNING_H

#include "c-secp256k1/include/secp256k1.h"

int sign_transaction(
	const secp256k1_context* ctx,
	const unsigned char *digest,
	const unsigned char *privkey,
	unsigned char *signature,
	int *recid
);

// pubkey is expected to be 33 bytes long so that a compressed public key fits.
int verify_recoverable_signature(
	const secp256k1_context* ctx,
	const unsigned char *digest,
	const unsigned char *signature,
	int recid,
	unsigned char *pubkey
);

#endif