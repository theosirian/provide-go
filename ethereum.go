package provide

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/scrypt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

// The purpose of this class is to expose generic transactional and ABI-related helper
// methods; ethereum.go is a convenience wrapper around JSON-RPC.

// It also caches JSON-RPC client instances in a few flavors (*ethclient.Client and *ethrpc.Client)
// and maps them to an arbitrary `networkID` after successfully dialing the given RPC URL.

var chainConfigs = map[string]*params.ChainConfig{}        // mapping of network ids to *params.ChainConfig
var ethclientRpcClients = map[string][]*ethclient.Client{} // mapping of network ids to *ethclient.Client instances
var ethrpcClients = map[string][]*ethrpc.Client{}          // mapping of network ids to *ethrpc.Client instances

var mutex = &sync.Mutex{}

func clearCachedClients(networkID string) {
	mutex.Lock()
	delete(chainConfigs, networkID)
	for i := range ethrpcClients[networkID] {
		ethrpcClients[networkID][i].Close()
	}
	for i := range ethclientRpcClients[networkID] {
		ethclientRpcClients[networkID][i].Close()
	}
	ethrpcClients[networkID] = make([]*ethrpc.Client, 0)
	ethclientRpcClients[networkID] = make([]*ethclient.Client, 0)
	mutex.Unlock()
}

// DialJsonRpc - dials and caches a new JSON-RPC client instance at the JSON-RPC url and caches it using the given network id
func DialJsonRpc(networkID, rpcURL string) (*ethclient.Client, error) {
	var client *ethclient.Client

	if networkClients, _ := ethclientRpcClients[networkID]; len(networkClients) == 0 {
		rpcClient, err := ResolveJsonRpcClient(networkID, rpcURL)
		if err != nil {
			Log.Warningf("Failed to dial JSON-RPC host: %s", rpcURL)
			return nil, err
		}
		client = ethclient.NewClient(rpcClient)
		mutex.Lock()
		ethrpcClients[networkID] = append(ethrpcClients[networkID], rpcClient)
		ethclientRpcClients[networkID] = append(networkClients, client)
		mutex.Unlock()
		Log.Debugf("Dialed JSON-RPC host @ %s", rpcURL)
	} else {
		client = ethclientRpcClients[networkID][0]
	}

	_, err := GetSyncProgress(client)
	if err != nil {
		Log.Warningf("Failed to read sync progress for *ethclient.Client instance: %s; %s", client, err.Error())
		clearCachedClients(networkID)
		return nil, err
	}

	return client, nil
}

// InvokeJsonRpcClient - invokes the JSON-RPC client for the given network and url
func InvokeJsonRpcClient(networkID, rpcURL, method string, params []interface{}, response interface{}) error {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: time.Second * 60,
	}
	payload := map[string]interface{}{
		"method":  method,
		"params":  params,
		"id":      GetChainID(networkID, rpcURL),
		"jsonrpc": "2.0",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		Log.Warningf("Failed to marshal JSON payload for %s JSON-RPC invocation; %s", method, err.Error())
		return err
	}
	resp, err := client.Post(rpcURL, "application/json", bytes.NewReader(body))
	if err != nil {
		Log.Warningf("Failed to invoke JSON-RPC method: %s; %s", method, err.Error())
		return err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	err = json.Unmarshal(buf.Bytes(), response)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal %s JSON-RPC response: %s; %s", method, buf.Bytes(), err.Error())
	}
	Log.Debugf("Invocation of JSON-RPC method %s succeeded (%v-byte response)", method, buf.Len())
	return nil
}

// ResolveEthClient resolves a cached *ethclient.Client client or dials and caches a new instance
func ResolveEthClient(networkID, rpcURL string) (*ethclient.Client, error) {
	var client *ethclient.Client
	if networkClients, _ := ethclientRpcClients[networkID]; len(networkClients) == 0 {
		client, err := DialJsonRpc(networkID, rpcURL)
		if err != nil {
			Log.Warningf("Failed to dial RPC client for JSON-RPC host: %s", rpcURL)
			return nil, err
		}
		mutex.Lock()
		ethclientRpcClients[networkID] = append(networkClients, client)
		mutex.Unlock()
		Log.Debugf("Dialed JSON-RPC host @ %s", rpcURL)
	} else {
		client = ethclientRpcClients[networkID][0]
		Log.Debugf("Resolved cached *ethclient.Client instance for JSON-RPC host @ %s", rpcURL)
	}
	return client, nil
}

// ResolveJsonRpcClient resolves a cached *ethclient.Client client or dials and caches a new instance
func ResolveJsonRpcClient(networkID, rpcURL string) (*ethrpc.Client, error) {
	var client *ethrpc.Client
	if networkClients, _ := ethrpcClients[networkID]; len(networkClients) == 0 {
		erpc, err := ethrpc.Dial(rpcURL)
		if err != nil {
			Log.Warningf("Failed to dial RPC client for JSON-RPC host: %s", rpcURL)
			return nil, err
		}
		client = erpc
		mutex.Lock()
		ethrpcClients[networkID] = append(networkClients, client)
		mutex.Unlock()
		Log.Debugf("Dialed JSON-RPC host @ %s", rpcURL)
	} else {
		client = ethrpcClients[networkID][0]
		Log.Debugf("Resolved JSON-RPC host @ %s", rpcURL)
	}
	return client, nil
}

// EncodeABI returns the ABI-encoded calldata for the given method and params
func EncodeABI(method *abi.Method, params ...interface{}) ([]byte, error) {
	var methodDescriptor = fmt.Sprintf("method %s", method.Name)
	defer func() {
		if r := recover(); r != nil {
			Log.Debugf("Failed to encode ABI-compliant calldata for method: %s", methodDescriptor)
		}
	}()

	Log.Debugf("Attempting to encode %d parameters prior to executing contract method: %s", len(params), methodDescriptor)
	var args []interface{}

	for i := range params {
		if i >= len(method.Inputs) {
			break
		}
		input := method.Inputs[i]
		Log.Debugf("Attempting to coerce encoding of %v abi parameter; value: %s", input.Type, params[i])
		args = append(args, params[i])
	}

	encodedArgs, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}

	Log.Debugf("Encoded %v abi params prior to executing contract method: %s; abi-encoded arguments %v bytes packed", len(params), methodDescriptor, len(encodedArgs))
	return append(method.Id(), encodedArgs...), nil
}

// GenerateKeyPair - creates and returns an ECDSA keypair;
// the returned *ecdsa.PrivateKey can be encoded with: hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
func GenerateKeyPair() (address *string, privateKey *ecdsa.PrivateKey, err error) {
	privateKey, err = ethcrypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	address = stringOrNil(ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	return address, privateKey, nil
}

// MarshalKeyPairJSON - returns keystore JSON representation of given private key
func MarshalKeyPairJSON(addr common.Address, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	type keyJSON struct {
		ID         string `json:"id"`
		Address    string `json:"address"`
		PrivateKey string `json:"privatekey"`
		Version    int    `json:"version"`
	}
	keyUUID, _ := generateKeyUUID()
	key := keyJSON{
		ID:         keyUUID,
		Address:    hex.EncodeToString(addr[:]),
		PrivateKey: hex.EncodeToString(ethcrypto.FromECDSA(privateKey)),
		Version:    3,
	}
	return json.Marshal(key)
}

// MarshalEncryptedKey encrypts key as version 3.
func MarshalEncryptedKey(addr common.Address, privateKey *ecdsa.PrivateKey, secret string) ([]byte, error) {
	const (
		// n,r,p = 2^12, 8, 6 uses 4MB memory and approx 100ms CPU time on a modern CPU.
		LightScryptN = 1 << 12
		LightScryptP = 6

		scryptR     = 4
		scryptDKLen = 32
	)

	type cipherparamsJSON struct {
		IV string `json:"iv"`
	}

	type cryptoJSON struct {
		Cipher       string                 `json:"cipher"`
		CipherText   string                 `json:"ciphertext"`
		CipherParams cipherparamsJSON       `json:"cipherparams"`
		KDF          string                 `json:"kdf"`
		KDFParams    map[string]interface{} `json:"kdfparams"`
		MAC          string                 `json:"mac"`
	}

	type web3v3 struct {
		ID      string     `json:"id"`
		Address string     `json:"address"`
		Crypto  cryptoJSON `json:"crypto"`
		Version int        `json:"version"`
	}

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		Log.Errorf("Failed while reading from crypto/rand; %s", err.Error())
		return nil, err
	}

	derivedKey, err := scrypt.Key([]byte(secret), salt, LightScryptN, scryptR, LightScryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := ethcrypto.FromECDSA(privateKey)

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		Log.Errorf("Failed while reading from crypto/rand; %s", err.Error())
		return nil, err
	}

	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := ethcrypto.Keccak256(derivedKey[16:32], cipherText)

	keyUUID, _ := generateKeyUUID()

	return json.Marshal(web3v3{
		ID:      keyUUID,
		Address: hex.EncodeToString(addr[:]),
		Crypto: cryptoJSON{
			Cipher:     "aes-128-ctr",
			CipherText: hex.EncodeToString(cipherText),
			CipherParams: cipherparamsJSON{
				IV: hex.EncodeToString(iv),
			},
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     LightScryptN,
				"r":     scryptR,
				"p":     LightScryptP,
				"dklen": scryptDKLen,
				"salt":  hex.EncodeToString(salt),
			},
			MAC: hex.EncodeToString(mac),
		},
		Version: 3,
	})
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func generateKeyUUID() (string, error) {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:]), nil
}

// HashFunctionSelector returns the first 4 bytes of the Keccak256 hash of the given function selector
func HashFunctionSelector(sel string) string {
	hash := Keccak256(sel)
	return common.Bytes2Hex(hash[0:4])
}

// Keccak256 hash the given string
func Keccak256(str string) []byte {
	return ethcrypto.Keccak256([]byte(str))
}

// Transaction broadcast helpers

// BroadcastTx injects a signed transaction into the pending pool for execution.
func BroadcastTx(ctx context.Context, networkID, rpcURL string, tx *types.Transaction, client *ethclient.Client, result interface{}) error {
	rpcClient, err := ResolveJsonRpcClient(networkID, rpcURL)
	if err != nil {
		return err
	}

	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}

	return rpcClient.CallContext(ctx, result, "eth_sendRawTransaction", common.ToHex(data))
}

// BroadcastSignedTx emits a given signed tx for inclusion in a block
func BroadcastSignedTx(networkID, rpcURL string, signedTx *types.Transaction) error {
	client, err := DialJsonRpc(networkID, rpcURL)
	if err != nil {
		return fmt.Errorf("Failed to dial JSON-RPC host; %s", err.Error())
	} else if signedTx != nil {
		Log.Debugf("Transmitting signed tx to JSON-RPC host")
		err = BroadcastTx(context.TODO(), networkID, rpcURL, signedTx, client, nil)
		if err != nil {
			return fmt.Errorf("Failed to transmit signed tx to JSON-RPC host; %s", err.Error())
		}
	}
	return nil
}

// SignTx signs a transaction using the given private key and calldata;
// providing 0 gas results in the tx attempting to use up to the block
// gas limit for execution
func SignTx(networkID, rpcURL, from, privateKey string, to, data *string, val *big.Int, gasLimit uint64) (*types.Transaction, *string, error) {
	client, err := DialJsonRpc(networkID, rpcURL)
	if err != nil {
		return nil, nil, err
	}
	_, err = GetSyncProgress(client)
	if err == nil {
		cfg := GetChainConfig(networkID, rpcURL)
		blockNumber, err := GetLatestBlockNumber(networkID, rpcURL)
		if err != nil {
			return nil, nil, err
		}
		nonce, err := client.PendingNonceAt(context.TODO(), common.HexToAddress(from))
		if err != nil {
			return nil, nil, err
		}
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		var _data []byte
		if data != nil {
			_data = common.FromHex(*data)
		}

		var tx *types.Transaction

		if gasLimit == 0 {
			callMsg := asCallMsg(from, data, to, val, gasPrice.Uint64(), gasLimit)
			gasLimit, err = client.EstimateGas(context.TODO(), callMsg)
		}

		if to != nil {
			addr := common.HexToAddress(*to)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to estimate gas for tx; %s", err.Error())
			}
			Log.Debugf("Estimated %d total gas required for tx with %d-byte data payload", gasLimit, len(_data))
			tx = types.NewTransaction(nonce, addr, val, gasLimit, gasPrice, _data)
		} else {
			Log.Debugf("Attempting to deploy contract via tx; network: %s", networkID)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to estimate gas for tx; %s", err.Error())
			}
			Log.Debugf("Estimated %d total gas required for contract deployment tx with %d-byte data payload", gasLimit, len(_data))
			tx = types.NewContractCreation(nonce, val, gasLimit, gasPrice, _data)
		}
		signer := types.MakeSigner(cfg, big.NewInt(int64(blockNumber)))
		hash := signer.Hash(tx).Bytes()
		Log.Debugf("Signing tx on behalf of %s", from)
		_privateKey, err := ethcrypto.HexToECDSA(privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed read private key bytes prior to signing tx; %s", err.Error())
		}
		sig, err := ethcrypto.Sign(hash, _privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to sign tx on behalf of %s; %s", *to, err.Error())
		}
		if err == nil {
			signedTx, _ := tx.WithSignature(signer, sig)
			hash := stringOrNil(fmt.Sprintf("0x%x", signedTx.Hash()))
			signedTxJSON, _ := signedTx.MarshalJSON()
			Log.Debugf("Signed tx for broadcast via JSON-RPC: %s", signedTxJSON)
			return signedTx, hash, nil
		}
		return nil, nil, err
	}
	return nil, nil, err
}

// Calldata construction helpers

func asCallMsg(from string, data, to *string, val *big.Int, gasPrice, gasLimit uint64) ethereum.CallMsg {
	var _to *common.Address
	var _data []byte
	if to != nil {
		addr := common.HexToAddress(*to)
		_to = &addr
	}
	if data != nil {
		_data = common.FromHex(*data)
	}
	return ethereum.CallMsg{
		From:     common.HexToAddress(from),
		To:       _to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    val,
		Data:     _data,
	}
}

func parseContractABI(contractAbi interface{}) (*abi.ABI, error) {
	abistr, err := json.Marshal(contractAbi)
	if err != nil {
		Log.Warningf("Failed to marshal ABI from contract params to json; %s", err.Error())
		return nil, err
	}

	abival, err := abi.JSON(strings.NewReader(string(abistr)))
	if err != nil {
		Log.Warningf("Failed to initialize ABI from contract  params to json; %s", err.Error())
		return nil, err
	}

	return &abival, nil
}

// EthCall invokes eth_call manually via JSON-RPC
func EthCall(networkID, rpcURL string, params []interface{}) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_call", params, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// GetBlockNumber retrieves the latest block known to the JSON-RPC client
func GetBlockNumber(networkID, rpcURL string) *uint64 {
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch latest block number via JSON-RPC eth_blockNumber method")
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_blockNumber", params, &resp)
	if err != nil {
		Log.Warningf("Failed to invoke eth_blockNumber method via JSON-RPC; %s", err.Error())
		return nil
	}
	blockNumber, err := hexutil.DecodeBig(resp.Result.(string))
	if err != nil {
		return nil
	}
	_blockNumber := blockNumber.Uint64()
	return &_blockNumber
}

// GetChainConfig parses the cached network config mapped to the given
// `networkID`, if one exists; otherwise, the mainnet chain config is returned.
func GetChainConfig(networkID, rpcURL string) *params.ChainConfig {
	if cfg, ok := chainConfigs[networkID]; ok {
		return cfg
	}
	cfg := params.MainnetChainConfig
	chainID, err := strconv.ParseUint(networkID, 10, 0)
	if err != nil {
		cfg.ChainID = big.NewInt(int64(chainID))
		chainConfigs[networkID] = cfg
	}
	return cfg
}

// GetChainID retrieves the current chainID via JSON-RPC
func GetChainID(networkID, rpcURL string) *big.Int {
	ethClient, err := DialJsonRpc(networkID, rpcURL)
	if err != nil {
		Log.Warningf("Failed to read network id for *ethclient.Client instance: %s; %s", ethClient, err.Error())
		return nil
	}
	if ethClient == nil {
		Log.Warningf("Failed to read network id for unresolved *ethclient.Client instance; network id: %s; JSON-RPC URL: %s", networkID, rpcURL)
		return nil
	}
	chainID, err := ethClient.NetworkID(context.TODO())
	if err != nil {
		Log.Warningf("Failed to read network id for *ethclient.Client instance: %s; %s", ethClient, err.Error())
		return nil
	}
	if chainID != nil {
		Log.Debugf("Received chain id from *ethclient.Client instance: %s", ethClient, chainID)
	}
	return chainID
}

// GetGasPrice returns the gas price
func GetGasPrice(networkID, rpcURL string) *string {
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch gas price via JSON-RPC eth_gasPrice method")
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_gasPrice", params, &resp)
	if err != nil {
		Log.Warningf("Failed to invoke eth_gasPrice method via JSON-RPC; %s", err.Error())
		return nil
	}
	return stringOrNil(resp.Result.(string))
}

// GetLatestBlock retrieves the latsest block
func GetLatestBlock(networkID, rpcURL string) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_getBlockByNumber", []interface{}{"latest", true}, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// GetLatestBlockNumber retrieves the latest block number
func GetLatestBlockNumber(networkID, rpcURL string) (uint64, error) {
	resp, err := GetLatestBlock(networkID, rpcURL)
	if err != nil {
		return 0, err
	}
	blockNumberStr, blockNumberStrOk := resp.Result.(map[string]interface{})["number"].(string)
	if !blockNumberStrOk {
		return 0, errors.New("Unable to parse block number from JSON-RPC response")
	}
	blockNumber, err := hexutil.DecodeUint64(blockNumberStr)
	if err != nil {
		return 0, fmt.Errorf("Unable to decode block number hex; %s", err.Error())
	}
	return blockNumber, nil
}

// GetBlockGasLimit retrieves the latest block gas limit
func GetBlockGasLimit(networkID, rpcURL string) (uint64, error) {
	resp, err := GetLatestBlock(networkID, rpcURL)
	if err != nil {
		return 0, err
	}
	blockGasLimitStr, blockGasLimitStrOk := resp.Result.(map[string]interface{})["gasLimit"].(string)
	if !blockGasLimitStrOk {
		return 0, errors.New("Unable to parse block gas limit from JSON-RPC response")
	}
	blockGasLimit, err := hexutil.DecodeUint64(blockGasLimitStr)
	if err != nil {
		return 0, fmt.Errorf("Unable to decode block gas limit hex; %s", err.Error())
	}
	return blockGasLimit, nil
}

// GetBlockByNumber retrieves a given block by number
func GetBlockByNumber(networkID, rpcURL string, blockNumber uint64) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_getBlockByNumber", []interface{}{hexutil.EncodeUint64(blockNumber), true}, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// GetHeaderByNumber retrieves a given block header by number
func GetHeaderByNumber(networkID, rpcURL string, blockNumber uint64) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_getHeaderByNumber", []interface{}{hexutil.EncodeUint64(blockNumber), true}, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// GetNativeBalance retrieves a wallet's native currency balance
func GetNativeBalance(networkID, rpcURL, addr string) (*big.Int, error) {
	client, err := DialJsonRpc(networkID, rpcURL)
	if err != nil {
		return nil, err
	}
	return client.BalanceAt(context.TODO(), common.HexToAddress(addr), nil)
}

// GetNetworkStatus retrieves current metadata from the JSON-RPC client;
// returned struct includes block height, chainID, number of connected peers,
// protocol version, and syncing state.
func GetNetworkStatus(networkID, rpcURL string) (*NetworkStatus, error) {
	ethClient, err := DialJsonRpc(networkID, rpcURL)
	if err != nil || rpcURL == "" || ethClient == nil {
		meta := map[string]interface{}{
			"error": nil,
		}
		if err != nil {
			Log.Warningf("Failed to dial JSON-RPC host: %s; %s", rpcURL, err.Error())
			meta["error"] = err.Error()
		} else if rpcURL == "" {
			meta["error"] = "No 'full-node' JSON-RPC URL configured or resolvable"
		} else if ethClient == nil {
			meta["error"] = "Configured 'full-node' JSON-RPC client not resolved"
		}
		return &NetworkStatus{
			State: stringOrNil("configuring"),
			Meta:  meta,
		}, nil
	}

	defer func() {
		if r := recover(); r != nil {
			Log.Debugf("Recovered from failed attempt to retrieve network sync progress from JSON-RPC host: %s", rpcURL)
			clearCachedClients(networkID)
		}
	}()

	syncProgress, err := GetSyncProgress(ethClient)
	if err != nil {
		Log.Warningf("Failed to read network sync progress using JSON-RPC host; %s", err.Error())
		clearCachedClients(networkID)
		return nil, err
	}
	var state string
	var block uint64        // current block; will be less than height while syncing in progress
	var height *uint64      // total number of blocks
	var lastBlockAt *uint64 // unix timestamp of last block
	chainID := GetChainID(networkID, rpcURL)
	peers := GetPeerCount(networkID, rpcURL)
	protocolVersion := GetProtocolVersion(networkID, rpcURL)
	meta := map[string]interface{}{}
	var syncing = false
	if syncProgress == nil {
		state = "synced"
		resp, err := GetLatestBlock(networkID, rpcURL)
		if err != nil {
			Log.Warningf("Failed to read latest block for %s using JSON-RPC host; %s", rpcURL, err.Error())
			return nil, err
		}
		hdr := resp.Result.(map[string]interface{})
		delete(hdr, "transactions") // HACK
		delete(hdr, "uncles")       // HACK

		meta["last_block_header"] = hdr
		block, err = hexutil.DecodeUint64(hdr["number"].(string))
		if err != nil {
			return nil, fmt.Errorf("Unable to decode block number hex; %s", err.Error())
		}

		_lastBlockAt, err := hexutil.DecodeUint64(hdr["timestamp"].(string))
		if err != nil {
			return nil, fmt.Errorf("Unable to decode block timestamp hex; %s", err.Error())
		}
		lastBlockAt = &_lastBlockAt
	} else {
		block = syncProgress.CurrentBlock
		height = &syncProgress.HighestBlock
		syncing = true
	}
	return &NetworkStatus{
		Block:           block,
		Height:          height,
		ChainID:         stringOrNil(hexutil.EncodeBig(chainID)),
		PeerCount:       peers,
		LastBlockAt:     lastBlockAt,
		ProtocolVersion: protocolVersion,
		State:           stringOrNil(state),
		Syncing:         syncing,
		Meta:            meta,
	}, nil
}

// GetPeerCount returns the number of peers currently connected to the JSON-RPC client
func GetPeerCount(networkID, rpcURL string) uint64 {
	var peerCount uint64
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch peer count via net_peerCount method via JSON-RPC")
	err := InvokeJsonRpcClient(networkID, rpcURL, "net_peerCount", params, &resp)
	if err != nil {
		Log.Debugf("Attempting to fetch peer count via parity_netPeers method via JSON-RPC")
		err := InvokeJsonRpcClient(networkID, rpcURL, "parity_netPeers", params, &resp)
		Log.Warningf("Failed to invoke parity_netPeers method via JSON-RPC; %s", err.Error())
		return 0
	}
	if peerCountStr, ok := resp.Result.(string); ok {
		peerCount, err = hexutil.DecodeUint64(peerCountStr)
		if err != nil {
			return 0
		}
	}
	return peerCount
}

// GetProtocolVersion returns the JSON-RPC client protocol version
func GetProtocolVersion(networkID, rpcURL string) *string {
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch protocol version via JSON-RPC eth_protocolVersion method")
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_protocolVersion", params, &resp)
	if err != nil {
		Log.Debugf("Attempting to fetch protocol version via JSON-RPC net_version method")
		err := InvokeJsonRpcClient(networkID, rpcURL, "net_version", params, &resp)

		Log.Warningf("Failed to invoke eth_protocolVersion method via JSON-RPC; %s", err.Error())
		return nil
	}
	return stringOrNil(resp.Result.(string))
}

// GetCode retrieves the code stored at the named address in the given scope;
// scope can be a block number, latest, earliest or pending
func GetCode(networkID, rpcURL, addr, scope string) (*string, error) {
	params := make([]interface{}, 0)
	params = append(params, addr)
	params = append(params, scope)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch code from %s via eth_getCode JSON-RPC method", addr)
	err := InvokeJsonRpcClient(networkID, rpcURL, "eth_getCode", params, &resp)
	if err != nil {
		Log.Warningf("Failed to invoke eth_getCode method via JSON-RPC; %s", err.Error())
		return nil, err
	}
	return stringOrNil(resp.Result.(string)), nil
}

// GetSyncProgress retrieves the status of the current network sync
func GetSyncProgress(client *ethclient.Client) (*ethereum.SyncProgress, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	progress, err := client.SyncProgress(ctx)
	if err != nil {
		Log.Warningf("Failed to read sync progress for *ethclient.Client instance: %s; %s", client, err.Error())
		cancel()
		return nil, err
	}
	if progress != nil {
		Log.Debugf("Latest synced block reported by *ethclient.Client instance: %v [of %v]", client, progress.CurrentBlock, progress.HighestBlock)
	}
	cancel()
	return progress, nil
}

// GetTokenBalance retrieves a token balance for a specific token contract and network address
func GetTokenBalance(networkID, rpcURL, tokenAddr, addr string, contractABI interface{}) (*big.Int, error) {
	var balance *big.Int
	abi, err := parseContractABI(contractABI)
	if err != nil {
		return nil, err
	}
	client, err := DialJsonRpc(networkID, rpcURL)
	gasPrice, _ := client.SuggestGasPrice(context.TODO())
	to := common.HexToAddress(tokenAddr)
	msg := ethereum.CallMsg{
		From:     common.HexToAddress(addr),
		To:       &to,
		Gas:      0,
		GasPrice: gasPrice,
		Value:    nil,
		Data:     common.FromHex(HashFunctionSelector("balanceOf(address)")),
	}
	result, _ := client.CallContract(context.TODO(), msg, nil)
	if method, ok := abi.Methods["balanceOf"]; ok {
		method.Outputs.Unpack(&balance, result)
		if balance != nil {
			symbol, _ := GetTokenSymbol(networkID, rpcURL, addr, tokenAddr, contractABI)
			Log.Debugf("Read %s token balance (%v) from token contract address: %s", symbol, balance, addr)
		}
	} else {
		Log.Warningf("Unable to read balance of unsupported token contract address: %s", tokenAddr)
	}
	return balance, nil
}

// GetTokenSymbol attempts to retrieve the symbol of a token presumed to be deployed at the given token contract address
func GetTokenSymbol(networkID, rpcURL, from, tokenAddr string, contractABI interface{}) (*string, error) {
	client, err := DialJsonRpc(networkID, rpcURL)
	if err != nil {
		return nil, err
	}
	_abi, err := parseContractABI(contractABI)
	if err != nil {
		return nil, err
	}
	to := common.HexToAddress(tokenAddr)
	msg := ethereum.CallMsg{
		From:     common.HexToAddress(from),
		To:       &to,
		Gas:      0,
		GasPrice: big.NewInt(0),
		Value:    nil,
		Data:     common.FromHex(HashFunctionSelector("symbol()")),
	}
	result, _ := client.CallContract(context.TODO(), msg, nil)
	var symbol string
	if method, ok := _abi.Methods["symbol"]; ok {
		err = method.Outputs.Unpack(&symbol, result)
		if err != nil {
			Log.Warningf("Failed to read token symbol from deployed token contract %s; %s", tokenAddr, err.Error())
		}
	}
	return stringOrNil(symbol), nil
}

// TraceTx returns the VM traces; requires parity JSON-RPC client and the node must
// be configured with `--fat-db on --tracing on --pruning archive`
func TraceTx(networkID, rpcURL string, hash *string) (interface{}, error) {
	var addr = *hash
	if !strings.HasPrefix(addr, "0x") {
		addr = fmt.Sprintf("0x%s", addr)
	}
	params := make([]interface{}, 0)
	params = append(params, addr)
	var result = &EthereumTxTraceResponse{}
	Log.Debugf("Attempting to trace tx via trace_transaction method via JSON-RPC; tx hash: %s", addr)
	err := InvokeJsonRpcClient(networkID, rpcURL, "trace_transaction", params, &result)
	if err != nil {
		Log.Warningf("Failed to invoke trace_transaction method via JSON-RPC; %s", err.Error())
		return nil, err
	}
	return result, nil
}

// GetTxReceipt retrieves the full transaction receipt via JSON-RPC given the transaction hash
func GetTxReceipt(networkID, rpcURL, txHash, from string) (*types.Receipt, error) {
	client, err := DialJsonRpc(networkID, rpcURL)
	if err != nil {
		Log.Warningf("Failed to retrieve tx receipt for broadcast tx: %s; %s", txHash, err.Error())
		return nil, err
	}
	Log.Debugf("Attempting to retrieve tx receipt for broadcast tx: %s", txHash)
	return client.TransactionReceipt(context.TODO(), common.HexToHash(txHash))
}
