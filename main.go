package main

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	ethNodeUrl = "https://mainnet.infura.io/ZOfDzLOSnNxb7l9ayaHt"
	Dex2Abi    abi.ABI
	tokenInfo  = map[uint16]*TokenInfo{
		100: &TokenInfo{"LOOM", 1e18},
		101: &TokenInfo{"KNC", 1e18},
		102: &TokenInfo{"ZIL", 1e12},
		103: &TokenInfo{"CTXC", 1e18},
		104: &TokenInfo{"YEE", 1e18},
		105: &TokenInfo{"QKC", 1e18},
		106: &TokenInfo{"MEDX", 1e8},
		107: &TokenInfo{"PAL", 1e18},
		108: &TokenInfo{"HPB", 1e18},
		109: &TokenInfo{"XUC", 1e18},
		110: &TokenInfo{"BUT", 1e18},
		116: &TokenInfo{"TTC", 1e18},
		117: &TokenInfo{"AIT", 1e18},
		118: &TokenInfo{"HSC", 1e18},
		119: &TokenInfo{"SNTR", 1e4},
		120: &TokenInfo{"MTC", 1e18},
		121: &TokenInfo{"VITE", 1e18},
		122: &TokenInfo{"XYO", 1e18},
		123: &TokenInfo{"TAU", 1e18},
		124: &TokenInfo{"SNT", 1e18},
		125: &TokenInfo{"TFD", 1e18},
		126: &TokenInfo{"LND", 1e18},
		127: &TokenInfo{"MVC", 1e18},
		128: &TokenInfo{"TOMO", 1e18},
		129: &TokenInfo{"TRAC", 1e18},
		130: &TokenInfo{"PAI", 1e18},
		131: &TokenInfo{"EDR", 1e18},
		132: &TokenInfo{"MAN", 1e18},
		133: &TokenInfo{"HYDRO", 1e18},
		134: &TokenInfo{"DAG", 1e8},
	}
)

type TokenInfo struct {
	Name    string
	Decimal uint
}

// hash: DATA, 32 Bytes - hash of the transaction.
// nonce: QUANTITY - the number of transactions made by the sender prior to this one.
// *blockHash: DATA, 32 Bytes - hash of the block where this tx was in or null if it is pending.
// *blockNumber: QUANTITY - block number where this tx was in or null if it is pending.
type Transaction struct {
	Hash        common.Hash    `json:"hash"`
	Nonce       *hexutil.Big   `json:"nonce"`
	BlockHash   *common.Hash   `json:"blockHash"`
	BlockNumber *hexutil.Big   `json:"blockNumber"`
	Input       *hexutil.Bytes `json:"input"`

	// Add more fields on demand.
	// Spec: https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash
}

type DepositEthInput struct {
	TraderAddr common.Address
}

type DepositTokenInput struct {
	TraderAddr     common.Address
	TokenCode      uint16
	OriginalAmount *big.Int
}

type WithdrawEthInput struct {
	TraderAddr common.Address
}

type WithdrawTokenInput struct {
	TraderAddr common.Address
	TokenCode  uint16
}

type ExeSequenceInput struct {
	Header *big.Int
	Body   []*big.Int
}

func init() {
	var err error
	Dex2Abi, err = abi.JSON(strings.NewReader(Dex2ABI))
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println(os.Args)

	if len(os.Args) != 2 {
		fmt.Println("Usage: parsetx <tx input data in hex>")
		os.Exit(1)
	}
	result, err := DecodeInputData(os.Args[1])
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(result)
	}
}

// var testData1 = "0x96f7d81da5d0ecfd9e26e04937a6fa15f31f39f84acb6f9c50a72fad63689857"
// var testData2 = "0xb7195e7b989de56d6e9c32057c69420d71b084da93d611dd9b30db8274b7a5ff"
