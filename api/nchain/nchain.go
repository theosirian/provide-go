package nchain

import (
	"fmt"
	"os"

	"github.com/provideservices/provide-go/api"
	"github.com/provideservices/provide-go/common"
)

const defaultNChainHost = "nchain.provide.services"
const defaultNChainPath = "api/v1"
const defaultNChainScheme = "https"

// Service for the nchain api
type Service struct {
	api.Client
}

// InitNChain convenience method to initialize an `nchain.Service` instance
func InitNChain(token string) *Service {
	host := defaultNChainHost
	if os.Getenv("NCHAIN_API_HOST") != "" {
		host = os.Getenv("NCHAIN_API_HOST")
	}

	path := defaultNChainPath
	if os.Getenv("NCHAIN_API_PATH") != "" {
		host = os.Getenv("NCHAIN_API_PATH")
	}

	scheme := defaultNChainScheme
	if os.Getenv("NCHAIN_API_SCHEME") != "" {
		scheme = os.Getenv("NCHAIN_API_SCHEME")
	}

	return &Service{
		api.Client{
			Host:   host,
			Path:   path,
			Scheme: scheme,
			Token:  common.StringOrNil(token),
		},
	}
}

// CreateAccount
func CreateAccount(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("accounts", params)
}

// ListAccounts
func ListAccounts(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("accounts", params)
}

// GetAccountDetails
func GetAccountDetails(token, accountID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("accounts/%s", accountID)
	return InitNChain(token).Get(uri, params)
}

// GetAccountBalance
func GetAccountBalance(token, accountID, tokenID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("accounts/%s/balances/%s", accountID, tokenID)
	return InitNChain(token).Get(uri, params)
}

// CreateBridge
func CreateBridge(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("bridges", params)
}

// ListBridges
func ListBridges(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("bridges", params)
}

// GetBridgeDetails
func GetBridgeDetails(token, bridgeID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("bridges/%s", bridgeID)
	return InitNChain(token).Get(uri, params)
}

// CreateConnector
func CreateConnector(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("connectors", params)
}

// ListConnectors
func ListConnectors(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("connectors", params)
}

// GetConnectorDetails
func GetConnectorDetails(token, connectorID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("connectors/%s", connectorID)
	return InitNChain(token).Get(uri, params)
}

// DeleteConnector
func DeleteConnector(token, connectorID string) (int, interface{}, error) {
	uri := fmt.Sprintf("connectors/%s", connectorID)
	return InitNChain(token).Delete(uri)
}

// CreateContract
func CreateContract(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("contracts", params)
}

// ExecuteContract
func ExecuteContract(token, contractID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("contracts/%s/execute", contractID)
	return InitNChain(token).Post(uri, params)
}

// ListContracts
func ListContracts(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("contracts", params)
}

// GetContractDetails
func GetContractDetails(token, contractID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("contracts/%s", contractID)
	return InitNChain(token).Get(uri, params)
}

// CreateNetwork
func CreateNetwork(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("networks", params)
}

// UpdateNetwork updates an existing network
func UpdateNetwork(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s", networkID)
	return InitNChain(token).Put(uri, params)
}

// ListNetworks
func ListNetworks(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("networks", params)
}

// GetNetworkDetails
func GetNetworkDetails(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkAccounts
func ListNetworkAccounts(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/accounts", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkBlocks
func ListNetworkBlocks(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/blocks", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkBridges
func ListNetworkBridges(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/bridges", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkConnectors
func ListNetworkConnectors(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/connectors", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkContracts
func ListNetworkContracts(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/contracts", networkID)
	return InitNChain(token).Get(uri, params)
}

// GetNetworkContractDetails
func GetNetworkContractDetails(token, networkID, contractID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/contracts/%s", networkID, contractID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkOracles
func ListNetworkOracles(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/oracles", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkTokens
func ListNetworkTokens(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/tokens", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkTransactions
func ListNetworkTransactions(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/transactions", networkID)
	return InitNChain(token).Get(uri, params)
}

// GetNetworkTransactionDetails
func GetNetworkTransactionDetails(token, networkID, txID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/transactions/%s", networkID, txID)
	return InitNChain(token).Get(uri, params)
}

// GetNetworkStatusMeta
func GetNetworkStatusMeta(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/status", networkID)
	return InitNChain(token).Get(uri, params)
}

// ListNetworkNodes
func ListNetworkNodes(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/nodes", networkID)
	return InitNChain(token).Get(uri, params)
}

// CreateNetworkNode
func CreateNetworkNode(token, networkID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/nodes", networkID)
	return InitNChain(token).Post(uri, params)
}

// GetNetworkNodeDetails
func GetNetworkNodeDetails(token, networkID, nodeID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/nodes/%s", networkID, nodeID)
	return InitNChain(token).Get(uri, params)
}

// GetNetworkNodeLogs
func GetNetworkNodeLogs(token, networkID, nodeID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/nodes/%s/logs", networkID, nodeID)
	return InitNChain(token).Get(uri, params)
}

// DeleteNetworkNode
func DeleteNetworkNode(token, networkID, nodeID string) (int, interface{}, error) {
	uri := fmt.Sprintf("networks/%s/nodes/%s", networkID, nodeID)
	return InitNChain(token).Delete(uri)
}

// CreateOracle
func CreateOracle(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("oracles", params)
}

// ListOracles
func ListOracles(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("oracles", params)
}

// GetOracleDetails
func GetOracleDetails(token, oracleID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("oracles/%s", oracleID)
	return InitNChain(token).Get(uri, params)
}

// CreateTokenContract
func CreateTokenContract(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("tokens", params)
}

// ListTokenContracts
func ListTokenContracts(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("tokens", params)
}

// GetTokenContractDetails
func GetTokenContractDetails(token, tokenID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("tokens/%s", tokenID)
	return InitNChain(token).Get(uri, params)
}

// CreateTransaction
func CreateTransaction(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("transactions", params)
}

// ListTransactions
func ListTransactions(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("transactions", params)
}

// GetTransactionDetails
func GetTransactionDetails(token, txID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("transactions/%s", txID)
	return InitNChain(token).Get(uri, params)
}

// CreateWallet
func CreateWallet(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Post("wallets", params)
}

// ListWallets
func ListWallets(token string, params map[string]interface{}) (int, interface{}, error) {
	return InitNChain(token).Get("wallets", params)
}

// GetWalletDetails
func GetWalletDetails(token, walletID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("wallets/%s", walletID)
	return InitNChain(token).Get(uri, params)
}

// ListWalletAccounts
func ListWalletAccounts(token, walletID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("wallets/%s/accounts", walletID)
	return InitNChain(token).Get(uri, params)
}
