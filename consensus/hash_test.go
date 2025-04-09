// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestBytesToHash(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	hash := BytesToHash(data)
	if !bytes.Equal(hash[HashLength-len(data):], data) {
		t.Errorf("BytesToHash failed, got %x, want %x at the end", hash, data)
	}
}

func TestBytesToHashOverflow(t *testing.T) {
	data := make([]byte, HashLength+10)
	for i := 0; i < len(data); i++ {
		data[i] = byte(i)
	}
	hash := BytesToHash(data)
	if !bytes.Equal(hash[:], data[len(data)-HashLength:]) {
		t.Errorf("BytesToHash overflow handling failed")
	}
}

func TestHexToHash(t *testing.T) {
	hexStr := "0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"
	hash := HexToHash(hexStr)
	if hash.Hex() != hexStr {
		t.Errorf("HexToHash failed, got %s, want %s", hash.Hex(), hexStr)
	}
}

func TestHashBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	hash := BytesToHash(data)
	if !bytes.Equal(hash.Bytes(), hash[:]) {
		t.Errorf("Hash.Bytes failed")
	}
}

func TestHashBig(t *testing.T) {
	hexStr := "0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"
	hash := HexToHash(hexStr)
	expected := new(big.Int).SetBytes(hash[:])
	if hash.Big().Cmp(expected) != 0 {
		t.Errorf("Hash.Big failed")
	}
}

func TestHashHex(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	hash := BytesToHash(data)
	if hash.Hex() != hexutil.Encode(hash[:]) {
		t.Errorf("Hash.Hex failed")
	}
}

func TestHashTerminalString(t *testing.T) {
	hash := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	expected := fmt.Sprintf("%x…%x", hash[:3], hash[29:])
	if hash.TerminalString() != expected {
		t.Errorf("Hash.TerminalString failed, got %s, want %s", hash.TerminalString(), expected)
	}
}

func TestHashString(t *testing.T) {
	hash := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	if hash.String() != hash.Hex() {
		t.Errorf("Hash.String failed, got %s, want %s", hash.String(), hash.Hex())
	}
}

func TestHashFormat(t *testing.T) {
	hash := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	s := fmt.Sprintf("%x", hash)
	expected := fmt.Sprintf("%x", hash[:])
	if s != expected {
		t.Errorf("Hash.Format failed, got %s, want %s", s, expected)
	}
}

func TestHashUnmarshalText(t *testing.T) {
	hashStr := "0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102"
	var hash Hash
	err := hash.UnmarshalText([]byte(hashStr))
	if err != nil {
		t.Errorf("Hash.UnmarshalText failed with error: %v", err)
	}
	if hash.Hex() != hashStr {
		t.Errorf("Hash.UnmarshalText failed, got %s, want %s", hash.Hex(), hashStr)
	}
}

func TestHashUnmarshalJSON(t *testing.T) {
	hashStr := "\"0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102\""
	var hash Hash
	err := hash.UnmarshalJSON([]byte(hashStr))
	if err != nil {
		t.Errorf("Hash.UnmarshalJSON failed with error: %v", err)
	}

	expected := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	if hash != expected {
		t.Errorf("Hash.UnmarshalJSON failed, got %s, want %s", hash.Hex(), expected.Hex())
	}
}

func TestHashMarshalText(t *testing.T) {
	hash := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	text, err := hash.MarshalText()
	if err != nil {
		t.Errorf("Hash.MarshalText failed with error: %v", err)
	}

	expected, _ := hexutil.Bytes(hash[:]).MarshalText()
	if !bytes.Equal(text, expected) {
		t.Errorf("Hash.MarshalText failed, got %s, want %s", string(text), string(expected))
	}
}

func TestHashSetBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	var hash Hash
	hash.SetBytes(data)
	if !bytes.Equal(hash[HashLength-len(data):], data) {
		t.Errorf("Hash.SetBytes failed, got %x, want %x at the end", hash, data)
	}
}

func TestHashSetBytesOverflow(t *testing.T) {
	data := make([]byte, HashLength+10)
	for i := 0; i < len(data); i++ {
		data[i] = byte(i)
	}
	var hash Hash
	hash.SetBytes(data)
	if !bytes.Equal(hash[:], data[len(data)-HashLength:]) {
		t.Errorf("Hash.SetBytes overflow handling failed")
	}
}

func TestZeroHash(t *testing.T) {
	var expected Hash
	if Zero != expected {
		t.Errorf("Zero hash not initialized correctly")
	}
}

func TestHashesType(t *testing.T) {
	hashes := make(Hashes, 2)
	hashes[0] = HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	hashes[1] = HexToHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")

	if len(hashes) != 2 {
		t.Errorf("Hashes slice implementation failed")
	}
}

func TestHashesSetType(t *testing.T) {
	hashesSet := make(HashesSet)
	hash1 := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	hash2 := HexToHash("0x0202030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0103")

	hashesSet[hash1] = struct{}{}
	hashesSet[hash2] = struct{}{}

	if len(hashesSet) != 2 {
		t.Errorf("HashesSet map implementation failed")
	}

	if _, exists := hashesSet[hash1]; !exists {
		t.Errorf("HashesSet doesn't contain expected hash")
	}
}

func TestHashJSONMarshaling(t *testing.T) {
	hash := HexToHash("0x0102030405060708090a0b0c0d0e0f0102030405060708090a0b0c0d0e0f0102")
	data, err := json.Marshal(hash)
	if err != nil {
		t.Errorf("Failed to marshal hash: %v", err)
	}

	var hash2 Hash
	err = json.Unmarshal(data, &hash2)
	if err != nil {
		t.Errorf("Failed to unmarshal hash: %v", err)
	}

	if hash != hash2 {
		t.Errorf("JSON marshal/unmarshal failed, got %s, want %s", hash2.Hex(), hash.Hex())
	}
}
