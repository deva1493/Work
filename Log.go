package main

import (
	"crypto/sha256"
	"log"
	"os"

	"github.com/wilfreddenton/merkle"
)

func main() {
	// create a hash function
	// it's ok to resuse this in merkle calls because it will be reset after use
	h := sha256.New()

	// create an io.Reader
	f, err := os.Open("BT.go")
	if err != nil {
		log.Fatal(err)
	}

	// shard the file into segments (1024 byte sized segments in this case)
	preLeaves, err := merkle.Shard(f, 1024)
	if err != nil {
		log.Fatal(err)
	}

	// initialize the tree
	t := merkle.NewTree()

	// compute the root hash from the pre-leaves using the sha256 hash function
	err = t.Hash(preLeaves, h)
	if err != nil {
		log.Fatal(err)
	}

	// use the LeafHash function to convert a pre-leaf into a leaf
	leaf := merkle.LeafHash(preLeaves[0], h)

	// create the merkle path for the leaf
	path := t.MerklePath(leaf)
	if path == nil {
		log.Fatalf("tree does not contain %x", leaf)
	}

	// prove with the path that the tree contains the leaf
	if !merkle.Prove(leaf, t.Root(), path, h) {
		log.Fatalf("tree should container %x", leaf)
	}
}
