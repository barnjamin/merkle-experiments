package main

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"hash"
	"log"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/protocol"
)

var (
	ErrRootMismatch = errors.New("Root Mismatch")
	domainSep       = "MA"
	hashSize        = 32
)

func NewHasher() hash.Hash {
	//TODO: add more hashers
	return sha512.New512_256()
}

func Verify(root []byte, hash []byte, proof models.ProofResponse) error {
	hasher := NewHasher()
	pl := partialLayer{{hash: hash, pos: proof.Idx}}

	var hints = make([]crypto.GenericDigest, proof.Treedepth)
	for x := 0; x < int(proof.Treedepth); x++ {
		hints[x] = proof.Proof[x*hashSize : (x+1)*hashSize]
	}

	s := &siblings{hints: hints}

	log.Printf("%+v", pl)
	var err error
	for l := uint64(0); len(s.hints) > 0 || len(pl) > 1; l++ {
		log.Printf("On Level: %+v %+v", l, pl)
		if pl, err = pl.up(s, l, hasher); err != nil {
			return err
		}
	}

	computedroot := pl[0]
	if computedroot.pos != 0 || !bytes.Equal(computedroot.hash, root) {
		return ErrRootMismatch
	}
	return nil
}

// A pair represents an internal node in the Merkle tree.
type pair struct {
	l              crypto.GenericDigest
	r              crypto.GenericDigest
	hashDigestSize int
}

func (p *pair) ToBeHashed() (protocol.HashID, []byte) {
	// hashing of internal node will always be fixed length.
	// If one of the children is missing we use [0...0].
	// The size of the slice is based on the relevant hash function output size
	buf := make([]byte, 2*p.hashDigestSize)
	copy(buf[:], p.l[:])
	copy(buf[len(p.l):], p.r[:])
	return protocol.HashID(domainSep), buf[:]
}

type siblings struct {
	hints []crypto.GenericDigest
}

func (s *siblings) get(l uint64, i uint64) (res crypto.GenericDigest, err error) {
	res = s.hints[0].ToSlice()
	s.hints = s.hints[1:]
	err = nil
	return
}

type partialLayer []layerItem

type layerItem struct {
	pos  uint64
	hash crypto.GenericDigest
}

func (pl partialLayer) up(s *siblings, l uint64, hsh hash.Hash) (partialLayer, error) {
	var res partialLayer

	for i := 0; i < len(pl); i++ {

		item := pl[i]
		pos := item.pos
		posHash := item.hash

		siblingPos := pos ^ 1

		var siblingHash crypto.GenericDigest

		var err error
		siblingHash, err = s.get(l, siblingPos)
		if err != nil {
			return nil, err
		}

		log.Printf("%+v %+v", l, siblingHash)

		nextLayerPos := pos / 2
		var nextLayerHash crypto.GenericDigest

		var p pair
		p.hashDigestSize = hsh.Size()
		if pos&1 == 0 {
			// We are left
			p.l = posHash
			p.r = siblingHash
		} else {
			// We are right
			p.l = siblingHash
			p.r = posHash
		}
		nextLayerHash = crypto.GenericHashObj(hsh, &p)

		res = append(res, layerItem{
			pos:  nextLayerPos,
			hash: nextLayerHash,
		})
	}

	return res, nil
}
