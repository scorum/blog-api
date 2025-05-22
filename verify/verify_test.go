package verify

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcutil"
	"github.com/stretchr/testify/require"
)

func pubKeyFromWif(t *testing.T, wif string) []byte {
	w, err := btcutil.DecodeWIF(wif)
	require.NoError(t, err)
	w.CompressPubKey = true
	return w.SerializePubKey()
}

func TestVerifyAnyValid(t *testing.T) {
	cases := []struct {
		wif       string
		hash      string
		signature string
	}{
		{
			wif:       "5J7FEcpqc1sZ7ZbKx2kVvBHx2oTjWG2wMU2e2FYX85sGA2qu8KT",
			hash:      "28e55f6ef3d8010caa64b74c0d4ff2e792f5f158170dcb04a2efec0dfec5e4d0",
			signature: "1f65116880dd659a9709956e9409095fa0c2e282fefe2c6511d4fad2b8301cf09b1ee9473100a504e08091acdbc8cd1042e857d637a506720d2b35a6976b1afe99",
		},
		{
			wif:       "5KHK69Be8P8NQLy46KXugJWyNkxw8Nw3Mzue4wD8ygx48emMugd",
			hash:      "b901e39b9f719c41f4ddefa8b3f0742c88a35ec7adee6e06189f99a2598f56cd",
			signature: "1f266da35169f8a552a356c3550779fa43df4426327e361896a3a80d06c9ee9a546d966d9feafbcb9f0c864e13ca517e2f90d00230ecd8645b1b2e110198e576ec",
		},
	}

	for _, c := range cases {
		pubKey := pubKeyFromWif(t, c.wif)

		byteHash, err := hex.DecodeString(c.hash)
		require.NoError(t, err)
		valid, err := VerifyAny([][]byte{pubKey}, c.signature, byteHash)
		require.NoError(t, err)
		require.True(t, valid)
	}
}

func TestVerifyAnyMultiple(t *testing.T) {
	const hash = "28e55f6ef3d8010caa64b74c0d4ff2e792f5f158170dcb04a2efec0dfec5e4d0"

	pubKey1 := pubKeyFromWif(t, "5KHK69Be8P8NQLy46KXugJWyNkxw8Nw3Mzue4wD8ygx48emMugd")
	pubKey2 := pubKeyFromWif(t, "5J7FEcpqc1sZ7ZbKx2kVvBHx2oTjWG2wMU2e2FYX85sGA2qu8KT")

	byteHash, err := hex.DecodeString(hash)
	require.NoError(t, err)

	t.Run("signed with pubKey1", func(t *testing.T) {
		const signature = "1f65116880dd659a9709956e9409095fa0c2e282fefe2c6511d4fad2b8301cf09b1ee9473100a504e08091acdbc8cd1042e857d637a506720d2b35a6976b1afe99"

		valid, err := VerifyAny([][]byte{pubKey1, pubKey2}, signature, byteHash)
		require.NoError(t, err)
		require.True(t, valid)
	})

	t.Run("signed not with pubKey1 nor pubKey2", func(t *testing.T) {
		const signature = "1f266da35169f8a552a356c3550779fa43df4426327e361896a3a80d06c9ee9a546d966d9feafbcb9f0c864e13ca517e2f90d00230ecd8645b1b2e110198e576ec"

		valid, err := VerifyAny([][]byte{pubKey1, pubKey2}, signature, byteHash)
		require.NoError(t, err)
		require.False(t, valid)
	})
}

func TestVerifyAnyInvalid(t *testing.T) {
	cases := []struct {
		wif       string
		hash      string
		signature string
	}{
		{
			wif:       "5KHK69Be8P8NQLy46KXugJWyNkxw8Nw3Mzue4wD8ygx48emMugd",
			hash:      "7901e39b9f719c41f4ddefa8b3f0742c88a35ec7adee6e06189f99a2598f56cd",
			signature: "1f266da35169f8a552a356c3550779fa43df4426327e361896a3a80d06c9ee9a546d966d9feafbcb9f0c864e13ca517e2f90d00230ecd8645b1b2e110198e576ec",
		},
		{
			wif:       "5KHK69Be8P8NQLy46KXugJWyNkxw8Nw3Mzue4wD8ygx48emMugd",
			hash:      "8901e39b9f719c41f4ddefa8b3f0742c88a35ec7adee6e06189f99a2598f56cd",
			signature: "1f266da35169f8a552a356c3550779fa43df4426327e361896a3a80d06c9ee9a546d966d9feafbcb9f0c864e13ca517e2f90d00230ecd8645b1b2e110198e576ec",
		},
	}

	for _, c := range cases {
		pubKey := pubKeyFromWif(t, c.wif)

		byteHash, err := hex.DecodeString(c.hash)
		require.NoError(t, err)
		valid, err := VerifyAny([][]byte{pubKey}, c.signature, byteHash)
		require.NoError(t, err)
		require.False(t, valid)
	}
}
