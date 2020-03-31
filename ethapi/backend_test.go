package ethapi

import (
	"context"
	"math/big"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	notify "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-lachesis/evmcore"
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/inter/pos"
	"github.com/Fantom-foundation/go-lachesis/inter/sfctype"
	"github.com/Fantom-foundation/go-lachesis/kvdb/nokeyiserr"
	"github.com/Fantom-foundation/go-lachesis/kvdb/table"
)

func method() string {
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		parts := strings.Split(details.Name(), ".")
		name := parts[len(parts)-1]
		// fmt.Printf("Method name = '%s'\n", name)
		return name
	}
	return ""
}

func TestMethod(t *testing.T) {
	assert.Equal(t, method(), "TestMethod")
}

func TestGetAPIs(t *testing.T) {
	b := NewTestBackend()

	assert.NotPanics(t, func() {
		res := GetAPIs(b)
		assert.NotEmpty(t, res)
	})
}

type testBackend struct {
	StateDB *state.StateDB
	AM      *accounts.Manager
	result  struct {
		returned map[string][]interface{}
		err      map[string]error
		panic    map[string]string
	}
}

func NewTestBackend() *testBackend {
	b := &testBackend{
		result: struct {
			returned map[string][]interface{}
			err      map[string]error
			panic    map[string]string
		}{
			returned: make(map[string][]interface{}),
			err:      make(map[string]error),
			panic:    make(map[string]string),
		},
	}
	b.PrepareMethods()
	return b
}

func (b *testBackend) PrepareMethods() {
	b.Returned("GetTd", big.NewInt(1))
	b.Returned("SuggestPrice", big.NewInt(1))
	b.Returned("GetPoolNonce", uint64(1))
	b.Returned("RPCGasCap", big.NewInt(1))
	b.Returned("Stats", 2, 2)
	b.Returned("ProtocolVersion", 1)
	b.Returned("GetBlock", &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:     big.NewInt(1),
			Hash:       common.Hash{2},
			ParentHash: common.Hash{3},
			Root:       common.Hash{4},
			TxHash:     common.Hash{5},
			Time:       6,
			Coinbase:   common.Address{7},
			GasLimit:   8,
			GasUsed:    9,
		},
		Transactions: types.Transactions{
			types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{}),
		},
	})
	b.Returned("BlockByNumber", &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:     big.NewInt(1),
			Hash:       common.Hash{2},
			ParentHash: common.Hash{3},
			Root:       common.Hash{4},
			TxHash:     common.Hash{5},
			Time:       6,
			Coinbase:   common.Address{7},
			GasLimit:   8,
			GasUsed:    9,
		},
		Transactions: types.Transactions{
			types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{}),
		},
	})
	b.Returned("TxPoolContent",
		map[common.Address]types.Transactions{
			common.Address{1}: types.Transactions{
				types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{}),
				types.NewTransaction(2, common.Address{2}, big.NewInt(2), 2, big.NewInt(0), []byte{}),
			},
		},
		map[common.Address]types.Transactions{
			common.Address{1}: types.Transactions{
				types.NewTransaction(3, common.Address{3}, big.NewInt(3), 3, big.NewInt(0), []byte{}),
				types.NewTransaction(4, common.Address{4}, big.NewInt(4), 4, big.NewInt(0), []byte{}),
			},
		},
	)
	b.Returned("GetPoolTransactions", types.Transactions{
		types.NewTransaction(3, common.Address{3}, big.NewInt(3), 3, big.NewInt(0), []byte{}),
		types.NewTransaction(4, common.Address{4}, big.NewInt(4), 4, big.NewInt(0), []byte{}),
	},
	)
	b.Returned("HeaderByNumber", &evmcore.EvmHeader{
		Number: big.NewInt(1),
	})
	b.Returned("HeaderByHash", &evmcore.EvmHeader{
		Number: big.NewInt(1),
	})
	b.Returned("ChainConfig", &params.ChainConfig{
		ChainID: big.NewInt(1),
	})
	ts := inter.Timestamp(time.Now().Add(-91 * time.Minute).UnixNano())
	b.Returned("Progress", PeerProgress{
		CurrentEpoch:     1,
		CurrentBlock:     2,
		CurrentBlockHash: hash.Event{3},
		CurrentBlockTime: ts,
		HighestBlock:     5,
		HighestEpoch:     6,
	})

	// Set state DB
	db1 := rawdb.NewDatabase(
		nokeyiserr.Wrap(
			table.New(
				memorydb.New(), []byte("evm1_"))))
	b.StateDB, _ = state.New(common.HexToHash("0x0"), state.NewDatabase(db1))
	b.Returned("StateAndHeaderByNumber", b.StateDB, &evmcore.EvmHeader{})
	b.StateDB.SetNonce(common.Address{1}, 1)
	b.StateDB.AddBalance(common.Address{1}, big.NewInt(10))
	b.StateDB.SetCode(common.Address{1}, []byte{1, 2, 3})

	// Set EVM
	vmCtx := vm.Context{}
	evm := vm.NewEVM(vmCtx, b.StateDB, &params.ChainConfig{}, vm.Config{})
	b.Returned("GetEVM", evm, func() error { return nil })

	b.Returned("Wallets", []accounts.Wallet{
		&testWallet{},
		&testWallet{},
		&testWallet{},
	})

	b.Returned("Subscribe", notify.NewSubscription(func(c <-chan struct{}) error { return nil }))

	// Prepare keystore
	keyStore := keystore.NewKeyStore("/tmp", 2, 2)
	for _, ac := range keyStore.Accounts() {
		keyStore.Delete(ac, "1234")
	}

	// Prepare account manager
	b.AM = accounts.NewManager(&accounts.Config{InsecureUnlockAllowed: true}, b, keyStore)
	b.Returned("AccountManager", b.AM)

	// GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, uint64, uint64, error)
	b.Returned("GetTransaction",
		types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{}),
		uint64(1), uint64(1))

	rec1 := types.NewReceipt([]byte{}, false, 100)
	rec1.PostState = []byte{1, 2, 3}
	rec1.ContractAddress = common.Address{1}
	rec2 := types.NewReceipt([]byte{}, false, 100)
	b.Returned("GetReceiptsByNumber", types.Receipts{
		rec1,
		rec2,
	})

	b.Returned("Subscribe", &testAccountSubscription{})
	b.Returned("CurrentEpoch", idx.Epoch(1))
	b.Returned("GetConsensusTime", inter.Timestamp(1))
	b.Returned("GetEpochStats", &sfctype.EpochStats{
		Start: 1,
		End:   2,
		Epoch: 1,
	})
	b.Returned("GetEvent", &inter.Event{
		EventHeader: inter.EventHeader{
			EventHeaderData: inter.EventHeaderData{
				Version:       1,
				Epoch:         2,
				Seq:           1,
				Frame:         1,
				IsRoot:        true,
				Creator:       1,
				PrevEpochHash: common.Hash{0},
				Parents:       nil,
				GasPowerLeft:  inter.GasPowerLeft{},
				GasPowerUsed:  0,
				Lamport:       0,
				ClaimedTime:   0,
				MedianTime:    0,
				TxHash:        common.Hash{},
				Extra:         nil,
			},
			Sig: nil,
		},
		Transactions: types.Transactions{
			types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{}),
		},
	})
	b.Returned("GetEventHeader", &inter.EventHeaderData{
		Version:       1,
		Epoch:         2,
		Seq:           1,
		Frame:         1,
		IsRoot:        true,
		Creator:       1,
		PrevEpochHash: common.Hash{0},
		Parents:       nil,
		GasPowerLeft:  inter.GasPowerLeft{},
		GasPowerUsed:  0,
		Lamport:       0,
		ClaimedTime:   0,
		MedianTime:    0,
		TxHash:        common.Hash{},
		Extra:         nil,
	})
	b.Returned("GetHeads", hash.Events{
		hash.Event{1},
	})

	b.Returned("GetDelegator", &sfctype.SfcDelegator{
		CreatedEpoch:     1,
		CreatedTime:      2,
		DeactivatedEpoch: 0,
		DeactivatedTime:  0,
		Amount:           nil,
		ToStakerID:       1,
	})
	b.Returned("GetDelegatorClaimedRewards", big.NewInt(1))
	b.Returned("GetDelegatorsOf", []sfctype.SfcDelegatorAndAddr{
		{
			Delegator: &sfctype.SfcDelegator{
				CreatedEpoch:     1,
				CreatedTime:      2,
				DeactivatedEpoch: 0,
				DeactivatedTime:  0,
				Amount:           nil,
				ToStakerID:       1,
			},
			Addr: common.Address{1},
		},
	})
	b.Returned("GetDowntime", idx.Block(1), inter.Timestamp(1))
	b.Returned("GetOriginationScore", big.NewInt(1))
	b.Returned("GetRewardWeights", big.NewInt(1), big.NewInt(1))
	b.Returned("GetStaker", &sfctype.SfcStaker{
		CreatedEpoch:     1,
		CreatedTime:      1,
		DeactivatedEpoch: 0,
		DeactivatedTime:  0,
		Address:          common.Address{1},
		StakeAmount:      big.NewInt(1),
		DelegatedMe:      big.NewInt(0),
	})
	b.Returned("GetStakerPoI", big.NewInt(1))
	b.Returned("GetValidationScore", big.NewInt(1))
	b.Returned("GetStakerClaimedRewards", big.NewInt(1))
	b.Returned("GetStakerDelegatorsClaimedRewards", big.NewInt(1))
	b.Returned("GetStakers", []sfctype.SfcStakerAndID{
		{
			StakerID: 1,
			Staker: &sfctype.SfcStaker{
				CreatedEpoch:     1,
				CreatedTime:      1,
				DeactivatedEpoch: 0,
				DeactivatedTime:  0,
				Address:          common.Address{1},
				StakeAmount:      big.NewInt(1),
				DelegatedMe:      big.NewInt(0),
			},
		},
	})
	b.Returned("GetStakerID", idx.StakerID(1))
	b.Returned("TtfReport", map[hash.Event]time.Duration{
		hash.HexToEventHash("0x1"): time.Second,
		hash.HexToEventHash("0x2"): 2 * time.Second,
		hash.HexToEventHash("0x3"): 3 * time.Second,
		hash.HexToEventHash("0x4"): 4 * time.Second,
	})
	b.Returned("ValidatorTimeDrifts", map[idx.StakerID]map[hash.Event]time.Duration{
		idx.StakerID(1): {
			hash.HexToEventHash("0x1"): time.Second,
			hash.HexToEventHash("0x2"): 2 * time.Second,
		},
		idx.StakerID(2): {
			hash.HexToEventHash("0x3"): 3 * time.Second,
			hash.HexToEventHash("0x4"): 4 * time.Second,
		},
	})

	db2 := rawdb.NewDatabase(
		nokeyiserr.Wrap(
			table.New(
				memorydb.New(), []byte("evm2_"))))
	b.Returned("ChainDb", db2)
	b.Returned("ExtRPCEnabled", false)
}

func (b *testBackend) Returned(method string, args ...interface{}) {
	b.result.err[method] = nil
	b.result.panic[method] = ""
	b.result.returned[method] = make([]interface{}, len(args), len(args))
	for i, v := range args {
		b.result.returned[method][i] = v
	}
}

func (b *testBackend) Error(method string, err error) {
	b.result.err[method] = err
}

func (b *testBackend) Panic(method string, msg string) {
	b.result.panic[method] = msg
}

func (b *testBackend) checkPanic(method string) {
	if b.result.panic[method] != "" {
		panic(b.result.panic[method])
	}
}

func (b *testBackend) ProtocolVersion() int {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(int)
}
func (b *testBackend) Progress() PeerProgress {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(PeerProgress)
}
func (b *testBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) ChainDb() ethdb.Database {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(ethdb.Database)
}
func (b *testBackend) AccountManager() *accounts.Manager {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*accounts.Manager)
}
func (b *testBackend) ExtRPCEnabled() bool {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(bool)
}
func (b *testBackend) RPCGasCap() *big.Int {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int)
} // global gas cap for eth_call over rpc: DoS protection

// Blockchain API
func (b *testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*evmcore.EvmHeader), b.result.err[method]
}
func (b *testBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeader, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*evmcore.EvmHeader), b.result.err[method]
}
func (b *testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmBlock, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*evmcore.EvmBlock), b.result.err[method]
}
func (b *testBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *evmcore.EvmHeader, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*state.StateDB),
		b.result.returned[method][1].(*evmcore.EvmHeader),
		b.result.err[method]
}

//GetHeader(ctx context.Context, hash common.Hash) *evmcore.EvmHeader
func (b *testBackend) GetBlock(ctx context.Context, hash common.Hash) (*evmcore.EvmBlock, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*evmcore.EvmBlock), b.result.err[method]
}
func (b *testBackend) GetReceiptsByNumber(ctx context.Context, number rpc.BlockNumber) (types.Receipts, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(types.Receipts), b.result.err[method]
}
func (b *testBackend) GetTd(hash common.Hash) *big.Int {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int)
}
func (b *testBackend) GetEVM(ctx context.Context, msg evmcore.Message, state *state.StateDB, header *evmcore.EvmHeader) (*vm.EVM, func() error, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*vm.EVM), b.result.returned[method][1].(func() error), b.result.err[method]
}

// Transaction pool API
func (b *testBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	method := method()
	b.checkPanic(method)
	return b.result.err[method]
}
func (b *testBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, uint64, uint64, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*types.Transaction),
		b.result.returned[method][1].(uint64),
		b.result.returned[method][2].(uint64),
		b.result.err[method]
}
func (b *testBackend) GetPoolTransactions() (types.Transactions, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(types.Transactions), b.result.err[method]
}
func (b *testBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*types.Transaction)
}
func (b *testBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(uint64), b.result.err[method]
}
func (b *testBackend) Stats() (pending int, queued int) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(int), b.result.returned[method][1].(int)
}
func (b *testBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(map[common.Address]types.Transactions),
		b.result.returned[method][1].(map[common.Address]types.Transactions)
}
func (b *testBackend) SubscribeNewTxsNotify(chan<- evmcore.NewTxsNotify) notify.Subscription {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(notify.Subscription)
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*params.ChainConfig)
}
func (b *testBackend) CurrentBlock() *evmcore.EvmBlock {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*evmcore.EvmBlock)
}

// Lachesis DAG API
func (b *testBackend) GetEvent(ctx context.Context, shortEventID string) (*inter.Event, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*inter.Event), b.result.err[method]
}
func (b *testBackend) GetEventHeader(ctx context.Context, shortEventID string) (*inter.EventHeaderData, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*inter.EventHeaderData), b.result.err[method]
}
func (b *testBackend) GetConsensusTime(ctx context.Context, shortEventID string) (inter.Timestamp, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(inter.Timestamp), b.result.err[method]
}
func (b *testBackend) GetHeads(ctx context.Context, epoch rpc.BlockNumber) (hash.Events, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(hash.Events), b.result.err[method]
}
func (b *testBackend) CurrentEpoch(ctx context.Context) idx.Epoch {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(idx.Epoch)
}
func (b *testBackend) GetEpochStats(ctx context.Context, requestedEpoch rpc.BlockNumber) (*sfctype.EpochStats, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*sfctype.EpochStats), b.result.err[method]
}
func (b *testBackend) TtfReport(ctx context.Context, untilBlock rpc.BlockNumber, maxBlocks idx.Block, mode string) (map[hash.Event]time.Duration, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(map[hash.Event]time.Duration), b.result.err[method]
}
func (b *testBackend) ForEachEvent(ctx context.Context, epoch rpc.BlockNumber, onEvent func(event *inter.Event) bool) error {
	method := method()
	b.checkPanic(method)
	onEvent(&inter.Event{
		EventHeader: inter.EventHeader{
			EventHeaderData: inter.EventHeaderData{
				Version: 1,
				Epoch:   2,
				Seq:     1,
				Frame:   0,
				IsRoot:  false,
			},
			Sig: nil,
		},
		Transactions: nil,
	})
	return b.result.err[method]
}
func (b *testBackend) ValidatorTimeDrifts(ctx context.Context, epoch rpc.BlockNumber, maxEvents idx.Event) (map[idx.StakerID]map[hash.Event]time.Duration, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(map[idx.StakerID]map[hash.Event]time.Duration), b.result.err[method]
}

// Lachesis SFC API
func (b *testBackend) GetValidators(ctx context.Context) *pos.Validators {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*pos.Validators)
}
func (b *testBackend) GetValidationScore(ctx context.Context, stakerID idx.StakerID) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetOriginationScore(ctx context.Context, stakerID idx.StakerID) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetRewardWeights(ctx context.Context, stakerID idx.StakerID) (*big.Int, *big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.returned[method][1].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetStakerPoI(ctx context.Context, stakerID idx.StakerID) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetDowntime(ctx context.Context, stakerID idx.StakerID) (idx.Block, inter.Timestamp, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(idx.Block), b.result.returned[method][1].(inter.Timestamp), b.result.err[method]
}
func (b *testBackend) GetDelegatorClaimedRewards(ctx context.Context, addr common.Address) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetStakerClaimedRewards(ctx context.Context, stakerID idx.StakerID) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetStakerDelegatorsClaimedRewards(ctx context.Context, stakerID idx.StakerID) (*big.Int, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*big.Int), b.result.err[method]
}
func (b *testBackend) GetStaker(ctx context.Context, stakerID idx.StakerID) (*sfctype.SfcStaker, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*sfctype.SfcStaker), b.result.err[method]
}
func (b *testBackend) GetStakerID(ctx context.Context, addr common.Address) (idx.StakerID, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(idx.StakerID), b.result.err[method]
}
func (b *testBackend) GetStakers(ctx context.Context) ([]sfctype.SfcStakerAndID, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].([]sfctype.SfcStakerAndID), b.result.err[method]
}
func (b *testBackend) GetDelegatorsOf(ctx context.Context, stakerID idx.StakerID) ([]sfctype.SfcDelegatorAndAddr, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].([]sfctype.SfcDelegatorAndAddr), b.result.err[method]
}
func (b *testBackend) GetDelegator(ctx context.Context, addr common.Address) (*sfctype.SfcDelegator, error) {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(*sfctype.SfcDelegator), b.result.err[method]
}
func (b *testBackend) Wallets() []accounts.Wallet {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].([]accounts.Wallet)
}
func (b *testBackend) Subscribe(sink chan<- accounts.WalletEvent) notify.Subscription {
	method := method()
	b.checkPanic(method)
	return b.result.returned[method][0].(notify.Subscription)
}

type testWallet struct{}

func (w *testWallet) URL() accounts.URL {
	return accounts.URL{
		Scheme: "https",
		Path:   "test.ru/test",
	}
}

func (w *testWallet) Status() (string, error) {
	return "ok", nil
}

func (w *testWallet) Open(passphrase string) error {
	return nil
}

func (w *testWallet) Close() error {
	return nil
}
func (w *testWallet) Accounts() []accounts.Account {
	return []accounts.Account{
		accounts.Account{
			Address: common.Address{1},
			URL:     w.URL(),
		},
	}
}

func (w *testWallet) Contains(account accounts.Account) bool {
	return true
}

func (w *testWallet) Derive(path accounts.DerivationPath, pin bool) (accounts.Account, error) {
	return w.Accounts()[0], nil
}

func (w *testWallet) SelfDerive(bases []accounts.DerivationPath, chain ethereum.ChainStateReader) {
}

func (w *testWallet) SignData(account accounts.Account, mimeType string, data []byte) ([]byte, error) {
	return make([]byte, 128, 128), nil
}

func (w *testWallet) SignDataWithPassphrase(account accounts.Account, passphrase, mimeType string, data []byte) ([]byte, error) {
	return make([]byte, 128, 128), nil
}

func (w *testWallet) SignText(account accounts.Account, text []byte) ([]byte, error) {
	return make([]byte, 128, 128), nil
}

func (w *testWallet) SignTextWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return make([]byte, 128, 128), nil
}

func (w *testWallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	trx := types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{})
	return trx, nil
}

func (w *testWallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	trx := types.NewTransaction(1, common.Address{1}, big.NewInt(1), 1, big.NewInt(0), []byte{})
	return trx, nil
}

type testAccountSubscription struct{}

func (s *testAccountSubscription) Err() <-chan error {
	ch := make(chan error)
	return ch
}
func (s *testAccountSubscription) Unsubscribe() {}
