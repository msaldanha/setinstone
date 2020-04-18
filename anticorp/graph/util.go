package graph

import (
	"encoding/hex"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"time"
)

func createNode(data []byte, branch string, branches []string, prev *dag.Node,
	addr *address.Address) (*dag.Node, error) {
	node := dag.NewNode()
	node.Data = data
	if prev != nil {
		node.Previous = prev.Hash
		node.BranchSeq = prev.BranchSeq + 1
	} else {
		node.BranchSeq = 1
	}
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Branches = branches
	node.Branch = branch
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