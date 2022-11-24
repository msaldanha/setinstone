package graph

import (
	"time"

	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
)

func createNode(node NodeData, keyRoot, prev string,
	addr *address.Address, seq int32) (*dag.Node, error) {
	n := dag.NewNode()
	n.Data = node.Data
	if prev != "" {
		n.Previous = prev
	}
	n.Seq = seq
	n.Address = addr.Address
	n.PubKey = addr.Keys.PublicKey
	n.Timestamp = time.Now().UTC().Format(time.RFC3339)
	n.Branches = node.Branches
	n.Branch = node.Branch
	n.BranchRoot = keyRoot
	er := n.Sign(addr.Keys.ToEcdsaPrivateKey())
	if er != nil {
		return nil, er
	}
	return n, nil
}
