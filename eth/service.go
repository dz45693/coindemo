package eth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
	"time"
)

var (
	decimal0         = decimal.NewFromInt(0)
	decimal18        = decimal.New(1, 18)
	decimal9         = decimal.New(1, 9)
	transferMethodId = "a9059cbb"
)

type Service struct {
	client                *ethclient.Client
	blockConfirmationNum  uint64
	eabi                  abi.ABI
	estimateGasMultiplier float64
}

func NewService(client *ethclient.Client, blockConfirmationNum uint64, estimateGasMultiplier float64) *Service {
	decimal.DivisionPrecision = 18
	eabi, _ := abi.JSON(strings.NewReader(TokenMetaData.ABI))
	return &Service{
		client:                client,
		blockConfirmationNum:  blockConfirmationNum,
		estimateGasMultiplier: estimateGasMultiplier,
		eabi:                  eabi,
	}
}

func (svc *Service) Client() *ethclient.Client {
	return svc.client
}

func (svc *Service) CurrentBlockHeight(ctx context.Context) (uint64, error) {
	return svc.client.BlockNumber(ctx)
}

func (svc *Service) Nonce(ctx context.Context, fromAddress string) (uint64, error) {
	return svc.client.PendingNonceAt(ctx, common.HexToAddress(fromAddress))
}

func (svc *Service) CreateAddress(ctx context.Context, mnemonic string, index uint32) (string, error) {
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("m/44'/60'/0'/0/%d", index)
	account, err := wallet.Derive(hdwallet.MustParseDerivationPath(path), false)
	if err != nil {
		return "", err
	}

	return wallet.AddressHex(account)
}

func (svc *Service) CreateAddressByPubKey(ctx context.Context, publicKey string) (string, error) {
	decodePublicKey, err := hexutil.Decode(publicKey)
	if err != nil {
		return "", err
	}

	publicKey1, err := crypto.DecompressPubkey(decodePublicKey)
	if err != nil {
		return "", err
	}

	return crypto.PubkeyToAddress(*publicKey1).Hex(), nil
}

func (svc *Service) BalanceETH(ctx context.Context, address string) (*decimal.Decimal, error) {
	hexAddress := common.HexToAddress(address)
	balance, err := svc.client.BalanceAt(ctx, hexAddress, nil)
	if err != nil {
		return nil, err
	}

	result, err := decimal.NewFromString(balance.String())
	if err != nil {
		return nil, err
	}

	result = result.Div(decimal18)
	return &result, nil
}

func (svc *Service) BalanceERC20(ctx context.Context, tokenAddr, ownerAddr string) (*decimal.Decimal, error) {
	result := decimal.Decimal{}
	erc20Token, err := svc.ERC20Info(ctx, tokenAddr)
	if err != nil {
		return nil, err
	}

	decimalPlaces := decimal.New(1, int32(erc20Token.Decimals))
	tokenAddress := common.HexToAddress(tokenAddr)
	ownerAddress := common.HexToAddress(ownerAddr)

	instance, err := NewToken(tokenAddress, svc.client)
	if err != nil {
		return nil, err
	}

	amount, err := instance.BalanceOf(&bind.CallOpts{}, ownerAddress)
	if err != nil {
		return nil, err
	}

	result, err = decimal.NewFromString(amount.String())
	if err != nil {
		return nil, err
	}

	result = result.Div(decimalPlaces)
	return &result, nil
}

func (svc *Service) ERC20Info(ctx context.Context, contractAddress string) (*ERC20Info, error) {
	var err error
	info := ERC20Info{ContractAddress: contractAddress}
	tokenAddress := common.HexToAddress(contractAddress)
	instance, err := NewToken(tokenAddress, svc.client)
	if err != nil {
		return nil, err
	}

	info.Name, err = instance.Name(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	info.Decimals, err = instance.Decimals(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	info.Symbol, err = instance.Symbol(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	totalSupply, err := instance.TotalSupply(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	info.TotalSupply = totalSupply.String()

	return &info, nil
}

func (svc *Service) SuggestGasPrice(ctx context.Context) (*decimal.Decimal, error) {
	gasPrice, err := svc.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	result, err := decimal.NewFromString(gasPrice.String())
	if err != nil {
		return nil, err
	}

	result = result.Div(decimal18)
	return &result, nil
}

func (svc *Service) MaxFee(ctx context.Context, tip int32) (*decimal.Decimal, error) {
	gasPrice, err := svc.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	tipString := new(big.Int).Mul(big.NewInt(int64(tip)), big.NewInt(params.GWei)).String()
	tipDecimal, err := decimal.NewFromString(tipString)
	if err != nil {
		return nil, err
	}

	tipDecimal = tipDecimal.Div(decimal18)
	result := gasPrice.Add(tipDecimal)
	return &result, nil
}

func (svc *Service) CreateTransaction(ctx context.Context, request CreateTransactionRequest) (*types.Transaction, error) {
	reqAmount, err := decimal.NewFromString(request.Amount)
	if err != nil {
		return nil, err
	}

	if reqAmount.LessThan(decimal0) {
		return nil, ErrInvalidInput
	}

	fromAddress := common.HexToAddress(request.From)
	toAddress := common.HexToAddress(request.To)

	maxFee, err := decimal.NewFromString(request.GasMaxFee)
	if err != nil {
		return nil, err
	}
	maxFee = maxFee.Mul(decimal18)

	chainID, err := svc.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Print(chainID)
	//  DynamicFeeTx
	if len(request.TokenAddress) < 1 {
		var data []byte
		tx := types.NewTransaction(request.Nonce, toAddress, reqAmount.Mul(decimal18).BigInt(), request.GasLimit, maxFee.BigInt(), data)
/*
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   chainID,
				Nonce:     request.Nonce,
				GasFeeCap: maxFee.BigInt(),
				GasTipCap: new(big.Int).Mul(big.NewInt(int64(request.GasTip)), big.NewInt(params.Wei)),
				Gas:       request.GasLimit,
				To:        &toAddress,
				Value:     reqAmount.Mul(decimal18).BigInt(),
				Data:      data,
				AccessList: nil,
			})
*/
		return tx, nil
	}

	tokenInfo, err := svc.ERC20Info(ctx, request.TokenAddress)
	if err != nil {
		return nil, err
	}

	tokenAddress := common.HexToAddress(request.TokenAddress)
	decimalPlace := decimal.New(1, int32(tokenInfo.Decimals))
	amount := reqAmount.Mul(decimalPlace)

	data, err := svc.eabi.Pack("transfer", toAddress, amount.BigInt())
	if err != nil {
		return nil, err
	}

	if !request.DisableEstimateGas {
		request.GasLimit, err = svc.client.EstimateGas(ctx, ethereum.CallMsg{
			From:  fromAddress,
			To:    &tokenAddress,
			Value: big.NewInt(0),
			Data:  data,
		})
		if err != nil {
			return nil, err
		}

		request.GasLimit = uint64(float64(request.GasLimit) * svc.estimateGasMultiplier)
	}
	tx := types.NewTransaction(request.Nonce, toAddress, reqAmount.Mul(decimal18).BigInt(), request.GasLimit, maxFee.BigInt(), data)
	/*
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     request.Nonce,
			GasFeeCap: maxFee.BigInt(),
			GasTipCap: new(big.Int).Mul(big.NewInt(int64(request.GasTip)), big.NewInt(params.Wei)),
			Gas:       request.GasLimit,
			To:        &tokenAddress,
			Value:     big.NewInt(0),
			Data:      data,
		})

	*/
	return tx, nil
}

func (svc *Service) SignTransaction(ctx context.Context, tx *types.Transaction, privateKey string) (*types.Transaction, error) {
	if strings.HasPrefix(privateKey, "0x") {
		privateKey = privateKey[2:]
	}
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	chainID, err := svc.Client().NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	return types.SignTx(tx, types.NewLondonSigner(chainID), privateKeyECDSA)
}

func (svc *Service) SignerHash(ctx context.Context, tx *types.Transaction) ([]byte, error) {
	chainId, err := svc.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	signer := types.NewLondonSigner(chainId).Hash(tx)
	return signer.Bytes(), nil
}

func (svc *Service) WithSignature(ctx context.Context, tx *types.Transaction, signature []byte) (*types.Transaction, error) {
	chainId, err := svc.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	return tx.WithSignature(types.NewLondonSigner(chainId), signature)
}

func (svc *Service) Broadcast(ctx context.Context, tx *types.Transaction) error {
	return svc.client.SendTransaction(ctx, tx)
}

func (svc *Service) createTransactionInfo(ctx context.Context, tx *types.Transaction, receipt *types.Receipt, signer types.Signer,
	currentBlockHeight uint64, block *types.Block) (*TransactionInfo, error) {
	if tx.To() == nil {
		return nil, ErrNotSupportTX
	}

	txInfo := TransactionInfo{
		ID:          tx.Hash().String(),
		To:          strings.ToUpper(tx.To().Hex()),
		BlockNumber: uint64(receipt.BlockNumber.Int64()),
	}

	//fee
	gasUsed := decimal.NewFromInt(int64(receipt.GasUsed))
	switch tx.Type() {
	case 0: //legacy
		gasPrice, err := decimal.NewFromString(tx.GasPrice().String())
		if err != nil {
			return nil, err
		}
		txInfo.Fee = gasPrice.Mul(gasUsed).Div(decimal18)
	case 2: //eip-1559
		tip, err := decimal.NewFromString(tx.GasTipCap().String())
		if err != nil {
			return nil, err
		}

		baseFee, err := decimal.NewFromString(block.BaseFee().String())
		if err != nil {
			return nil, err
		}
		txInfo.Fee = baseFee.Add(tip).Mul(gasUsed).Div(decimal18)
	}

	if currentBlockHeight > txInfo.BlockNumber+svc.blockConfirmationNum {
		if receipt.Status == 1 {
			txInfo.State = TransactionSateSuccess
		} else {
			txInfo.State = TransactionSateFail
		}
	} else {
		txInfo.State = TransactionSatePending
	}

	isTokenAddress := true
	tokenInfo, err := svc.ERC20Info(ctx, tx.To().Hex())
	if err != nil {
		if err.Error() == "no contract code at given address" {
			isTokenAddress = false
		} else {
			return nil, ErrNotSupportTX
		}
	}

	if isTokenAddress {
		txInfo.TokenAddress = strings.ToUpper(tx.To().Hex())
		inputData := hex.EncodeToString(tx.Data())
		if !isMethodSupport(tx) {
			return nil, ErrNotSupportTX
		}

		methodID := inputData[0:8]
		decimalPlaces := decimal.New(1, int32(tokenInfo.Decimals))
		//交易失败logs不会有资料 需要自己解析
		if len(receipt.Logs) == 0 {
			decodeSig, err := hex.DecodeString(methodID)
			if err != nil {
				return nil, err
			}
			method, err := svc.eabi.MethodById(decodeSig)
			if err != nil {
				return nil, err
			}

			decodeData, err := hex.DecodeString(inputData[8:])
			if err != nil {
				return nil, err
			}

			result, err := method.Inputs.UnpackValues(decodeData)
			if err != nil {
				return nil, err
			}

			toAddr, err := cast.ToString(result[0])
			if err != nil {
				return nil, err
			}

			strAmount, err := cast.ToString(result[1])
			if err != nil {
				return nil, err
			}
			amount, err := decimal.NewFromString(strAmount)
			if err != nil {
				return nil, err
			}

			amount = amount.Div(decimalPlaces)

			txInfo.To = strings.ToUpper(common.HexToAddress(toAddr).String())
			txInfo.Amount = amount
			return &txInfo, nil
		}
		instance, err := NewToken(*tx.To(), svc.client)
		if err != nil {
			return nil, err
		}
		log := receipt.Logs[0]
		ethTransfer, err := instance.ParseTransfer(*log)
		if err != nil {
			return nil, err
		}

		amount, err := decimal.NewFromString(ethTransfer.Value.String())
		if err != nil {
			return nil, ErrNotSupportTX
		}

		txInfo.TokenAddress = strings.ToUpper(log.Address.Hex())
		txInfo.From = strings.ToUpper(ethTransfer.From.Hex())
		txInfo.To = strings.ToUpper(ethTransfer.To.Hex())
		txInfo.Amount = amount.Div(decimalPlaces)
		return &txInfo, nil
	}
	from, err := types.Sender(signer, tx)
	if err != nil {
		return nil, err
	}

	txInfo.From = strings.ToUpper(from.Hex())
	amount, err := decimal.NewFromString(tx.Value().String())
	if err != nil {
		return nil, err
	}
	txInfo.Amount = amount.Div(decimal18)

	return &txInfo, nil
}

func (svc *Service) Transaction(ctx context.Context, txID string) (*TransactionInfo, error) {
	txHash := common.HexToHash(txID)
	tx, isPending, err := svc.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if isPending {
		return nil, fmt.Errorf("tx was not found,hash:%s", txHash.Hex())
	}

	receipt, err := svc.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}

	chainId, err := svc.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	signer := types.NewLondonSigner(chainId)
	blockHeight, err := svc.CurrentBlockHeight(ctx)
	if err != nil {
		return nil, err
	}

	block, err := svc.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return nil, err
	}

	txInfo, err := svc.createTransactionInfo(ctx, tx, receipt, signer, blockHeight, block)
	if err != nil {
		return nil, err
	}

	return txInfo, nil
}

func (svc *Service) IsContractAddress(ctx context.Context, contractAddress common.Address, currentBlockHeight uint64) (bool, error) {
	blockNumber := big.NewInt(int64(currentBlockHeight))
	b, err := svc.client.CodeAt(ctx, contractAddress, blockNumber)
	if err != nil {
		return false, err
	}

	if len(b) > 3 {
		return true, nil
	}

	return false, nil
}

func (svc *Service) Block(ctx context.Context, number uint64) (*BlockInfo, error) {
	blockNumber := big.NewInt(int64(number))
	block, err := svc.client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	blockInfo := BlockInfo{
		BlockNumber:  number,
		Hash:         block.Hash().String(),
		Time:         time.Unix(int64(block.Time()), 0),
		Transactions: []*TransactionInfo{},
	}

	currentBlockHeight, err := svc.CurrentBlockHeight(ctx)
	if err != nil {
		return nil, err
	}

	chainID, err := svc.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	signer := types.NewLondonSigner(chainID)
	for _, tx := range block.Transactions() {
		receipt, err := svc.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}

		txInfo, err := svc.createTransactionInfo(ctx, tx, receipt, signer, currentBlockHeight, block)
		if err != nil {
			if errors.Is(err, ErrNotSupportTX) {
				continue
			}
			return nil, err
		}
		txInfo.Time = time.Unix(int64(block.Time()), 0)
		blockInfo.Transactions = append(blockInfo.Transactions, txInfo)
	}
	return &blockInfo, nil
}

func isMethodSupport(tx *types.Transaction) bool {
	inputData := hex.EncodeToString(tx.Data())
	if len(inputData) < 8 {
		return false
	}

	methodID := inputData[0:8]
	return methodID == transferMethodId
}