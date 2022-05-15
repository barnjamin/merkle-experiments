package main

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"hash"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

var (
	ErrRootMismatch = errors.New("root Mismatch")
	domainSep       = "MA"
	hashSize        = 32
)

func NewHasher() hash.Hash {
	//TODO: add more hashers
	return sha512.New512_256()
}

func Verify(root []byte, hash []byte, proof models.ProofResponse) error {
	layer := layerItem{hash: hash, pos: proof.Idx}
	pl := partialLayer{layer}

	var hints = make([][]byte, proof.Treedepth)
	for x := 0; x < int(proof.Treedepth); x++ {
		hints[x] = proof.Proof[x*hashSize : (x+1)*hashSize]
	}

	s := &siblings{hints: hints}

	for len(s.hints) > 0 {
		pl = pl.up(s.next())
	}

	computedroot := pl[0]
	if computedroot.pos != 0 || !bytes.Equal(computedroot.hash, root) {
		return ErrRootMismatch
	}
	return nil
}

func HashLayer(left, right []byte) []byte {
	buf := make([]byte, 2*hashSize)
	copy(buf[:], left[:])
	copy(buf[len(left):], right[:])
	return append([]byte(domainSep), buf[:]...)
}

type siblings struct {
	hints [][]byte
}

func (s *siblings) next() (res []byte) {
	res = s.hints[0]
	s.hints = s.hints[1:]
	return
}

type partialLayer []layerItem

type layerItem struct {
	pos  uint64
	hash []byte
}

func (pl partialLayer) up(siblingHash []byte) partialLayer {
	item := pl[0]

	var (
		left, right []byte
	)

	if item.pos&1 == 0 {
		left = item.hash
		right = siblingHash
	} else {
		left = siblingHash
		right = item.hash
	}

	nh := NewHasher()
	nh.Write(HashLayer(left, right))

	return partialLayer{{
		pos:  item.pos / 2,
		hash: nh.Sum(nil),
	}}
}
