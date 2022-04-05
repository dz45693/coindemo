package eth

import (
	"context"
	"crypto/ecdsa"
	"demo/token"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"log"
	"math/big"
	"testing"
	"time"
)

const (
	owner1Addr       = "0xE280029a7867BA5C9154434886c241775ea87e53"
	owner1PrivateKey = "0xf1b3f8e0d52caec13491368449ab8d90f3d222a3e485aa7f02591bbceb5efba5"
	owner2Addr       = "0x68dB32D26d9529B2a142927c6f1af248fc6Ba7e9"
	tokenAddr        = "0xf3585FCD969502624c6A8ACf73721d1fce214E83"
)

func Test_ERc20Info(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	tokenInfo, err := svc.ERC20Info(ctx, tokenAddr)
	if err != nil {
		t.Log(err)
	}
	t.Log(tokenInfo)
	_, err = svc.ERC20Info(ctx, tokenAddr+"a")
	if err != nil {
		//no contract code at given address
		t.Log(err)
	}
}

func Test_SuggestionPrice(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	gasPrice, err := svc.SuggestGasPrice(ctx)
	if err != nil {
		t.Log(err)
	}
	t.Log(gasPrice.String())

}

func Test_CreateAddress(t *testing.T) {
	svc := getService()
	ctx := context.Background()

	mnemonic := "tag volcano eight thank tide danger coast health above argue embrace heavy"
	addr, err := svc.CreateAddress(ctx, mnemonic, 0)
	if err != nil {
		t.Log(err)
	}
	t.Log(addr)

	///
	privateKey, err := crypto.HexToECDSA("f1b3f8e0d52caec13491368449ab8d90f3d222a3e485aa7f02591bbceb5efba5")
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	publicKeyBytes := crypto.CompressPubkey(publicKeyECDSA)
	publicKeyStr := hexutil.Encode(publicKeyBytes)[:]
	addr, err = svc.CreateAddressByPubKey(ctx, publicKeyStr)
	if err != nil {
		t.Log(err)
	}
	t.Log(addr)
}

func Test_Balance(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	balance, err := svc.BalanceETH(ctx, owner1Addr)
	if err != nil {
		t.Log(err)
	}
	t.Log(balance.String())

	balance, err = svc.BalanceERC20(ctx, tokenAddr, owner1Addr)
	if err != nil {
		t.Log(err)
	}
	t.Log(balance.String())
}

func Test_GetTransaction(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	txAddress := ethTransaction()
	txInfo, err := svc.Transaction(ctx, txAddress)
	if err != nil {
		t.Log(err)
	}
	t.Log(txInfo)

	txAddress = ercTransaction()
	txInfo, err = svc.Transaction(ctx, txAddress)
	if err != nil {
		t.Log(err)
	}
	t.Log(txInfo)
}

func Test_CreateTransactEth(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	var gasTip int32 = 2
	gasMaxFee, _ := svc.MaxFee(ctx, gasTip)

	req := CreateTransactionRequest{
		From:      owner1Addr,
		To:        owner2Addr,
		Amount:    "0.0001",
		GasLimit:  uint64(21000),
		GasMaxFee: gasMaxFee.String(),
		GasTip:    gasTip,
	}
	req.Nonce, _ = svc.Nonce(ctx, req.From)
	tx, err := svc.CreateTransaction(ctx, req)
	if err != nil {
		t.Log(err)
	}
	t.Log(tx.Value().String())

	//
	tx, err = svc.SignTransaction(ctx, tx, owner1PrivateKey)
	if err != nil {
		t.Log(err)
	}

	err = svc.Broadcast(ctx, tx)
	if err != nil {
		t.Log(err)
	}
	///
	hash := tx.Hash().String()
	time.Sleep(3 * time.Second)
	txtInfo, err := svc.Transaction(ctx, hash)
	if err != nil {
		t.Log(err)
	}
	t.Log(txtInfo)
}

func Test_CreateTransactERC(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	var gasTip int32 = 2
	gasMaxFee, _ := svc.MaxFee(ctx, gasTip)
	req := CreateTransactionRequest{
		TokenAddress:       tokenAddr,
		From:               owner1Addr,
		To:                 owner2Addr,
		Amount:             "2",
		GasMaxFee:          gasMaxFee.String(),
		GasTip:             gasTip,
		DisableEstimateGas: false,
	}
	req.Nonce, _ = svc.Nonce(ctx, req.From)
	tx, err := svc.CreateTransaction(ctx, req)
	if err != nil {
		t.Log(err)
	}
	t.Log(tx)
	//
	tx, err = svc.SignTransaction(ctx, tx, owner1PrivateKey)
	if err != nil {
		t.Log(err)
	}

	err = svc.Broadcast(ctx, tx)
	if err != nil {
		t.Log(err)
	}
}

func Test_Block(t *testing.T) {
	svc := getService()
	ctx := context.Background()
	height, err := svc.CurrentBlockHeight(ctx)
	if err != nil {
		t.Log(err)
	}
	block, err := svc.Block(ctx, height)
	if err != nil {
		t.Log(err)
	}
	t.Log(block)
}

func TestEip1559(t *testing.T) {
	//invalid remainder
	pkArr, _ := hex.DecodeString("f1b3f8e0d52caec13491368449ab8d90f3d222a3e485aa7f02591bbceb5efba5")
	prvKey, _ := crypto.ToECDSA(pkArr)
	addrTo, _ := hex.DecodeString("68dB32D26d9529B2a142927c6f1af248fc6Ba7e9")
	to := common.BytesToAddress(addrTo)
	nonce, gasMax, gasTip, gasLimit, value := uint64(0), big.NewInt(38694000460), big.NewInt(3869400046), uint64(22012), big.NewInt(50000000000000000)
	var data []byte
	tx := types.NewTx(&types.DynamicFeeTx{Nonce: nonce, GasFeeCap: gasMax, GasTipCap: gasTip, Gas: gasLimit, To: &to, Value: value, Data: data})
	config := params.RopstenChainConfig
	s := types.MakeSigner(config, config.LondonBlock)
	tx, err := types.SignTx(tx, s, prvKey)
	if err != nil {
		t.Log(err)
	}
	client := getClient()
	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		t.Log(err)
	}
}

/*
==================
(0) 0xE280029a7867BA5C9154434886c241775ea87e53 (100 ETH)
(1) 0x68dB32D26d9529B2a142927c6f1af248fc6Ba7e9 (100 ETH)
(2) 0x35bb6eF95c72bf4804334BB9d6A3c77Bef18d81B (100 ETH)


Private Keys
==================
(0) 0xf1b3f8e0d52caec13491368449ab8d90f3d222a3e485aa7f02591bbceb5efba5
(1) 0x91821f9af458d612362136648fc8552a47d8289c0f25a8a1bf0860510332cef9
(2) 0xbb32062807c162a5243dc9bcf21d8114cb636c376596e1cf2895ec9e5e3e0a68

ganache-cli -m "much repair shock carbon improve miss forget sock include bullet interest solution"
*/

func Test_Deploy(t *testing.T) {
	s := getService()

	auth := getAuth(s.client)
	input := "1.0"
	//address, tx, instance, err := store.DeployStore(auth, client, input)
	address, tx, _, err := token.DeployToken(auth, s.client, "gavin", input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(address.Hex())   // 0x147B8eb97fD247D06C4006D269c90C1908Fb5D54
	fmt.Println(tx.Hash().Hex()) // 0xdae8ba5444eefdc99f4d45cd0c4f24056cba6a02cefbf78066ef9f4188ff7dc0

}

func getAuth(client *ethclient.Client) *bind.TransactOpts {
	privateKey, err := crypto.HexToECDSA("f1b3f8e0d52caec13491368449ab8d90f3d222a3e485aa7f02591bbceb5efba5")
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	auth := bind.NewKeyedTransactor(privateKey)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(6721975) // in units
	auth.GasPrice = big.NewInt(20000000000)

	return auth
}

func getService() *Service {
	return NewService(getClient(), 0, 12)
}

func getClient() *ethclient.Client {
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}
	return client
}
func ethTransaction() string {
	client := getClient()

	privateKey, err := crypto.HexToECDSA(owner1PrivateKey[2:])
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(1000000000000000000) // in wei (1 eth)
	gasLimit := uint64(21000)                // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	toAddress := common.HexToAddress(owner2Addr)
	var data []byte
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s", signedTx.Hash().Hex())
	return signedTx.Hash().Hex()
}

func ercTransaction() string {
	client := getClient()
	instance, err := token.NewToken(common.HexToAddress(tokenAddr), client)
	if err != nil {
		log.Fatal(err)
	}
	toAddress := common.HexToAddress(owner2Addr)
	auth := getAuth(client)
	tx, err := instance.Transfer(auth, toAddress, big.NewInt(1))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s", tx.Hash().Hex()) // tx sent: 0x8d490e535678e9a24360e955d75b27ad307bdfb97a1dca51d0f3035dcee3e870
	bal, err := instance.BalanceOf(&bind.CallOpts{}, toAddress)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wei: %s\n", bal)
	return tx.Hash().String()
}
