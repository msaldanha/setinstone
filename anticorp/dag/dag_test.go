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

var _ = Describe("Dag", func() {

	var da dag.Dag
	var node *dag.Node
	var node2 *dag.Node
	var genesisNode *dag.Node
	var genesisAddr *address.Address
	var ctx context.Context
	var lts datastore.DataStore

	testGenesisNode, testGenesisAddr := CreateGenesisNode()
	testNode := CreateNode(testGenesisAddr, testGenesisNode)
	testNode2 := CreateNode(testGenesisAddr, testNode)

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

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.GetNode(ctx, genesisNode.Hash)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should NOT initialize with the same Genesis node twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		err = da.Initialize(ctx, genesisNode)
		Expect(err).NotTo(BeNil())
		Expect(err).To(Equal(dag.ErrDagAlreadyInitialized))
	})

	It("Should return the genesis node for an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.GetGenesisNode(ctx, genesisNode.Address)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Hash).To(Equal(genesisNode.Hash))
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should return the last node for default branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.GetLastNodeForBranch(ctx, genesisNode, dag.DefaultBranch)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Hash).To(Equal(genesisNode.Hash))
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should add node to the default branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		err = da.AddNode(ctx, node, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(BeNil())

		node2, err := da.GetNode(ctx, node.Hash)
		Expect(err).To(BeNil())
		Expect(node2).NotTo(BeNil())

		Expect(node).To(Equal(node2))
	})

	It("Should add node to a branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		g, gAddr := CreateGenesisNode()

		err := da.Initialize(ctx, g)
		Expect(err).To(BeNil())

		prev := g
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev)
			err = da.AddNode(ctx, n, g.Hash, dag.DefaultBranch)
			Expect(err).To(BeNil())
			prev = n
		}

		nodeWithBranches := CreateNodeWithBranches(gAddr, prev, []string{"likes", "comments"})
		err = da.AddNode(ctx, nodeWithBranches, g.Hash, dag.DefaultBranch)
		Expect(err).To(BeNil())

		prev = nodeWithBranches
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev)
			err = da.AddNode(ctx, n, g.Hash, dag.DefaultBranch)
			Expect(err).To(BeNil())
			prev = n
		}

		prev = nodeWithBranches
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev)
			err = da.AddNode(ctx, n, nodeWithBranches.Hash, "likes")
			Expect(err).To(BeNil())
			prev = n
		}

		lastLikes := prev

		prev = nodeWithBranches
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, prev)
			err = da.AddNode(ctx, n, nodeWithBranches.Hash, "comments")
			Expect(err).To(BeNil())
			prev = n
		}

		lastComments := prev

		n, err := da.GetLastNodeForBranch(ctx, nodeWithBranches, "likes")
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastLikes))

		n, err = da.GetLastNodeForBranch(ctx, nodeWithBranches, "comments")
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastComments))
	})

	It("Should NOT register node with invalid address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := *node
		t.Address = "xxxxxxxxxx"
		err = da.AddNode(ctx, &t, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(address.ErrInvalidChecksum))
	})

	It("Should NOT register node with invalid timestamp", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := &dag.Node{}
		t = BuildNode(t, testGenesisAddr)
		err = da.AddNode(ctx, t, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(dag.ErrInvalidNodeTimestamp))
	})

	It("Should NOT register tampered node (hash)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := *node
		t.Hash = node2.Hash
		err = da.AddNode(ctx, &t, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(dag.ErrNodeSignatureDoesNotMatch))
	})

	It("Should NOT register tampered node (signature)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := *node
		t.Signature = t.Signature + "3e"
		err = da.AddNode(ctx, &t, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(dag.ErrNodeSignatureDoesNotMatch))
	})

	It("Should NOT register node twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		err = da.AddNode(ctx, node, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(BeNil())
		err = da.AddNode(ctx, node, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(dag.ErrNodeAlreadyInDag))
	})

	It("Should NOT register node if previous node does not exists", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		dt := mock.NewMockDataStore(mockCtrl)
		da = dag.NewDag("test-ledger", dt)

		dt.EXPECT().Get(gomock.Any(), gomock.Any()).
			Return(nil, datastore.ErrNotFound).Times(2)

		err := da.AddNode(ctx, node, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(dag.ErrPreviousNodeNotFound))
	})

	It("Should NOT register node if previous node is not head", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := da.Initialize(ctx, genesisNode)
		Expect(err).To(BeNil())

		node1 := CreateNode(genesisAddr, genesisNode)
		err = da.AddNode(ctx, node1, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(BeNil())

		node2 := CreateNode(genesisAddr, node1)
		err = da.AddNode(ctx, node2, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(BeNil())

		node := CreateNode(genesisAddr, node1)
		err = da.AddNode(ctx, node, genesisNode.Hash, dag.DefaultBranch)
		Expect(err).To(Equal(dag.ErrPreviousNodeIsNotHead))
	})
})

func CreateGenesisNode() (*dag.Node, *address.Address) {
	addr, _ := address.NewAddressWithKeys()

	genesisNode := CreateNode(addr, nil)
	genesisNode.Address = addr.Address
	genesisNode.Branches = []string{dag.DefaultBranch}
	genesisNode.PubKey = hex.EncodeToString(addr.Keys.PublicKey)

	_ = genesisNode.SetPow()

	_ = genesisNode.Sign(addr.Keys.ToEcdsaPrivateKey())

	return genesisNode, addr
}

func CreateNode(addr *address.Address, prev *dag.Node) *dag.Node {
	node := &dag.Node{}

	if prev != nil {
		node.Previous = prev.Hash
	}
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Data = []byte(randString(256))

	_ = node.SetPow()

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}

func CreateNodeWithBranches(addr *address.Address, prev *dag.Node, branches []string) *dag.Node {
	node := &dag.Node{}

	if prev != nil {
		node.Previous = prev.Hash
	}
	node.Address = addr.Address
	node.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Data = []byte(randString(256))
	node.Branches = branches

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
