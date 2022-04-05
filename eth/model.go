package eth

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"time"
)

type ERC20Info struct {
	ContractAddress string
	Name            string
	Symbol          string
	Decimals        uint8
	TotalSupply     string
}

type CreateTransactionRequest struct {
	TokenAddress       string
	From               string
	To                 string
	Amount             string
	GasLimit           uint64
	GasMaxFee          string //wei
	GasTip             int32
	DisableEstimateGas bool
	Nonce              uint64
}

type BlockInfo struct {
	BlockNumber  uint64
	Time         time.Time
	Hash         string
	Transactions []*TransactionInfo
}

type TransactionInfo struct {
	ID           string
	BlockNumber  uint64
	Time         time.Time
	From         string
	To           string
	TokenAddress string
	Amount       decimal.Decimal
	State        TransactionSate
	Fee          decimal.Decimal
}

type TransactionSate int32

const (
	TransactionSateDefault TransactionSate = 0
	TransactionSateSuccess TransactionSate = 1
	TransactionSateFail    TransactionSate = 2
	TransactionSatePending TransactionSate = 3
)

type Server interface {
	Client() *ethclient.Client
	CreateAddress(ctx context.Context, mnemonic string, index uint32) (string, error)
	BalanceETH(ctx context.Context, address string) (*decimal.Decimal, error)
	BalanceERC20(ctx context.Context, tokenAddress, ownerAddress string) (*decimal.Decimal, error)
	ERC20Info(ctx context.Context, contractAddress string) (*ERC20Info, error)
	CreateTransaction(ctx context.Context, request CreateTransactionRequest) (*types.Transaction, error)
	SignTransaction(ctx context.Context, tx *types.Transaction, privateKey string) (*types.Transaction, error)
	Broadcast(ctx context.Context, tx *types.Transaction) error
	Block(ctx context.Context, number uint64) (*BlockInfo, error)
	Transaction(ctx context.Context, txID string) (*TransactionInfo, error)
	CurrentBlockHeight(ctx context.Context) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*decimal.Decimal, error)
	MaxFee(ctx context.Context, tip int32) (*decimal.Decimal, error)
	Nonce(ctx context.Context, fromAddress string) (uint64, error)
	SignerHash(ctx context.Context, tx *types.Transaction) ([]byte, error)
	WithSignature(ctx context.Context, tx *types.Transaction, signature []byte) (*types.Transaction, error)
}

/*------------------------------------------*/

/*-----------------------------------------------*/
type AppErr struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Status  codes.Code             `json:"status"`
	Details map[string]interface{} `json:"details"`
}

func (e AppErr) Error() string {
	return e.Message
}

func NewAppErr(code, message string) AppErr {
	return AppErr{Code: code, Message: message}
}

var (
	ErrNotFound               = &AppErr{Code: "NOT_FOUND", Message: "resource was not found", Status: codes.NotFound}
	ErrInvalidInput           = &AppErr{Code: "INVALID_INPUT", Message: "input is valid", Status: codes.InvalidArgument}
	ErrNotSupportTX           = &AppErr{Code: "NOT_SUPPORT_TX", Message: "the transaction is not support", Status: codes.InvalidArgument}
	ErrNotSupportContractType = &AppErr{Code: "NOT_SUPPORT_CONTRACT_TYPE", Message: "not support contract type", Status: codes.InvalidArgument}
)
