package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
	"math/big"
)

type Service struct {
	rpc *rpcclient.Client
}

func NewService(host, user, password string) (*Service, error) {
	conn := &rpcclient.ConnConfig{
		Host:         host,
		User:         user,
		Pass:         password,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(conn, nil)
	if err != nil {
		return nil, err
	}

	return &Service{rpc: client}, nil
}

//检查比特币地址是否有效
func ValidateAddress(ctx context.Context, address string) bool {
	fullHash := base58.Decode(address)
	length := len(fullHash)
	if length != 25 {
		return false
	}

	prefixHash := fullHash[:length-CheckSumLength]
	tailHash := fullHash[length-CheckSumLength:]
	tailHash2 := CheckSumHash(prefixHash)
	if bytes.Compare(tailHash, tailHash2) == 0 {
		return true
	}
	return false
}

func (t *Service) Block(ctx context.Context, index int64) (block *Block, err error) {
	blockHash, err := t.rpc.GetBlockHash(index)
	if err != nil {
		return nil, err
	}

	msgBlock, err := t.rpc.GetBlock(blockHash)
	if err != nil {
		return nil, err
	}

	block = &Block{Block: msgBlock}

	for _, tx := range msgBlock.Transactions {
		rawTransaction, _ := t.Transaction(ctx, tx.TxHash().String())
		for _, outTx := range rawTransaction.Result.Vout {
			transaction := TransactionInfo{
				N:             outTx.N,
				Address:       outTx.ScriptPubKey.Addresses,
				Asm:           outTx.ScriptPubKey.Asm,
				BlockHash:     rawTransaction.Result.BlockHash,
				BlockTime:     rawTransaction.Result.Blocktime,
				Confirmations: rawTransaction.Result.Confirmations,
				Hex:           outTx.ScriptPubKey.Hex,
				Value:         decimal.NewFromFloat(outTx.Value),
				ReqSigs:       outTx.ScriptPubKey.ReqSigs,
				Time:          rawTransaction.Result.Time,
				State:         rawTransaction.State,
				TxId:          rawTransaction.Result.Txid,
			}
			block.Transactions = append(block.Transactions, &transaction)
		}
	}
	return block, nil
}

func (t *Service) CurrentBlockHeight(ctx context.Context) (int64, error) {
	currentBlockCount, err := t.rpc.GetBlockCount()
	return currentBlockCount, err
}

func (t *Service) Transaction(ctx context.Context, txId string) (transaction *RawTransactionInfo, err error) {
	txHash, err := chainhash.NewHashFromStr(txId)
	if err != nil {
		return nil, err
	}

	rawResult, err := t.rpc.GetRawTransactionVerbose(txHash)
	if err != nil {
		return nil, err
	}

	transaction = &RawTransactionInfo{State: TransactionSateDefault, Result: rawResult}
	if rawResult.Confirmations > 5 {
		transaction.State = TransactionSateSuccess
	} else if rawResult.Confirmations < 6 && rawResult.Confirmations > 0 {
		transaction.State = TransactionSatePending
	}
	return transaction, nil
}

func (t *Service) CreateAddressByPubKey(ctx context.Context, publicKey string) (string, error) {
	decodePublicKey, err := hexutil.Decode(publicKey)
	if err != nil {
		return "", err
	}

	publicKey1, err := crypto.DecompressPubkey(decodePublicKey)
	if err != nil {
		return "", err
	}

	mainAddress, err := btcutil.NewAddressPubKey(crypto.CompressPubkey(publicKey1), &chaincfg.TestNet3Params)
	if err != nil {
		return "", err
	}
	return mainAddress.EncodeAddress(), nil
}

func (t *Service) CreateTx(ctx context.Context, from, to string, amount, fee decimal.Decimal, utxOS TransactionOutPut) (*wire.MsgTx, []byte, error) {
	baseCoin := decimal.NewFromFloat(100000000)
	amountValue := amount.Mul(baseCoin).IntPart()
	outputValue := utxOS.Value.Mul(baseCoin).IntPart()
	feeValue := fee.Mul(baseCoin).IntPart()
	if outputValue+feeValue < amountValue {
		return nil, nil, fmt.Errorf("the blance of the account is not sufficient,ouptValue[%v] + feeValue[%v] >amountValue[%v]", outputValue, feeValue, amountValue)
	}

	restValue := outputValue - amountValue - feeValue

	destinationAddress, err := btcutil.DecodeAddress(to, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, nil, err
	}

	destinationAddressByte, err := txscript.PayToAddrScript(destinationAddress)
	if err != nil {
		return nil, nil, err
	}

	fromAddress, err := btcutil.DecodeAddress(from, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, nil, err
	}

	fromAddressByte, err := txscript.PayToAddrScript(fromAddress)
	if err != nil {
		return nil, nil, err
	}

	redeemTx := NewTx()
	utxHash, err := chainhash.NewHashFromStr(utxOS.TXId)
	if err != nil {
		return nil, nil, err
	}

	outPoint := wire.NewOutPoint(utxHash, utxOS.N)
	txtIn := wire.NewTxIn(outPoint, nil, nil)
	redeemTx.AddTxIn(txtIn)
	redeemTx.AddTxOut(wire.NewTxOut(amountValue, destinationAddressByte))
	redeemTx.AddTxOut(wire.NewTxOut(restValue, fromAddressByte))
	redeemTx, calcHash, err := SigTx(utxOS.Hex, redeemTx)

	return redeemTx, calcHash, err
}

func (t *Service) SigTx(ctx context.Context, publicKey string, r, s *big.Int, redeemTx *wire.MsgTx) (*wire.MsgTx, error) {
	decodePublicKey, err := hexutil.Decode(publicKey)
	if err != nil {
		return redeemTx, err
	}

	publicKey1, err := crypto.DecompressPubkey(decodePublicKey)
	if err != nil {
		return redeemTx, err
	}

	sig := btcec.Signature{R: r, S: s}
	sigHash := append(sig.Serialize(), byte(txscript.SigHashAll))
	signature, err := SignatureScript(sigHash, (*btcec.PublicKey)(publicKey1), false)
	if err != nil {
		return redeemTx, err
	}

	redeemTx.TxIn[0].SignatureScript = signature
	return redeemTx, nil
}

func (t *Service) BroadcastTx(ctx context.Context, tx *wire.MsgTx) (string, error) {
	txHash, err := t.rpc.SendRawTransaction(tx, false)
	if err != nil {
		return "", err
	}

	return txHash.String(), nil
}

func (t *Service) EstimateFee(ctx context.Context, numBlocks int64) (string, error) {
	estimateFee, err := t.rpc.EstimateFee(numBlocks)
	return fmt.Sprintf("%f", estimateFee), err
}

func NewTx() *wire.MsgTx {
	return wire.NewMsgTx(wire.TxVersion)
}

func SigTx(pkScript string, redeemTx *wire.MsgTx) (*wire.MsgTx, []byte, error) {
	sourcePKScript, err := hex.DecodeString(pkScript)
	if err != nil {
		return nil, nil, err
	}

	hash, err := txscript.CalcSignatureHash(sourcePKScript, txscript.SigHashAll, redeemTx, 0)
	return redeemTx, hash, err
}

func SignatureScript(sig []byte, pk *btcec.PublicKey, compress bool) ([]byte, error) {
	var pkData []byte
	if compress {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}
	return txscript.NewScriptBuilder().AddData(sig).AddData(pkData).Script()
}
