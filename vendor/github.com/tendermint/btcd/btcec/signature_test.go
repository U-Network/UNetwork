// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

// decodeHex decodes the passed hex string and returns the resulting bytes.  It
// panics if an error occurs.  This is only used in the tests as a helper since
// the only way it can fail is if there is an error in the test source code.
func decodeHex(hexStr string) []byte {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		panic("invalid hex string in test source: err " + err.Error() +
			", hex: " + hexStr)
	}

	return b
}

// TestSignatureVerify ensures that signature verification works, and that
// non-canonical signatures fail.
func TestSignatureVerify(t *testing.T) {
	pk1, _ := ParsePubKey(decodeHex("022430063aa9ff5f320f47d4352eb2d04a9836b7f135c1cdbae04226384bb5ca9c"), S256())
	pk2, _ := ParsePubKey(decodeHex("03f1f9e87ce98171173c2f8566029f95cf2ae3c60d74334b88e5bf33556cb6b46e"), S256())
	tests := []struct {
		ecKey    *PublicKey
		ecsig    *Signature
		msg      []byte
		expected bool
	}{
		{
			pk1,
			&Signature{
				R: fromHex("b6324dc7b0eb8f109fbb092ab86126a12aca6222cf3129bcb88ada3dde4ed41f"),
				S: fromHex("1e3409ede2c183e4b415b85997697c91285fb883436816c92b8a8df303dde2ef"),
			},
			decodeHex("27ffa644256a2b0456a5a70d689fdceb816f8561e9f012e9225e58564ecc7bab"),
			true,
		},
		{
			pk2,
			&Signature{
				R: fromHex("5395803083d72d74f040494f0b7d5cbb2721571287d67a43fbe25bd8efbb6abe"),
				S: fromHex("16adf29ff5c0ed4a607fe61b461607049260b1f31109739cddd599f77a588724"),
			},
			decodeHex("d4636f24ea65648670948c3373cf5c39dfc44a1d5e35c349f26762dceb43c94c"),
			true,
		},
	}

	for i, test := range tests {
		result := test.ecsig.Verify(test.msg, test.ecKey)
		if result != test.expected {
			t.Errorf("SignatureVerify #%d incorrect result:\n"+
				"got:  %v\nwant: %v", i, result, test.expected)
		}
		test.ecsig.S.Sub(S256().CurveParams.N, test.ecsig.S)
		result = test.ecsig.Verify(test.msg, test.ecKey)
		if result != false {
			t.Errorf("SignatureVerify #%d incorrect result with malleated S:\n"+
				"got:  %v\nwant: %v", i, result, false)
		}
	}
	numTests := 10
	for i := 0; i < numTests; i++ {
		keyBytes := make([]byte, 32)
		rand.Read(keyBytes)
		privKey, pubKey := PrivKeyFromBytes(S256(), keyBytes)
		msgBytes := make([]byte, 32)
		rand.Read(msgBytes)
		fmt.Println(hex.EncodeToString(msgBytes))
		sig, _ := privKey.Sign(msgBytes)
		fmt.Println(hex.EncodeToString(pubKey.SerializeCompressed()))
		fmt.Println(hex.EncodeToString(sig.R.Bytes()))
		fmt.Println(hex.EncodeToString(sig.S.Bytes()))
	}
}

// TestSignatureSerialize ensures that serializing signatures works as expected.
func TestSignatureSerialize(t *testing.T) {
	tests := []struct {
		name     string
		ecsig    *Signature
		expected []byte
	}{
		// signature from bitcoin blockchain tx
		// 0437cd7f8525ceed2324359c2d0ba26006d92d85
		{
			"valid 1 - r and s most significant bits are zero",
			&Signature{
				R: fromHex("4e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd41"),
				S: fromHex("181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d09"),
			},
			decodeHex("4e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd" +
				"41181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d09"),
		},
		// signature from bitcoin blockchain tx
		// cb00f8a0573b18faa8c4f467b049f5d202bf1101d9ef2633bc611be70376a4b4
		{
			"valid 2 - r most significant bit is one",
			&Signature{
				R: fromHex("0082235e21a2300022738dabb8e1bbd9d19cfb1e7ab8c30a23b0afbb8d178abcf3"),
				S: fromHex("24bf68e256c534ddfaf966bf908deb944305596f7bdcc38d69acad7f9c868724"),
			},
			decodeHex("82235e21a2300022738dabb8e1bbd9d19cfb1e7ab8c30a23b0afbb8d178abc" +
				"f324bf68e256c534ddfaf966bf908deb944305596f7bdcc38d69acad7f9c868724"),
		},
		// signature from bitcoin blockchain tx
		// fda204502a3345e08afd6af27377c052e77f1fefeaeb31bdd45f1e1237ca5470
		{
			"valid 3 - s most significant bit is one",
			&Signature{
				R: fromHex("1cadddc2838598fee7dc35a12b340c6bde8b389f7bfd19a1252a17c4b5ed2d71"),
				S: new(big.Int).Add(fromHex("00c1a251bbecb14b058a8bd77f65de87e51c47e95904f4c0e9d52eddc21c1415ac"), S256().N),
			},
			decodeHex("1cadddc2838598fee7dc35a12b340c6bde8b389f7bfd19a1252a17c4b5ed2d" +
				"71c1a251bbecb14b058a8bd77f65de87e51c47e95904f4c0e9d52eddc21c1415ac"),
		},
		{
			"valid 4 - s is bigger than half order",
			&Signature{
				R: fromHex("a196ed0e7ebcbe7b63fe1d8eecbdbde03a67ceba4fc8f6482bdcb9606a911404"),
				S: fromHex("971729c7fa944b465b35250c6570a2f31acbb14b13d1565fab7330dcb2b3dfb1"),
			},
			decodeHex("a196ed0e7ebcbe7b63fe1d8eecbdbde03a67ceba4fc8f6482bdcb9606a9114" +
				"0468e8d638056bb4b9a4cadaf39a8f5d0b9fe32b9b9b7749dc145f2db01d826190"),
		},
		{
			"zero signature",
			&Signature{
				R: big.NewInt(0),
				S: big.NewInt(0),
			},
			decodeHex("00000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000000000"),
		},
	}

	for i, test := range tests {
		result := test.ecsig.Serialize()
		if !bytes.Equal(result, test.expected) {
			t.Errorf("Serialize #%d (%s) unexpected result:\n"+
				"got:  %x\nwant: %x", i, test.name, result,
				test.expected)
		}
	}
}

func testSignCompact(t *testing.T, tag string, curve *KoblitzCurve,
	data []byte, isCompressed bool) {
	tmp, _ := NewPrivateKey(curve)
	priv := (*PrivateKey)(tmp)

	hashed := []byte("testing")
	sig, err := SignCompact(curve, priv, hashed, isCompressed)
	if err != nil {
		t.Errorf("%s: error signing: %s", tag, err)
		return
	}

	pk, wasCompressed, err := RecoverCompact(curve, sig, hashed)
	if err != nil {
		t.Errorf("%s: error recovering: %s", tag, err)
		return
	}
	if pk.X.Cmp(priv.X) != 0 || pk.Y.Cmp(priv.Y) != 0 {
		t.Errorf("%s: recovered pubkey doesn't match original "+
			"(%v,%v) vs (%v,%v) ", tag, pk.X, pk.Y, priv.X, priv.Y)
		return
	}
	if wasCompressed != isCompressed {
		t.Errorf("%s: recovered pubkey doesn't match compressed state "+
			"(%v vs %v)", tag, isCompressed, wasCompressed)
		return
	}

	// If we change the compressed bit we should get the same key back,
	// but the compressed flag should be reversed.
	if isCompressed {
		sig[0] -= 4
	} else {
		sig[0] += 4
	}

	pk, wasCompressed, err = RecoverCompact(curve, sig, hashed)
	if err != nil {
		t.Errorf("%s: error recovering (2): %s", tag, err)
		return
	}
	if pk.X.Cmp(priv.X) != 0 || pk.Y.Cmp(priv.Y) != 0 {
		t.Errorf("%s: recovered pubkey (2) doesn't match original "+
			"(%v,%v) vs (%v,%v) ", tag, pk.X, pk.Y, priv.X, priv.Y)
		return
	}
	if wasCompressed == isCompressed {
		t.Errorf("%s: recovered pubkey doesn't match reversed "+
			"compressed state (%v vs %v)", tag, isCompressed,
			wasCompressed)
		return
	}
}

func TestSignCompact(t *testing.T) {
	for i := 0; i < 256; i++ {
		name := fmt.Sprintf("test %d", i)
		data := make([]byte, 32)
		_, err := rand.Read(data)
		if err != nil {
			t.Errorf("failed to read random data for %s", name)
			continue
		}
		compressed := i%2 != 0
		testSignCompact(t, name, S256(), data, compressed)
	}
}

// recoveryTests assert basic tests for public key recovery from signatures.
// The cases are borrowed from github.com/fjl/btcec-issue.
var recoveryTests = []struct {
	msg string
	sig string
	pub string
	err error
}{
	{
		// Valid curve point recovered.
		msg: "ce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce971008",
		sig: "0190f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc93",
		pub: "04E32DF42865E97135ACFB65F3BAE71BDC86F4D49150AD6A440B6F15878109880A0A2B2667F7E725CEEA70C673093BF67663E0312623C8E091B13CF2C0F11EF652",
	},
	{
		// Invalid curve point recovered.
		msg: "00c547e4f7b0f325ad1e56f57e26c745b09a3e503d86e00e5255ff7f715d3d1c",
		sig: "0100b1693892219d736caba55bdb67216e485557ea6b6af75f37096c9aa6a5a75f00b940b1d03b21e36b0e47e79769f095fe2ab855bd91e3a38756b7d75a9c4549",
		err: fmt.Errorf("invalid square root"),
	},
	{
		// Low R and S values.
		msg: "ba09edc1275a285fb27bfe82c4eea240a907a0dbaf9e55764b8f318c37d5974f",
		sig: "00000000000000000000000000000000000000000000000000000000000000002c0000000000000000000000000000000000000000000000000000000000000004",
		pub: "04A7640409AA2083FDAD38B2D8DE1263B2251799591D840653FB02DBBA503D7745FCB83D80E08A1E02896BE691EA6AFFB8A35939A646F1FC79052A744B1C82EDC3",
	},
}

func TestRecoverCompact(t *testing.T) {
	for i, test := range recoveryTests {
		msg := decodeHex(test.msg)
		sig := decodeHex(test.sig)

		// Magic DER constant.
		sig[0] += 27

		pub, _, err := RecoverCompact(S256(), sig, msg)

		// Verify that returned error matches as expected.
		if !reflect.DeepEqual(test.err, err) {
			t.Errorf("unexpected error returned from pubkey "+
				"recovery #%d: wanted %v, got %v",
				i, test.err, err)
			continue
		}

		// If check succeeded because a proper error was returned, we
		// ignore the returned pubkey.
		if err != nil {
			continue
		}

		// Otherwise, ensure the correct public key was recovered.
		exPub, _ := ParsePubKey(decodeHex(test.pub), S256())
		if !exPub.IsEqual(pub) {
			t.Errorf("unexpected recovered public key #%d: "+
				"want %v, got %v", i, exPub, pub)
		}
	}
}

func TestRFC6979(t *testing.T) {
	// Test vectors matching Trezor and CoreBitcoin implementations, but with signatures under our encoding scheme.
	// - https://github.com/trezor/trezor-crypto/blob/9fea8f8ab377dc514e40c6fd1f7c89a74c1d8dc6/tests.c#L432-L453
	// - https://github.com/oleganza/CoreBitcoin/blob/e93dd71207861b5bf044415db5fa72405e7d8fbc/CoreBitcoin/BTCKey%2BTests.m#L23-L49
	tests := []struct {
		key       string
		msg       string
		nonce     string
		signature string
	}{
		{
			"cca9fbcc1b41e5a95d369eaa6ddcff73b61a4efaa279cfc6567e8daa39cbaf50",
			"sample",
			"2df40ca70e639d89528a6b670d9d48d9165fdc0febc0974056bdce192b8e16a3",
			"af340daf02cc15c8d5d08d7735dfe6b98a474ed373bdb5fbecf7571be52b38425009fb27f37034a9b24b707b7c6b79ca23ddef9e25f7282e8a797efe53a8f124",
		},
		{
			// This signature hits the case when S is higher than halforder.
			// If S is not canonicalized (lowered by halforder), this test will fail.
			"0000000000000000000000000000000000000000000000000000000000000001",
			"Satoshi Nakamoto",
			"8f8a276c19f4149656b280621e358cce24f5f52542772691ee69063b74f15d15",
			"934b1ea10a4b3c1757e2b0c017d0b6143ce3c9a7e6a4a49860d7a6ab210ee3d82442ce9d2b916064108014783e923ec36b49743e2ffa1c4496f01a512aafd9e5",
		},
		{
			"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140",
			"Satoshi Nakamoto",
			"33a19b60e25fb6f4435af53a3d42d493644827367e6453928554f43e49aa6f90",
			"fd567d121db66e382991534ada77a6bd3106f0a1098c231e47993447cd6af2d06b39cd0eb1bc8603e159ef5c20a5c8ad685a45b06ce9bebed3f153d10d93bed5",
		},
		{
			"f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181",
			"Alan Turing",
			"525a82b70e67874398067543fd84c83d30c175fdc45fdeee082fe13b1d7cfdf1",
			"7063ae83e7f62bbb171798131b4a0564b956930092b33b07b395615d9ec7e15c58dfcc1e00a35e1572f366ffe34ba0fc47db1e7189759b9fb233c5b05ab388ea",
		},
		{
			"0000000000000000000000000000000000000000000000000000000000000001",
			"All those moments will be lost in time, like tears in rain. Time to die...",
			"38aa22d72376b4dbc472e06c3ba403ee0a394da63fc58d88686c611aba98d6b3",
			"8600dbd41e348fe5c9465ab92d23e3db8b98b873beecd930736488696438cb6b547fe64427496db33bf66019dacbf0039c04199abb0122918601db38a72cfc21",
		},
		{
			"e91671c46231f833a6406ccbea0e3e392c76c167bac1cb013f6f1013980455c2",
			"There is a computer disease that anybody who works with computers knows about. It's a very serious disease and it interferes completely with the work. The trouble with computers is that you 'play' with them!",
			"1f4b84c23a86a221d233f2521be018d9318639d5b8bbd6374a8a59232d16ad3d",
			"b552edd27580141f3b2a5463048cb7cd3e047b97c9f98076c32dbdf85a68718b279fa72dd19bfae05577e06c7c0c1900c371fcd5893f7e1d56a37d30174671f6",
		},
	}

	for i, test := range tests {
		privKey, _ := PrivKeyFromBytes(S256(), decodeHex(test.key))
		hash := sha256.Sum256([]byte(test.msg))

		// Ensure deterministically generated nonce is the expected value.
		gotNonce := nonceRFC6979(privKey.D, hash[:]).Bytes()
		wantNonce := decodeHex(test.nonce)
		if !bytes.Equal(gotNonce, wantNonce) {
			t.Errorf("NonceRFC6979 #%d (%s): Nonce is incorrect: "+
				"%x (expected %x)", i, test.msg, gotNonce,
				wantNonce)
			continue
		}

		// Ensure deterministically generated signature is the expected value.
		gotSig, err := privKey.Sign(hash[:])
		if err != nil {
			t.Errorf("Sign #%d (%s): unexpected error: %v", i,
				test.msg, err)
			continue
		}
		gotSigBytes := gotSig.Serialize()
		wantSigBytes := decodeHex(test.signature)
		if !bytes.Equal(gotSigBytes, wantSigBytes) {
			t.Errorf("Sign #%d (%s): mismatched signature: %x "+
				"(expected %x)", i, test.msg, gotSigBytes,
				wantSigBytes)
			continue
		}
	}
}

func TestSignatureIsEqual(t *testing.T) {
	sig1 := &Signature{
		R: fromHex("0082235e21a2300022738dabb8e1bbd9d19cfb1e7ab8c30a23b0afbb8d178abcf3"),
		S: fromHex("24bf68e256c534ddfaf966bf908deb944305596f7bdcc38d69acad7f9c868724"),
	}
	sig2 := &Signature{
		R: fromHex("4e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd41"),
		S: fromHex("181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d09"),
	}

	if !sig1.IsEqual(sig1) {
		t.Fatalf("value of IsEqual is incorrect, %v is "+
			"equal to %v", sig1, sig1)
	}

	if sig1.IsEqual(sig2) {
		t.Fatalf("value of IsEqual is incorrect, %v is not "+
			"equal to %v", sig1, sig2)
	}
}
