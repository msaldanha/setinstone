package datachain_test

import (
	"context"
	"encoding/hex"
	"github.com/golang/mock/gomock"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/datachain"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/mock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Ledger", func() {

	var ld datachain.Ledger
	var tx *datachain.Transaction
	var tx2 *datachain.Transaction
	var genesisTx *datachain.Transaction
	var genesisAddr *address.Address
	var ctx context.Context
	var lts datastore.DataStore

	testGenesisTx, testGenesisAddr := CreateGenesisTransaction()
	testTx := CreateTransaction(datachain.TransactionTypes.Doc, testGenesisAddr, testGenesisTx)
	testTx2 := CreateTransaction(datachain.TransactionTypes.Doc, testGenesisAddr, testTx)

	BeforeEach(func() {
		ctx = context.Background()
		lts = datastore.NewLocalFileStore()

		genesisTx, genesisAddr = testGenesisTx, testGenesisAddr
		tx, tx2 = testTx, testTx2

		ld = datachain.NewLocalLedger("test-ledger", lts)
	})

	It("Should initialize with the Genesis transaction", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		tx, err := ld.GetTransaction(ctx, genesisTx.Address, genesisTx.Hash)
		Expect(err).To(BeNil())
		Expect(tx).NotTo(BeNil())
		Expect(tx.Address).To(Equal(genesisAddr.Address))
	})

	It("Should NOT initialize with the same Genesis transaction twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		err = ld.Initialize(ctx, genesisTx)
		Expect(err).NotTo(BeNil())
		Expect(err).To(Equal(datachain.ErrLedgerAlreadyInitialized))
	})

	It("Should NOT initialize with the other Genesis transaction", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		otherGenesis := CreateTransaction(datachain.TransactionTypes.Open, genesisAddr, nil)

		err = ld.Initialize(ctx, otherGenesis)
		Expect(err).NotTo(BeNil())
		Expect(err).To(Equal(datachain.ErrLedgerAlreadyInitialized))
	})

	It("Should return the genesis transaction for an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		tx, err := ld.GetGenesisTransaction(ctx, genesisTx.Address)
		Expect(err).To(BeNil())
		Expect(tx).NotTo(BeNil())
		Expect(tx.Hash).To(Equal(genesisTx.Hash))
		Expect(tx.Address).To(Equal(genesisAddr.Address))
	})

	It("Should return the last transaction for an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		tx, err := ld.GetLastTransaction(ctx, genesisTx.Address)
		Expect(err).To(BeNil())
		Expect(tx).NotTo(BeNil())
		Expect(tx.Hash).To(Equal(genesisTx.Hash))
		Expect(tx.Address).To(Equal(genesisAddr.Address))
	})

	It("Should register transaction", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		err = ld.Register(ctx, tx)
		Expect(err).To(BeNil())

		tx2, err := ld.GetTransaction(ctx, tx.Address, tx.Hash)
		Expect(err).To(BeNil())
		Expect(tx2).NotTo(BeNil())

		Expect(tx).To(Equal(tx2))
	})

	It("Should NOT register transaction with invalid address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		t := *tx
		t.Address = "xxxxxxxxxx"
		err = ld.Register(ctx, &t)
		Expect(err).To(Equal(address.ErrInvalidChecksum))
	})

	It("Should NOT register transaction with invalid timestamp", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		t := &datachain.Transaction{
			Type: datachain.TransactionTypes.Doc,
			Seq:  1,
		}
		t = BuildTransaction(t, testGenesisAddr)
		err = ld.Register(ctx, t)
		Expect(err).To(Equal(datachain.ErrInvalidTransactionTimestamp))
	})

	It("Should NOT register transaction with invalid seq", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		t := &datachain.Transaction{
			Type:      datachain.TransactionTypes.Doc,
			Seq:       100,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Previous:genesisTx.Hash,
		}
		t = BuildTransaction(t, testGenesisAddr)
		err = ld.Register(ctx, t)
		Expect(err).To(Equal(datachain.ErrInvalidTransactionSeq))
	})

	It("Should NOT register tampered transaction (hash)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		t := *tx
		t.Hash = tx2.Hash
		err = ld.Register(ctx, &t)
		Expect(err).To(Equal(datachain.ErrTransactionSignatureDoesNotMatch))
	})

	It("Should NOT register tampered transaction (signature)", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		t := *tx
		t.Signature = t.Signature + "3e"
		err = ld.Register(ctx, &t)
		Expect(err).To(Equal(datachain.ErrTransactionSignatureDoesNotMatch))
	})

	It("Should NOT register transaction twice", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		err = ld.Register(ctx, tx)
		Expect(err).To(BeNil())
		err = ld.Register(ctx, tx)
		Expect(err).To(Equal(datachain.ErrTransactionAlreadyInLedger))
	})

	It("Should NOT register transaction if previous tx does not exists", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		dt := mock.NewMockDataStore(mockCtrl)
		ld = datachain.NewLocalLedger("test-ledger", dt)

		dt.EXPECT().Get(gomock.Any(), gomock.Any()).
			Return(nil, datastore.ErrNotFound).Times(2)

		err := ld.Register(ctx, tx)
		Expect(err).To(Equal(datachain.ErrPreviousTransactionNotFound))
	})

	It("Should NOT register transaction if previous tx is not head", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		tx1 := CreateTransaction(datachain.TransactionTypes.Doc, genesisAddr, genesisTx)
		err = ld.Register(ctx, tx1)
		Expect(err).To(BeNil())

		tx2 := CreateTransaction(datachain.TransactionTypes.Doc, genesisAddr, tx1)
		err = ld.Register(ctx, tx2)
		Expect(err).To(BeNil())

		tx := CreateTransaction(datachain.TransactionTypes.Reference, genesisAddr, tx1)
		err = ld.Register(ctx, tx)
		Expect(err).To(Equal(datachain.ErrPreviousTransactionIsNotHead))
	})

	It("Should return the list of transactions for an address", func() {
		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		err := ld.Initialize(ctx, genesisTx)
		Expect(err).To(BeNil())

		err = ld.Register(ctx, tx)
		Expect(err).To(BeNil())

		err = ld.Register(ctx, tx2)
		Expect(err).To(BeNil())

		txs, err := ld.GetAddressStatement(ctx, genesisTx.Address)
		Expect(err).To(BeNil())
		Expect(txs).NotTo(BeNil())
		Expect(len(txs)).To(Equal(3))
		Expect(txs[2].Hash).To(Equal(genesisTx.Hash))
		Expect(txs[2].Address).To(Equal(genesisAddr.Address))
		Expect(txs[1].Hash).To(Equal(tx.Hash))
		Expect(txs[1].Address).To(Equal(tx.Address))
		Expect(txs[0].Hash).To(Equal(tx2.Hash))
		Expect(txs[0].Address).To(Equal(tx2.Address))
	})
})

func CreateGenesisTransaction() (*datachain.Transaction, *address.Address) {
	addr, _ := address.NewAddressWithKeys()

	genesisTx := CreateTransaction(datachain.TransactionTypes.Open, addr, nil)
	genesisTx.Address = addr.Address
	genesisTx.PubKey = hex.EncodeToString(addr.Keys.PublicKey)

	_ = genesisTx.SetPow()

	_ = genesisTx.Sign(addr.Keys.ToEcdsaPrivateKey())

	return genesisTx, addr
}

func CreateTransaction(ty string, addr *address.Address, prev *datachain.Transaction) *datachain.Transaction {
	tx := &datachain.Transaction{
		Type: ty,
	}

	if prev != nil {
		tx.Seq = prev.Seq + 1
		tx.Previous = prev.Hash
	}
	tx.Address = addr.Address
	tx.PubKey = hex.EncodeToString(addr.Keys.PublicKey)
	tx.Timestamp = time.Now().UTC().Format(time.RFC3339)

	_ = tx.SetPow()

	_ = tx.Sign(addr.Keys.ToEcdsaPrivateKey())

	return tx
}

func BuildTransaction(tx *datachain.Transaction, addr *address.Address) *datachain.Transaction {
	tx.Address = addr.Address
	tx.PubKey = hex.EncodeToString(addr.Keys.PublicKey)

	_ = tx.SetPow()

	_ = tx.Sign(addr.Keys.ToEcdsaPrivateKey())

	return tx
}
