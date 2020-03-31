package ethapi

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

// PublicAccountAPI

func TestPublicAccountAPI_Accounts(t *testing.T) {
	b := NewTestBackend()

	api := NewPublicAccountAPI(b.AM)
	assert.NotPanics(t, func() {
		api.Accounts()
	})
}

// PrivateAccountAPI

func TestPrivateAccountAPI_DeriveAccount(t *testing.T) {
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	assert.NotPanics(t, func() {
		api.DeriveAccount("https://test.ru", "/test", nil)
	})
}
func TestPrivateAccountAPI_ImportRawKey(t *testing.T) {
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	assert.NotPanics(t, func() {
		res, err := api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
func TestPrivateAccountAPI_ListAccounts(t *testing.T) {
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	assert.NotPanics(t, func() {
		api.ListAccounts()
	})
}
func TestPrivateAccountAPI_ListWallets(t *testing.T) {
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	assert.NotPanics(t, func() {
		api.ListWallets()
	})
}
func TestPrivateAccountAPI_NewAccount(t *testing.T) {
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	assert.NotPanics(t, func() {
		res, err := api.NewAccount("1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
func TestPrivateAccountAPI_UnlockAccount(t *testing.T) {
	ctx := context.TODO()
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	addr, _ := api.NewAccount("1234")
	api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")

	assert.NotPanics(t, func() {
		d := uint64(1)
		res, err := api.UnlockAccount(ctx, addr, "1234", &d)
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
func TestPrivateAccountAPI_LockAccount(t *testing.T) {
	ctx := context.TODO()
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	addr, _ := api.NewAccount("1234")
	_, _ = api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")
	d := uint64(1)
	_, _ = api.UnlockAccount(ctx, addr, "1234", &d)

	assert.NotPanics(t, func() {
		api.LockAccount(addr)
	})
}
func TestPrivateAccountAPI_Sign(t *testing.T) {
	ctx := context.TODO()
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	addr, _ := api.NewAccount("1234")
	key, _ := api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")
	d := uint64(1)
	_, _ = api.UnlockAccount(ctx, key, "1234", &d)

	assert.NotPanics(t, func() {
		res, err := api.Sign(ctx, hexutil.Bytes([]byte{1, 2, 3}), addr, "1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
func TestPrivateAccountAPI_SignTransaction(t *testing.T) {
	ctx := context.TODO()
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	addr, _ := api.NewAccount("1234")
	key, _ := api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")
	d := uint64(1)
	_, _ = api.UnlockAccount(ctx, key, "1234", &d)

	assert.NotPanics(t, func() {
		gas := hexutil.Uint64(0)
		gasPrice := hexutil.Big(*big.NewInt(0))
		nonce := hexutil.Uint64(1)
		res, err := api.SignTransaction(ctx, SendTxArgs{
			From:     addr,
			To:       &common.Address{1},
			Gas:      &gas,
			GasPrice: &gasPrice,
			Nonce:    &nonce,
		}, "1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
func TestPrivateAccountAPI_SignAndSendTransaction(t *testing.T) {
	ctx := context.TODO()
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	addr, _ := api.NewAccount("1234")
	key, _ := api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")
	d := uint64(1)
	_, _ = api.UnlockAccount(ctx, key, "1234", &d)

	assert.NotPanics(t, func() {
		gas := hexutil.Uint64(0)
		gasPrice := hexutil.Big(*big.NewInt(0))
		nonce := hexutil.Uint64(1)
		res, err := api.SignAndSendTransaction(ctx, SendTxArgs{
			From:     addr,
			To:       &common.Address{1},
			Gas:      &gas,
			GasPrice: &gasPrice,
			Nonce:    &nonce,
		}, "1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
func TestPrivateAccountAPI_SendTransaction(t *testing.T) {
	ctx := context.TODO()
	b := NewTestBackend()

	nonceLock := new(AddrLocker)
	api := NewPrivateAccountAPI(b, nonceLock)
	api.am = b.AM

	addr, _ := api.NewAccount("1234")
	key, _ := api.ImportRawKey("11223344556677889900aabbccddff0011223344556677889900aabbccddff00", "1234")
	d := uint64(1)
	_, _ = api.UnlockAccount(ctx, key, "1234", &d)

	assert.NotPanics(t, func() {
		gas := hexutil.Uint64(0)
		gasPrice := hexutil.Big(*big.NewInt(0))
		res, err := api.SendTransaction(ctx, SendTxArgs{
			From:     addr,
			To:       &common.Address{1},
			Gas:      &gas,
			GasPrice: &gasPrice,
		}, "1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
