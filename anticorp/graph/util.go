package graph

import (
	"encoding/hex"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"time"
)

func createNode(node NodeData, prev *dag.Node,
	addr *address.Address) (*dag.Node, error) {
	n := dag.NewNode()
	n.Data = node.Data
	if prev != nil {
		n.Previous = prev.Hash
		n.BranchSeq = prev.BranchSeq + 1
	} else {
		n.BranchSeq = 1
	}
	n.Address = addr.Address
	n.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	n.Timestamp = time.Now().UTC().Format(time.RFC3339)
	n.Branches = node.Branches
	n.Branch = node.Branch
	er := n.SetPow()
	if er != nil {
		return nil, er
	}
	er = n.Sign(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return nil, er
	}
	return n, nil
}
