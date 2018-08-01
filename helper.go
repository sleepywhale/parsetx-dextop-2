package main

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

func U64(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

func Sub(x, y *big.Int) *big.Int {
	return new(big.Int).Sub(x, y)
}

// Returns x**y
func Exp(x, y *big.Int) *big.Int {
	return new(big.Int).Exp(x, y, nil)
}

func And(x, y *big.Int) *big.Int {
	return new(big.Int).And(x, y)
}

// returns `u & 0xFFFFFFFFFFFFFFFF` and `u = u >> 64`
func PopUint64(u *big.Int) uint64 {
	result := And(u, U64(0xFFFFFFFFFFFFFFFF)).Uint64()
	u.Rsh(u, 64)
	return result
}

// returns `u & 0xFFFFFFFF` and `u = u >> 32`
func PopUint32(u *big.Int) uint32 {
	result := And(u, U64(0xFFFFFFFF)).Uint64()
	u.Rsh(u, 32)
	return uint32(result)
}

// returns `u & 0xFFFF` and `u = u >> 16`
func PopUint16(u *big.Int) uint16 {
	result := And(u, U64(0xFFFF)).Uint64()
	u.Rsh(u, 16)
	return uint16(result)
}

// returns `u & 0xFF` and `u = u >> 8`
func PopUint8(u *big.Int) uint8 {
	result := And(u, U64(0xFF)).Uint64()
	u.Rsh(u, 8)
	return uint8(result)
}

func PopUint160(u *big.Int) *big.Int {
	mask := Sub(Exp(U64(2), U64(160)), U64(1))
	result := And(u, mask)
	u.Rsh(u, 160)
	return result
}

// Returns an error if the tx is not found or failed to parse.
// Returns a tx with nil BlockHash and BlockNumber if it has not been confirmed.
func GetTransactionByHash(
	ethNodeUrl string, ctx context.Context, txHash common.Hash) (*Transaction, error) {

	cli, err := rpc.DialContext(ctx, ethNodeUrl)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	var result *Transaction
	err = cli.CallContext(ctx, &result, "eth_getTransactionByHash", txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ethereum.NotFound
	}

	if result.BlockHash != nil {
		if result.BlockHash.Big().BitLen() == 0 {
			result.BlockHash = nil
		}
	}
	if (result.BlockHash == nil) != (result.BlockNumber == nil) {
		return nil, fmt.Errorf("BlockHash and BlockNumber should be both or none nil. %+v", result)
	}
	return result, nil
}

// `header` can be nil, in which case it skips parsing header.
// Returns partial result when there is an error. The argument will NOT be modified.
func ParseOpsFromU256(header *big.Int, originalBody []*big.Int) (string, error) {
	// make a copy to avoid modifying the argument
	header = new(big.Int).Set(header)
	u256s := make([]*big.Int, len(originalBody))
	for i, u := range originalBody {
		u256s[i] = new(big.Int).Set(u)
	}

	out := new(bytes.Buffer)
	if len(u256s) == 0 {
		return "", fmt.Errorf("empty body")
	}

	if header != nil {
		// <newLogicTimeSec>(64) <beginIndex>(64)
		beginIndex := PopUint64(header)
		newLogicTimeSec := PopUint64(header)
		if header.BitLen() != 0 {
			return out.String(), fmt.Errorf("invalid header")
		}
		fmt.Fprintln(out, "newLogicTimeSec:", newLogicTimeSec)
		fmt.Fprintln(out, "beginIndex:", beginIndex)
	}
	fmt.Fprintln(out, "len(body):", len(u256s))

	for i := 0; i < len(u256s); {
		bits := u256s[i]
		opcode := PopUint16(bits)
		if (opcode >> 8) != 0xDE {
			return out.String(), fmt.Errorf("wrong magic number")
		}

		consumed := 1
		var err error
		switch opcode {
		case 0xDE01:
			err = parseConfirmDepositOp(out, bits)
		case 0xDE02:
			err = parseInitiateWithdrawOp(out, bits)
		case 0xDE03:
			consumed, err = parseMatchOrdersOp(out, bits, u256s[i+1:])
		case 0xDE04:
			err = parseHardCancelOrderOp(out, bits)
		case 0xDE05:
			err = parseSetFeeRatesOp(out, bits)
		case 0xDE06:
			err = parseSetFeeRebatePercentOp(out, bits)
		default:
			return out.String(), fmt.Errorf("invalid opcode %#x", opcode)
		}

		if err != nil {
			return out.String(), err
		}
		i += consumed
		if i < len(u256s) {
			fmt.Fprintln(out, "")
		}
	}
	return out.String(), nil
}

func parseConfirmDepositOp(out *bytes.Buffer, bits *big.Int) error {
	fmt.Fprintln(out, "operation ConfirmDeposit:")
	fmt.Fprintln(out, "  depositIndex: ", bits)
	return nil
}

func parseInitiateWithdrawOp(out *bytes.Buffer, bits *big.Int) error {
	fmt.Fprintln(out, "operation InitiateWithdraw:")
	// <amountE8>(64) <tokenCode>(16) <traderAddr>(160) <opcode>(16)
	fmt.Fprintf(out, "  traderAddr: %#x\n", PopUint160(bits))
	fmt.Fprintln(out, "  tokenCode:", PopUint16(bits))
	fmt.Fprintln(out, "  amountE8:", PopUint64(bits))
	return nil
}

func parseMatchOrdersOp(out *bytes.Buffer, bits *big.Int, rest []*big.Int) (consumed int, err error) {
	fmt.Fprintln(out, "operation MatchOrder:")
	v1 := PopUint8(bits)
	if v1 == 0 {
		fmt.Fprintln(out, "  makerOrder (existing):")
		if len(rest) < 1 {
			return 0, fmt.Errorf("not enough inputs for matching order")
		}
		consumed += 1
		if err := parseOrderOpOperand(out, bits, nil); err != nil {
			return consumed, err
		}
	} else {
		fmt.Fprintf(out, "  makerOrder (new, v1=%v):\n", v1)
		if v1 != 27 && v1 != 28 {
			return 0, fmt.Errorf("invalid v1: %v", v1)
		}
		if len(rest) < 4 {
			return 0, fmt.Errorf("not enough inputs for matching order")
		}
		consumed += 4
		if err := parseOrderOpOperand(out, bits, rest); err != nil {
			return consumed, err
		}
		rest = rest[3:]
	}

	bits, rest = rest[0], rest[1:]
	v2 := PopUint8(bits)
	if v2 == 0 {
		fmt.Fprintln(out, "  takerOrder (existing):")
		consumed += 1
		if err := parseOrderOpOperand(out, bits, nil); err != nil {
			return consumed, err
		}
	} else {
		fmt.Fprintf(out, "  takerOrder (new, v2=%v):\n", v2)
		if v2 != 27 && v2 != 28 {
			return 0, fmt.Errorf("invalid v2: %v", v2)
		}
		if len(rest) < 3 {
			return 0, fmt.Errorf("not enough inputs for matching order")
		}
		consumed += 4
		if err := parseOrderOpOperand(out, bits, rest); err != nil {
			return consumed, err
		}
	}

	return consumed, nil
}

// Precondition: `rest` is nil or has at least 3 elements.
func parseOrderOpOperand(out *bytes.Buffer, bits *big.Int, rest []*big.Int) error {
	fmt.Fprintf(out, "    trader: %#x\n", PopUint160(bits))
	fmt.Fprintln(out, "    nonce:", PopUint64(bits))
	if bits.BitLen() != 0 {
		return fmt.Errorf("extra bits in orderKey: %s", bits)
	}
	if rest == nil { // existing order
		return nil
	}

	// <expireTimeSec>(64) <amountE8>(64) <priceE8>(64) <ioc>(8) <action>(8) <pairId>(32)
	bits = rest[0]
	fmt.Fprintln(out, "    pairId  :", PopUint32(bits))
	fmt.Fprintln(out, "    action  :", PopUint8(bits))
	fmt.Fprintln(out, "    ioc     :", PopUint8(bits))
	fmt.Fprintln(out, "    priceE8 :", PopUint64(bits))
	fmt.Fprintln(out, "    amountE8:", PopUint64(bits))
	fmt.Fprintln(out, "    expire  :", PopUint64(bits))
	if bits.BitLen() != 0 {
		return fmt.Errorf("extra data in order bits: %#x", bits)
	}

	if rest[1].BitLen() == 0 {
		return fmt.Errorf("signature uint256 r is zero")
	}
	fmt.Fprintf(out, "    s       : %#x\n", rest[1])
	if rest[2].BitLen() == 0 {
		return fmt.Errorf("signature uint256 s is zero")
	}
	fmt.Fprintf(out, "    t       : %#x\n", rest[2])
	return nil
}

func parseHardCancelOrderOp(out *bytes.Buffer, bits *big.Int) error {
	fmt.Fprintln(out, "operation HardCancel:")
	fmt.Fprintln(out, "  <to be implemented>")
	return nil
}

func parseSetFeeRatesOp(out *bytes.Buffer, bits *big.Int) error {
	fmt.Fprintln(out, "operation SetFeeRates:")
	fmt.Fprintln(out, "  <to be implemented>")
	return nil
}

func parseSetFeeRebatePercentOp(out *bytes.Buffer, bits *big.Int) error {
	fmt.Fprintln(out, "operation SetFeeRebatePercent:")
	fmt.Fprintln(out, "  <to be implemented>")
	return nil
}

func normalizeAmount(orignalAmount *big.Int, decimal uint) string {
	n := new(big.Float).SetInt(orignalAmount)
	n = n.Quo(n, new(big.Float).SetInt(big.NewInt(int64(decimal))))
	return n.Text('f', 8)
}

func formatToken(tokenCode uint16, originalAmount *big.Int) string {
	_, ok := tokenInfo[tokenCode]
	if !ok {
		fmt.Println("tokenCode %v does not exist")
	}
	name := tokenInfo[tokenCode].Name
	amount := normalizeAmount(originalAmount, tokenInfo[tokenCode].Decimal)
	return fmt.Sprintf("Token(%s): %s", name, amount)
}

func CtxWithTimeoutMs(timeoutMs int64) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	return ctx
}

func getTransactionInputDataByHash(tx string) (string, error) {
	transaction, err := GetTransactionByHash(ethNodeUrl, CtxWithTimeoutMs(8e3), common.HexToHash(tx))
	if err != nil {
		return "", fmt.Errorf("failure in GetTransactionByHash")
	}
	return transaction.Input.String(), nil
}

func DecodeInputData(tx string) (string, error) {
	hexStr, err := getTransactionInputDataByHash(tx)
	if err != nil {
		return "", err
	}
	data, err := hexutil.Decode(hexStr)
	if err != nil {
		return "", err
	}
	if len(data) < 4 {
		return "", fmt.Errorf("invalid data length")
	}
	hashes := data[0:4]
	method, err := Dex2Abi.MethodById(hashes)
	if err != nil {
		return "", err
	}
	fmt.Println(method.Name)

	switch method.Name {
	case "depositEth":
		var out DepositEthInput
		err = method.Inputs.Unpack(&out, data[4:])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Deposit ETH by %s", out.TraderAddr.Hex()), nil

	case "depositToken":
		var out DepositTokenInput
		err = method.Inputs.Unpack(&out, data[4:])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Depsoit %s by %s",
			formatToken(out.TokenCode, out.OriginalAmount), out.TraderAddr.Hex()), nil

	case "withdrawEth":
		var out WithdrawEthInput
		err = method.Inputs.Unpack(&out, data[4:])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Withdraw ETH for %s", out.TraderAddr.Hex()), nil

	case "withdrawToken":
		var out WithdrawTokenInput
		err = method.Inputs.Unpack(&out, data[4:])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Withdraw Token (%s) for %s",
			tokenInfo[out.TokenCode].Name, out.TraderAddr.Hex()), nil

	case "exeSequence":
		var out ExeSequenceInput
		err = method.Inputs.Unpack(&out, data[4:])
		return ParseOpsFromU256(out.Header, out.Body)

	default:
		return method.Name, nil
	}
}
