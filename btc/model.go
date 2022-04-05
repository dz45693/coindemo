package btc

import (
	"context"
	"crypto/sha256"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/shopspring/decimal"
	"math/big"
)

type Block struct {
	Block        *wire.MsgBlock
	Transactions []*TransactionInfo
}

type TransactionSate int32

const (
	TransactionSateDefault TransactionSate = 0
	TransactionSateSuccess TransactionSate = 1
	TransactionSateFail    TransactionSate = 2
	TransactionSatePending TransactionSate = 3
)

type TransactionInfo struct {
	State         TransactionSate
	BlockHeight   int32
	TxId          string
	Value         decimal.Decimal
	N             uint32
	Asm           string
	Hex           string
	ReqSigs       int32
	Type          string
	Address       []string
	BlockHash     string
	Confirmations uint64
	Time          int64
	BlockTime     int64
	Fee           decimal.Decimal
}

type RawTransactionInfo struct {
	State       TransactionSate
	BlockHeight int32
	BlockHash   string
	Fee         decimal.Decimal
	Result      *btcjson.TxRawResult
}

type TransactionOutPut struct {
	TXId  string
	Value decimal.Decimal
	N     uint32
	Hex   string
}

type Server interface {
	ValidateAddress(ctx context.Context, address string) bool
	Block(ctx context.Context, index int64) (block *Block, err error)
	CurrentBlockHeight(ctx context.Context) (int64, error)
	CreateAddressByPubKey(ctx context.Context, publicKey string) (string, error)
	Transaction(ctx context.Context, txId string) (transaction *RawTransactionInfo, err error)
	CreateTx(ctx context.Context, from, to string, amount, fee decimal.Decimal, utxOS TransactionOutPut) (*wire.MsgTx, []byte, error)
	SigTx(ctx context.Context, publicKey string, r, s *big.Int, redeemTx *wire.MsgTx) (*wire.MsgTx, error)
	BroadcastTx(ctx context.Context, tx *wire.MsgTx) (string, error)
}

const CheckSumLength = 4

func CheckSumHash(versionPublicHash []byte) []byte {
	versionPublicHash1 := sha256.Sum256(versionPublicHash)
	versionPublicHash2 := sha256.Sum256(versionPublicHash1[:])
	tailHash := versionPublicHash2[:CheckSumLength]
	return tailHash
}
