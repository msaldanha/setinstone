package dag_test

import (
	"context"
	"encoding/hex"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"math/rand"
	"strings"
	"time"
)

const defaultBranch = "main"

var _ = Describe("Dag", func() {

	var da dag.Dag
	var node *dag.Node
	var node2 *dag.Node
	var genesisNode *dag.Node
	var genesisAddr *address.Address
	var ctx context.Context
	var lts datastore.DataStore

	testGenesisNode, testGenesisAddr := CreateGenesisNode()
	testNode := CreateNode(testGenesisAddr, testGenesisNode, defaultBranch, testGenesisNode.BranchSeq+1)
	testNode2 := CreateNode(testGenesisAddr, testNode, defaultBranch, testNode.BranchSeq+1)

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()

		genesisNode, genesisAddr = testGenesisNode, testGenesisAddr
		node, node2 = testNode, testNode2

		da = dag.NewDag("test-ledger", lts)
	})

	It("Should initialize with the Genesis node", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.Get(ctx, genesisNode.Hash)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should NOT initialize with the same Genesis node twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		err = da.SetRoot(ctx, genesisNode)
		Expect(err).NotTo(BeNil())
		Expect(err).To(Equal(dag.ErrDagAlreadyInitialized))
	})

	It("Should return the genesis node for an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.GetRoot(ctx, genesisNode.Address)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Hash).To(Equal(genesisNode.Hash))
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should return the last node for default branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.GetLast(ctx, genesisNode.Hash, defaultBranch)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Hash).To(Equal(genesisNode.Hash))
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should add node to the default branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		err = da.Append(ctx, node, genesisNode.Hash)
		Expect(err).To(BeNil())

		node2, err := da.Get(ctx, node.Hash)
		Expect(err).To(BeNil())
		Expect(node2).NotTo(BeNil())

		Expect(node).To(Equal(node2))
	})

	It("Should add node to a branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		g, gAddr := CreateGenesisNode()

		err := da.SetRoot(ctx, g)
		Expect(err).To(BeNil())

		// create main branch
		prev := g
		for x := 1; x <= 4; x++ {
			n := CreateNode(gAddr, prev, defaultBranch, prev.BranchSeq+1)
			err = da.Append(ctx, n, g.Hash)
			Expect(err).To(BeNil())
			prev = n
		}

		// add one node with other branches
		nodeWithBranches := CreateNodeWithBranches(gAddr, prev, []string{"likes", "comments"}, defaultBranch, prev.BranchSeq+1)
		err = da.Append(ctx, nodeWithBranches, g.Hash)
		Expect(err).To(BeNil())

		// add more nodes to main branch
		prev = nodeWithBranches
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev, defaultBranch, prev.BranchSeq+1)
			err = da.Append(ctx, n, g.Hash)
			Expect(err).To(BeNil())
			prev = n
		}

		lastMainBranch := prev

		// add nodes to the likes branch of the nodeWithBranches node
		prev = nodeWithBranches
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev, "likes", int32(x))
			err = da.Append(ctx, n, nodeWithBranches.Hash)
			Expect(err).To(BeNil())
			prev = n
		}

		lastLikes := prev

		// add nodes to the comments branch of the nodeWithBranches node
		prev = nodeWithBranches
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev, "comments", int32(x))
			err = da.Append(ctx, n, nodeWithBranches.Hash)
			Expect(err).To(BeNil())
			prev = n
		}

		// final graph should have the structure:
		//                      n  branch seq for this node should be 5
		//                      |
		//                      n
		//                      |
		//                      n
		//                      |
		//                      n
		//                      |
		//                      n  likes branch
		// root                 |
		// n - n - n - n - n - nodeWithBranches - n - n - n - n - n
		//                      |
		//                      n  comments branch
		//                      |
		//                      n
		//                      |
		//                      n
		//                      |
		//                      n
		//                      |
		//                      n branch seq for this node should be 5

		lastComments := prev

		Expect(nodeWithBranches.BranchSeq).To(Equal(int32(6)))

		n, err := da.GetLast(ctx, nodeWithBranches.Hash, "likes")
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastLikes))
		Expect(n.BranchSeq).To(Equal(int32(5)))

		n, err = da.GetLast(ctx, nodeWithBranches.Hash, "comments")
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastComments))
		Expect(n.BranchSeq).To(Equal(int32(5)))

		n, err = da.GetLast(ctx, g.Hash, g.Branch)
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastMainBranch))
		Expect(n.BranchSeq).To(Equal(int32(11)))

	})

	It("Should NOT register node with invalid address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := *node
		t.Address = "xxxxxxxxxx"
		err = da.Append(ctx, &t, genesisNode.Hash)
		Expect(err).To(Equal(address.ErrInvalidChecksum))
	})

	It("Should NOT register node with invalid timestamp", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := &dag.Node{}
		t = BuildNode(t, testGenesisAddr)
		err = da.Append(ctx, t, genesisNode.Hash)
		Expect(err).To(Equal(dag.ErrInvalidNodeTimestamp))
	})

	It("Should NOT register tampered node (hash)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := *node
		t.Hash = node2.Hash
		err = da.Append(ctx, &t, genesisNode.Hash)
		Expect(err).To(Equal(dag.ErrNodeSignatureDoesNotMatch))
	})

	It("Should NOT register tampered node (signature)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := *node
		t.Signature = t.Signature + "3e"
		err = da.Append(ctx, &t, genesisNode.Hash)
		Expect(err).To(Equal(dag.ErrNodeSignatureDoesNotMatch))
	})

	It("Should NOT register node twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		err = da.Append(ctx, node, genesisNode.Hash)
		Expect(err).To(BeNil())
		err = da.Append(ctx, node, genesisNode.Hash)
		Expect(err).To(Equal(dag.ErrNodeAlreadyInDag))
	})

	It("Should NOT register node if previous node does not exists", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		dt := mock.NewMockDataStore(mockCtrl)
		da = dag.NewDag("test-ledger", dt)

		dt.EXPECT().Get(gomock.Any(), gomock.Any()).
			Return(nil, datastore.ErrNotFound).Times(2)

		err := da.Append(ctx, node, genesisNode.Hash)
		Expect(err).To(Equal(dag.ErrPreviousNodeNotFound))
	})

	It("Should NOT register node if previous node is not head", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node1 := CreateNode(genesisAddr, genesisNode, defaultBranch, 2)
		err = da.Append(ctx, node1, genesisNode.Hash)
		Expect(err).To(BeNil())

		node2 := CreateNode(genesisAddr, node1, defaultBranch, 3)
		err = da.Append(ctx, node2, genesisNode.Hash)
		Expect(err).To(BeNil())

		node := CreateNode(genesisAddr, node1, defaultBranch, 4)
		err = da.Append(ctx, node, genesisNode.Hash)
		Expect(err).To(Equal(dag.ErrPreviousNodeIsNotHead))
	})
})

func CreateGenesisNode() (*dag.Node, *address.Address) {
	addr, _ := address.NewAddressWithKeys()

	genesisNode := CreateNode(addr, nil, defaultBranch, 1)
	genesisNode.Address = addr.Address
	genesisNode.Branches = []string{defaultBranch}
	genesisNode.PubKey = hex.EncodeToString(addr.Keys.PublicKey)

	_ = genesisNode.SetPow()

	_ = genesisNode.Sign(addr.Keys.ToEcdsaPrivateKey())

	return genesisNode, addr
}

func CreateNode(addr *address.Address, prev *dag.Node, branch string, seq int32) *dag.Node {
	node := &dag.Node{}

	if prev != nil {
		node.Previous = prev.Hash
		node.BranchSeq = seq
	} else {
		node.BranchSeq = 1
	}
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Data = []byte(randString(256))
	node.Branch = branch

	_ = node.SetPow()

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}

func CreateNodeWithBranches(addr *address.Address, prev *dag.Node, branches []string, branch string, seq int32) *dag.Node {
	node := &dag.Node{}

	if prev != nil {
		node.Previous = prev.Hash
		node.BranchSeq = seq
	} else {
		node.BranchSeq = 1
	}
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Data = []byte(randString(256))
	node.Branches = branches
	node.Branch = branch

	_ = node.SetPow()

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}

func BuildNode(node *dag.Node, addr *address.Address) *dag.Node {
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)

	_ = node.SetPow()

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}
func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	var src = rand.NewSource(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
