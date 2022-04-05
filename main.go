package main

import (
	"context"
	"crypto/ecdsa"
	"demo/token"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
)

/*
 ==================
(0) 0xE280029a7867BA5C9154434886c241775ea87e53 (100 ETH)
(1) 0x68dB32D26d9529B2a142927c6f1af248fc6Ba7e9 (100 ETH)
(2) 0x35bb6eF95c72bf4804334BB9d6A3c77Bef18d81B (100 ETH)
(3) 0x26d8094A90AD1A4440600fa18073730Ef17F5eCE (100 ETH)
(4) 0xb53cC19aD713e00cB71542d064215784c908D387 (100 ETH)
(5) 0xcb98ce2619f90f54052524Fb79b03E0261b01BEE (100 ETH)
(6) 0x4C4Cb54BdBad6c96805b92b8063F3c75B24F65eB (100 ETH)
(7) 0x67B27Df78bb0f199EDBd568e655BD9b2B202866d (100 ETH)
(8) 0xE93E2b43fC45CCEcc056A6Ea400972298A304b4B (100 ETH)
(9) 0x288Ab710f8DEc0b13753Fec71161E50Ee0cDA7e6 (100 ETH)

Private Keys
==================
(0) 0xf1b3f8e0d52caec13491368449ab8d90f3d222a3e485aa7f02591bbceb5efba5
(1) 0x91821f9af458d612362136648fc8552a47d8289c0f25a8a1bf0860510332cef9
(2) 0xbb32062807c162a5243dc9bcf21d8114cb636c376596e1cf2895ec9e5e3e0a68
(3) 0x95ce6122165d94aa51b0fcf51021895b39b0ff291aa640c803d5401bd87894d5
(4) 0x3af93668029f95d526fc1d2bdefccc120bfe1d26a0462d268e8f6b2f71402ba3
(5) 0x3b24a4fdf2e6e1375008c387c5456ce00cb0772435ae1938c2fe833103393b9a
(6) 0xcba858feeb49e1ca8053a5213987a22c3ee83d9f9f396e138940a018dd837ebb
(7) 0xdf48bfda4cb4b4e094803e923836a8538fbf607da79f6e46d68cdd43fb2f3f88
(8) 0x487efb6249a8a4d45a19c8e5d1e5c7d3f6610a7e69f8f81ddcf368f9a0c0d6d5
(9) 0xbb4cce73db59f456ea427e5862fdb0d5bc038a7d0b930cbb45e1c4f6d122289e

ganache-cli -m "much repair shock carbon improve miss forget sock include bullet interest solution"

Gas Price
==================
20000000000

Gas Limit
==================
6721975

Call Gas Limit
==================
9007199254740991
*/
func main() {
	//client, err := ethclient.Dial("https://cloudflare-eth.com")
	//client, err := ethclient.Dial("http://localhost:8545")
	//if err != nil {
	//log.Fatal(err)
	//	}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	auth := bind.NewKeyedTransactor(privateKey)

	balance := new(big.Int)
	balance.SetString("10000000000000000000", 10) // 10 eth in wei

	address := auth.From
	genesisAlloc := map[common.Address]core.GenesisAccount{
		address: {
			Balance: balance,
		},
	}

	blockGasLimit := uint64(4712388)
	client := backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)

	fromAddress := auth.From
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

	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	var data []byte
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	chainID := big.NewInt(1337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex()) // tx sent: 0xec3ceb05642c61d33fa6c951b54080d1953ac8227be81e7b5e4e2cfed69eeb51

	client.Commit()

	receipt, err := client.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		log.Fatal(err)
	}
	if receipt == nil {
		log.Fatal("receipt is nil. Forgot to commit?")
	}

	fmt.Printf("status: %v\n", receipt.Status) // status: 1
	ret, _ := client.BalanceAt(context.Background(), toAddress, nil)
	fmt.Println(ret.String())
}

func deployContract(client *ethclient.Client) string {
	/*
		0x4A31ECe693fB614935eFB337034F9C79efEC03B5
		0x6578c861a339fa8c3d30fabb6ef98f2f266b7cab697c54360dd2dc792ae632be
	*/

	/*
		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	*/

	auth := getAuth(client)
	input := "1.0"
	//address, tx, instance, err := store.DeployStore(auth, client, input)
	address, tx, instance, err := token.DeployToken(auth, client, "gavin", input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(address.Hex())   // 0x147B8eb97fD247D06C4006D269c90C1908Fb5D54
	fmt.Println(tx.Hash().Hex()) // 0xdae8ba5444eefdc99f4d45cd0c4f24056cba6a02cefbf78066ef9f4188ff7dc0

	_ = instance
	return address.Hex()
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

func writeAndRead(client *ethclient.Client, tokenAddress, toAddress common.Address) {
	instance, err := token.NewToken(tokenAddress, client)
	if err != nil {
		log.Fatal(err)
	}

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
	fmt.Printf("wei: %s\n", bal) // "wei: 74605500647408739782407023"
	/*
		auth.Nonce= big.NewInt(int64(tx.Nonce()))
		result, err := instance.Approve(auth,toAddress,big.NewInt(1))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("tx sent2: %s", result.Hash().Hex()) // tx sent: 0x8d490e535678e9a24360e955d75b27ad307bdfb97a1dca51d0f3035dcee3e870
		bal, err = instance.BalanceOf(&bind.CallOpts{}, toAddress)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wei: %s\n", bal) // "wei: 74605500647408739782407023"
	*/
}

type LogTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}

// LogApproval ..
type LogApproval struct {
	TokenOwner common.Address
	Spender    common.Address

	Tokens *big.Int
}
