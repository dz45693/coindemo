package btc

import (
	"context"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func Test_ValidateBitcoinAddress(t *testing.T) {
	ctx := context.Background()
	flag := ValidateAddress(ctx, "mmrb4vg9bN79TRwFCZNTccwuNhhiKHVR6r")
	assert.True(t, flag)
	flag = ValidateAddress(ctx, "1234567890")
	assert.False(t, flag)

}

func Test_CurrentBlockHeight(t *testing.T) {
	ctx := context.Background()
	svc := getService()
	blocks, _ := svc.CurrentBlockHeight(ctx)
	log.Println(blocks)
	assert.True(t, blocks > 1)
}

func Test_CreateAddressByPubKey(t *testing.T) {
	ctx := context.Background()
	publicStr := "0x037359ebcc9ec202fa5ee736388ff8ed2aea29b1d3a1b049588e593696f3fc6355"
	svc := getService()
	address, _ := svc.CreateAddressByPubKey(ctx, publicStr)
	assert.Equal(t, address, "mmrb4vg9bN79TRwFCZNTccwuNhhiKHVR6r")

}

func Test_GetRawTransaction(t *testing.T) {
	ctx := context.Background()
	svc := getService()
	transaction, _ := svc.Transaction(ctx, "92499d8de25472a191660e3a7374ded8ed78f8d7360aa8485e92ca90b75b306f")
	assert.Equal(t, transaction.State, TransactionSateSuccess)
}

func Test_GetBlock(t *testing.T) {
	ctx := context.Background()
	svc := getService()
	cblock, _ := svc.Block(ctx, 2103994)
	log.Println(len(cblock.Transactions))
	assert.True(t, len(cblock.Transactions) >= 1)

	amount := "0.001"
	result, _ := decimal.NewFromString(amount)

	amountVaule := result.Mul(decimal.NewFromFloat(100000000)).IntPart()

	log.Println("amountVaule ", amountVaule)

	//var balance float64=10000
	outPut := decimal.NewFromInt(amountVaule)

	outPutValue := outPut.Div(decimal.NewFromFloat(100000000))

	log.Println("outPutValue ", outPutValue)
	//
	//
	for _, tx := range cblock.Transactions {
		//log.Println(tx.State)
		//log.Println(tx.Asm)
		//log.Println(tx.Hex)
		//log.Println(tx.TxID)
		//log.Println(tx.BlockHash)
		//log.Println(tx.Blocktime)
		//log.Println(tx.Confirmations)
		//log.Println(tx.N)
		log.Println(tx.Address)
		log.Println("value ", tx.Value)
	}
}

func Test_EstimateFee(t *testing.T) {
	ctx := context.Background()
	svc := getService()
	fee, _ := svc.EstimateFee(ctx, 210301)

	log.Println(fee)
}

func getService() *Service {
	svc, _ := NewService("45.195.61.126:18332", "testbtc", "c2ckY1CvyU1WR97uWsoC")
	return svc
}
