package dag_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/msaldanha/setinstone/address"
	"github.com/msaldanha/setinstone/internal/dag"
	"github.com/msaldanha/setinstone/internal/datastore"
	"github.com/msaldanha/setinstone/internal/resolver"
	"github.com/msaldanha/setinstone/internal/util"
)

const defaultBranch = "main"

var _ = Describe("Dag", func() {

	var da dag.Dag
	var genesisNode *dag.Node
	var genesisAddr *address.Address
	var ctx context.Context
	var lts datastore.DataStore
	var res resolver.Resolver

	testGenesisNode, testGenesisAddr := CreateGenesisNode()

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()
		res = resolver.NewLocalResolver()
		_ = res.Manage(testGenesisAddr)

		genesisNode, genesisAddr = testGenesisNode, testGenesisAddr

		da = dag.NewDag("test-ledger", lts, res)
	})

	It("Should initialize with the Genesis node", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		key, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, err := da.Get(ctx, key)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should NOT initialize with the same Genesis node twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		_, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		_, err = da.SetRoot(ctx, genesisNode)
		Expect(err).NotTo(BeNil())
		Expect(err).To(Equal(dag.ErrDagAlreadyInitialized))
	})

	It("Should return the genesis node for an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, nodeKey, err := da.GetRoot(ctx, genesisNode.Address)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(nodeKey).To(Equal(genesisKey))
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should return the last node for default branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node, nodeKey, err := da.GetLast(ctx, genesisKey, defaultBranch)
		Expect(err).To(BeNil())
		Expect(node).NotTo(BeNil())
		Expect(nodeKey).To(Equal(genesisKey))
		Expect(node.Address).To(Equal(genesisAddr.Address))
	})

	It("Should add node to the default branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		key, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node := CreateNode(testGenesisAddr, key, key, defaultBranch, genesisNode.Seq+1)
		nodeKey, err := da.Append(ctx, node, key)
		Expect(err).To(BeNil())

		node2, err := da.Get(ctx, nodeKey)
		Expect(err).To(BeNil())
		Expect(node2).NotTo(BeNil())

		Expect(node).To(Equal(node2))
	})

	It("Should add node to a branch", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		g, gAddr := CreateGenesisNode()
		err := res.Manage(gAddr)
		Expect(err).To(BeNil())

		genesisKey, err := da.SetRoot(ctx, g)
		Expect(err).To(BeNil())

		// create main branch
		prev := genesisKey
		for x := 1; x <= 4; x++ {
			n := CreateNode(gAddr, genesisKey, prev, defaultBranch, g.Seq+int32(x))
			nodeKey, err := da.Append(ctx, n, genesisKey)
			Expect(err).To(BeNil())
			prev = nodeKey
		}

		// add one node with other branches
		nodeWithBranches := CreateNodeWithBranches(gAddr, genesisKey, prev, []string{"likes", "comments"}, defaultBranch, 6)
		nodeWithBranchesKey, err := da.Append(ctx, nodeWithBranches, genesisKey)
		Expect(err).To(BeNil())

		// add more nodes to main branch
		prev = nodeWithBranchesKey
		var lastMainBranch *dag.Node
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, genesisKey, prev, defaultBranch, nodeWithBranches.Seq+int32(x))
			nodeKey, err := da.Append(ctx, n, genesisKey)
			Expect(err).To(BeNil())
			prev = nodeKey
			lastMainBranch = n
		}

		// add nodes to the likes branch of the nodeWithBranches node
		prev = nodeWithBranchesKey
		var lastLikes *dag.Node
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, nodeWithBranchesKey, prev, "likes", int32(x))
			nodeKey, err := da.Append(ctx, n, nodeWithBranchesKey)
			Expect(err).To(BeNil())
			prev = nodeKey
			lastLikes = n
		}

		// add nodes to the comments branch of the nodeWithBranches node
		prev = nodeWithBranchesKey
		var lastComments *dag.Node
		for x := 1; x <= 5; x++ {
			n := CreateNode(gAddr, nodeWithBranchesKey, prev, "comments", int32(x))
			nodeKey, err := da.Append(ctx, n, nodeWithBranchesKey)
			Expect(err).To(BeNil())
			prev = nodeKey
			lastComments = n
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

		Expect(nodeWithBranches.Seq).To(Equal(int32(6)))

		n, _, err := da.GetLast(ctx, nodeWithBranchesKey, "likes")
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastLikes))
		Expect(n.Seq).To(Equal(int32(5)))

		n, _, err = da.GetLast(ctx, nodeWithBranchesKey, "comments")
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastComments))
		Expect(n.Seq).To(Equal(int32(5)))

		n, _, err = da.GetLast(ctx, genesisKey, g.Branch)
		Expect(err).To(BeNil())
		Expect(n).NotTo(BeNil())
		Expect(n).To(Equal(lastMainBranch))
		Expect(n.Seq).To(Equal(int32(11)))

	})

	It("Should NOT register node with invalid address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node := CreateNode(testGenesisAddr, genesisKey, genesisKey, defaultBranch, genesisNode.Seq+1)

		t := *node
		t.Address = "xxxxxxxxxx"
		_, err = da.Append(ctx, &t, genesisKey)
		Expect(err).To(Equal(address.ErrInvalidChecksum))
	})

	It("Should NOT register node with invalid timestamp", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		t := &dag.Node{}
		t = BuildNode(t, testGenesisAddr)
		_, err = da.Append(ctx, t, genesisKey)
		Expect(err).To(Equal(dag.ErrInvalidNodeTimestamp))
	})

	It("Should NOT register tampered node (hash)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node := CreateNode(testGenesisAddr, genesisKey, genesisKey, defaultBranch, genesisNode.Seq+1)
		node2 := CreateNode(testGenesisAddr, genesisKey, genesisKey, defaultBranch, genesisNode.Seq+1)

		t := *node
		t.Data = node2.Data
		_, err = da.Append(ctx, &t, genesisKey)
		Expect(err).To(Equal(dag.ErrNodeSignatureDoesNotMatch))
	})

	It("Should NOT register tampered node (signature)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node := CreateNode(testGenesisAddr, genesisKey, genesisKey, defaultBranch, genesisNode.Seq+1)

		t := *node
		t.Signature = t.Signature + "3e"
		_, err = da.Append(ctx, &t, genesisKey)
		Expect(err).To(Equal(dag.ErrNodeSignatureDoesNotMatch))
	})

	It("Should NOT register node twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node := CreateNode(testGenesisAddr, genesisKey, genesisKey, defaultBranch, genesisNode.Seq+1)

		_, err = da.Append(ctx, node, genesisKey)
		Expect(err).To(BeNil())
		_, err = da.Append(ctx, node, genesisKey)
		Expect(err).To(Equal(dag.ErrPreviousNodeIsNotHead))
	})

	It("Should NOT register node if previous node does not exists", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node := CreateNode(testGenesisAddr, genesisKey, "somekey", defaultBranch, genesisNode.Seq+1)

		_, err = da.Append(ctx, node, genesisKey)
		Expect(err).To(Equal(dag.ErrPreviousNodeNotFound))
	})

	It("Should NOT register node if previous node is not head", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		genesisKey, err := da.SetRoot(ctx, genesisNode)
		Expect(err).To(BeNil())

		node1 := CreateNode(genesisAddr, genesisKey, genesisKey, defaultBranch, 2)
		node1Key, err := da.Append(ctx, node1, genesisKey)
		Expect(err).To(BeNil())

		node2 := CreateNode(genesisAddr, genesisKey, node1Key, defaultBranch, 3)
		_, err = da.Append(ctx, node2, genesisKey)
		Expect(err).To(BeNil())

		node := CreateNode(genesisAddr, genesisKey, node1Key, defaultBranch, 4)
		_, err = da.Append(ctx, node, genesisKey)
		Expect(err).To(Equal(dag.ErrPreviousNodeIsNotHead))
	})
})

func CreateGenesisNode() (*dag.Node, *address.Address) {
	addr, _ := address.NewAddressWithKeys()

	genesisNode := CreateNode(addr, "", "", defaultBranch, 1)
	genesisNode.Address = addr.Address
	genesisNode.Branches = []string{defaultBranch}
	genesisNode.PubKey = addr.Keys.PublicKey

	_ = genesisNode.Sign(addr.Keys.ToEcdsaPrivateKey())

	return genesisNode, addr
}

func CreateNode(addr *address.Address, keyRoot, prev string, branch string, seq int32) *dag.Node {
	node := &dag.Node{}

	if prev != "" {
		node.Previous = prev
		node.Seq = seq
	} else {
		node.Seq = 1
	}
	node.Address = addr.Address
	node.PubKey = addr.Keys.PublicKey
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Data = []byte(util.RandString(256))
	node.Branch = branch
	node.BranchRoot = keyRoot

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}

func CreateNodeWithBranches(addr *address.Address, keyRoot, prev string, branches []string, branch string, seq int32) *dag.Node {
	node := &dag.Node{}

	if prev != "" {
		node.Previous = prev
		node.Seq = seq
	} else {
		node.Seq = 1
	}
	node.Address = addr.Address
	node.PubKey = addr.Keys.PublicKey
	node.Timestamp = time.Now().UTC().Format(time.RFC3339)
	node.Data = []byte(util.RandString(256))
	node.Branches = branches
	node.Branch = branch
	node.BranchRoot = keyRoot

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}

func BuildNode(node *dag.Node, addr *address.Address) *dag.Node {
	node.Address = addr.Address
	node.PubKey = addr.Keys.PublicKey

	_ = node.Sign(addr.Keys.ToEcdsaPrivateKey())

	return node
}
