package graph

import (
	"encoding/hex"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"time"
)

func createNode(data []byte, branches []string, prev *dag.Node,
	addr *address.Address) (*dag.Node, error) {
	node := dag.NewNode()
	node.Data = data
	if prev != nil {
		node.Previous = prev.Hash
	}
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Branches = branches
	er := node.SetPow()
	if er != nil {
		return nil, er
	}
	er = node.Sign(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return nil, er
	}
	return node, nil
}
