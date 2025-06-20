package aptos

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"github.com/hasura/go-graphql-client"
)

// NetworkConfig a configuration for the Client and which network to use.  Use one of the preconfigured [LocalnetConfig], [DevnetConfig], [TestnetConfig], or [MainnetConfig] unless you have your own full node.
//
// Name, ChainId, IndexerUrl, FaucetUrl are not required.
//
// If ChainId is 0, the ChainId wil be fetched on-chain
// If IndexerUrl or FaucetUrl are an empty string "", clients will not be made for them.
type NetworkConfig struct {
	Name       string
	ChainId    uint8
	NodeUrl    string
	IndexerUrl string
	FaucetUrl  string
}

// LocalnetConfig is for use with a localnet, created by the [Aptos CLI](https://aptos.dev/tools/aptos-cli)
//
// To start a localnet, install the Aptos CLI then run:
//
//	aptos node run-localnet --with-indexer-api
var LocalnetConfig = NetworkConfig{
	Name:    "localnet",
	ChainId: 4,
	// We use 127.0.0.1 as it is more foolproof than localhost
	NodeUrl:    "http://127.0.0.1:8080/v1",
	IndexerUrl: "http://127.0.0.1:8090/v1/graphql",
	FaucetUrl:  "http://127.0.0.1:8081",
}

// DevnetConfig is for use with devnet.  Note devnet resets at least weekly.  ChainId differs after each reset.
var DevnetConfig = NetworkConfig{
	Name:       "devnet",
	NodeUrl:    "https://api.devnet.aptoslabs.com/v1",
	IndexerUrl: "https://api.devnet.aptoslabs.com/v1/graphql",
	FaucetUrl:  "https://faucet.devnet.aptoslabs.com/",
}

// TestnetConfig is for use with testnet. Testnet does not reset.
var TestnetConfig = NetworkConfig{
	Name:       "testnet",
	ChainId:    2,
	NodeUrl:    "https://api.testnet.aptoslabs.com/v1",
	IndexerUrl: "https://api.testnet.aptoslabs.com/v1/graphql",
	FaucetUrl:  "https://faucet.testnet.aptoslabs.com/",
}

// MainnetConfig is for use with mainnet.  There is no faucet for Mainnet, as these are real user assets.
var MainnetConfig = NetworkConfig{
	Name:       "mainnet",
	ChainId:    1,
	NodeUrl:    "https://api.mainnet.aptoslabs.com/v1",
	IndexerUrl: "https://api.mainnet.aptoslabs.com/v1/graphql",
	FaucetUrl:  "",
}

// NamedNetworks Map from network name to NetworkConfig
var NamedNetworks map[string]NetworkConfig

func init() {
	NamedNetworks = make(map[string]NetworkConfig, 4)
	setNN := func(nc NetworkConfig) {
		NamedNetworks[nc.Name] = nc
	}
	setNN(LocalnetConfig)
	setNN(DevnetConfig)
	setNN(TestnetConfig)
	setNN(MainnetConfig)
}

// PollPeriod is an option to PollForTransactions
type PollPeriod time.Duration

// PollTimeout is an option to PollForTransactions
type PollTimeout time.Duration

// EstimateGasUnitPrice estimates the gas unit price for a transaction
type EstimateGasUnitPrice bool

// EstimateMaxGasAmount estimates the max gas amount for a transaction
type EstimateMaxGasAmount bool

// EstimatePrioritizedGasUnitPrice estimates the prioritized gas unit price for a transaction
type EstimatePrioritizedGasUnitPrice bool

// MaxGasAmount will set the max gas amount in gas units for a transaction
type MaxGasAmount uint64

// GasUnitPrice will set the gas unit price in octas (1/10^8 APT) for a transaction
type GasUnitPrice uint64

// ExpirationSeconds will set the number of seconds from the current time to expire a transaction
type ExpirationSeconds uint64

// FeePayer will set the fee payer for a transaction
type FeePayer *AccountAddress

// (Deprecated) FeePayerPublicKey will construct authenticator from public key.
type FeePayerPublicKey crypto.PublicKey

// AdditionalSigners will set the additional signers for a transaction
type AdditionalSigners []AccountAddress

// SequenceNumber will set the sequence number for a transaction
type SequenceNumber uint64

// ChainIdOption will set the chain ID for a transaction
// TODO: This one may want to be removed / renamed?
type ChainIdOption uint8

// AptosClient is an interface for all functionality on the Client.
// It is a combination of [AptosRpcClient], [AptosIndexerClient], and [AptosFaucetClient] for the purposes
// of mocking and convenience.
type AptosClient interface {
	AptosRpcClient
	AptosIndexerClient
	AptosFaucetClient
}

// AptosRpcClient is an interface for all functionality on the Client that is Node RPC related.  Its main implementation
// is [NodeClient]
type AptosRpcClient interface {
	// SetTimeout adjusts the HTTP client timeout
	//
	//	client.SetTimeout(5 * time.Millisecond)
	SetTimeout(timeout time.Duration)

	// SetHeader sets the header for all future requests
	//
	//	client.SetHeader("Authorization", "Bearer abcde")
	SetHeader(key string, value string)

	// RemoveHeader removes the header from being automatically set all future requests.
	//
	//	client.RemoveHeader("Authorization")
	RemoveHeader(key string)

	// Info Retrieves the node info about the network and it's current state
	Info() (NodeInfo, error)

	// Account Retrieves information about the account such as [SequenceNumber] and [crypto.AuthenticationKey]
	Account(address AccountAddress, ledgerVersion ...uint64) (AccountInfo, error)

	// AccountResource Retrieves a single resource given its struct name.
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore")
	//
	// Can also fetch at a specific ledger version
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore", 1)
	AccountResource(address AccountAddress, resourceType string, ledgerVersion ...uint64) (map[string]any, error)

	// AccountResources fetches resources for an account into a JSON-like map[string]any in AccountResourceInfo.Data
	// For fetching raw Move structs as BCS, See #AccountResourcesBCS
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResources(address)
	//
	// Can also fetch at a specific ledger version
	//
	//	address := AccountOne
	//	dataMap, _ := client.AccountResource(address, 1)
	AccountResources(address AccountAddress, ledgerVersion ...uint64) ([]AccountResourceInfo, error)

	// AccountResourcesBCS fetches account resources as raw Move struct BCS blobs in AccountResourceRecord.Data []byte
	AccountResourcesBCS(address AccountAddress, ledgerVersion ...uint64) ([]AccountResourceRecord, error)

	// AccountModule fetches a single account module's bytecode and ABI from on-chain state.
	AccountModule(address AccountAddress, moduleName string, ledgerVersion ...uint64) (*api.MoveBytecode, error)

	// EntryFunctionWithArgs generates an EntryFunction from on-chain Module ABI, and converts simple inputs to BCS encoded ones.
	EntryFunctionWithArgs(moduleAddress AccountAddress, moduleName string, functionName string, typeArgs []any, args []any, options ...any) (*EntryFunction, error)

	// BlockByHeight fetches a block by height
	//
	//	block, _ := client.BlockByHeight(1, false)
	//
	// Can also fetch with transactions
	//
	//	block, _ := client.BlockByHeight(1, true)
	BlockByHeight(blockHeight uint64, withTransactions bool) (*api.Block, error)

	// BlockByVersion fetches a block by ledger version
	//
	//	block, _ := client.BlockByVersion(123, false)
	//
	// Can also fetch with transactions
	//
	//	block, _ := client.BlockByVersion(123, true)
	BlockByVersion(ledgerVersion uint64, withTransactions bool) (*api.Block, error)

	// TransactionByHash gets info on a transaction
	// The transaction may be pending or recently committed.
	//
	//	data, err := client.TransactionByHash("0xabcd")
	//	if err != nil {
	//		if httpErr, ok := err.(aptos.HttpError) {
	//			if httpErr.StatusCode == 404 {
	//				// if we're sure this has been submitted, assume it is still pending elsewhere in the mempool
	//			}
	//		}
	//	} else {
	//		if data["type"] == "pending_transaction" {
	//			// known to local mempool, but not committed yet
	//		}
	//	}
	TransactionByHash(txnHash string) (*api.Transaction, error)

	// WaitTransactionByHash waits for a transaction to be confirmed by its hash.
	// This function allows you to monitor the status of a transaction until it is finalized.
	WaitTransactionByHash(txnHash string) (*api.Transaction, error)

	// TransactionByVersion gets info on a transaction from its LedgerVersion.  It must have been
	// committed to have a ledger version
	//
	//	data, err := client.TransactionByVersion("0xabcd")
	//	if err != nil {
	//		if httpErr, ok := err.(aptos.HttpError) {
	//			if httpErr.StatusCode == 404 {
	//				// if we're sure this has been submitted, the full node might not be caught up to this version yet
	//			}
	//		}
	//	}
	TransactionByVersion(version uint64) (*api.CommittedTransaction, error)

	// PollForTransaction waits up to 10 seconds for a transaction to be done, polling at 10Hz
	// Accepts options PollPeriod and PollTimeout which should wrap time.Duration values.
	// Not just a degenerate case of PollForTransactions, it may return additional information for the single transaction polled.
	PollForTransaction(hash string, options ...any) (*api.UserTransaction, error)

	// PollForTransactions Waits up to 10 seconds for transactions to be done, polling at 10Hz
	// Accepts options PollPeriod and PollTimeout which should wrap time.Duration values.
	//
	//	hashes := []string{"0x1234", "0x4567"}
	//	err := client.PollForTransactions(hashes)
	//
	// Can additionally configure different options
	//
	//	hashes := []string{"0x1234", "0x4567"}
	//	err := client.PollForTransactions(hashes, PollPeriod(500 * time.Milliseconds), PollTimeout(5 * time.Seconds))
	PollForTransactions(txnHashes []string, options ...any) error

	// WaitForTransaction Do a long-GET for one transaction and wait for it to complete
	//
	//	data, err := client.WaitForTransaction("0x1234")
	WaitForTransaction(txnHash string, options ...any) (*api.UserTransaction, error)

	// Transactions Get recent transactions.
	// Start is a version number. Nil for most recent transactions.
	// Limit is a number of transactions to return. 'about a hundred' by default.
	//
	//	client.Transactions(0, 2)   // Returns 2 transactions
	//	client.Transactions(1, 100) // Returns 100 transactions
	Transactions(start *uint64, limit *uint64) ([]*api.CommittedTransaction, error)

	// AccountTransactions Get transactions associated with an account.
	// Start is a version number. Nil for most recent transactions.
	// Limit is a number of transactions to return. 'about a hundred' by default.
	//
	//	client.AccountTransactions(AccountOne, 0, 2)   // Returns 2 transactions for 0x1
	//	client.AccountTransactions(AccountOne, 1, 100) // Returns 100 transactions for 0x1
	AccountTransactions(address AccountAddress, start *uint64, limit *uint64) ([]*api.CommittedTransaction, error)

	// EventsByHandle retrieves events by event handle and field name for a given account.
	//
	// Arguments:
	//   - account - The account address to get events for
	//   - eventHandle - The event handle struct tag
	//   - fieldName - The field in the event handle struct
	//   - start - The starting sequence number. nil for most recent events
	//   - limit - The number of events to return, 100 by default
	EventsByHandle(
		account AccountAddress,
		eventHandle string,
		fieldName string,
		start *uint64,
		limit *uint64,
	) ([]*api.Event, error)

	// EventsByCreationNumber retrieves events by creation number for a given account.
	//
	// Arguments:
	//   - account - The account address to get events for
	//   - creationNumber - The creation number identifying the event
	//   - start - The starting sequence number, nil for most recent events
	//   - limit - The number of events to return, 100 by default
	EventsByCreationNumber(
		account AccountAddress,
		creationNumber string,
		start *uint64,
		limit *uint64,
	) ([]*api.Event, error)

	// SubmitTransaction Submits an already signed transaction to the blockchain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "aptos_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	//	signedTxn, _ := sender.SignTransaction(rawTxn)
	//	submitResponse, err := client.SubmitTransaction(signedTxn)
	SubmitTransaction(signedTransaction *SignedTransaction) (*api.SubmitTransactionResponse, error)

	// BatchSubmitTransaction submits a collection of signed transactions to the network in a single request
	//
	// It will return the responses in the same order as the input transactions that failed.  If the response is empty, then
	// all transactions succeeded.
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "aptos_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	//	signedTxn, _ := sender.SignTransaction(rawTxn)
	//	submitResponse, err := client.BatchSubmitTransaction([]*SignedTransaction{signedTxn})
	BatchSubmitTransaction(signedTxns []*SignedTransaction) (*api.BatchSubmitTransactionResponse, error)

	// SimulateTransaction Simulates a raw transaction without sending it to the blockchain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "aptos_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	//	simResponse, err := client.SimulateTransaction(rawTxn, sender)
	SimulateTransaction(rawTxn *RawTransaction, sender TransactionSigner, options ...any) ([]*api.UserTransaction, error)

	// SimulateTransactionMultiAgent simulates a transaction as fee payer or multi agent
	SimulateTransactionMultiAgent(rawTxn *RawTransactionWithData, sender TransactionSigner, options ...any) ([]*api.UserTransaction, error)

	// GetChainId Retrieves the ChainId of the network
	// Note this will be cached forever, or taken directly from the config
	GetChainId() (uint8, error)

	// BuildTransaction Builds a raw transaction from the payload and fetches any necessary information from on-chain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "aptos_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, err := client.BuildTransaction(sender.AccountAddress(), txnPayload)
	BuildTransaction(sender AccountAddress, payload TransactionPayload, options ...any) (*RawTransaction, error)

	// BuildTransactionMultiAgent Builds a raw transaction for MultiAgent or FeePayer from the payload and fetches any necessary information from on-chain
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "aptos_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	rawTxn, err := client.BuildTransactionMultiAgent(sender.AccountAddress(), txnPayload, FeePayer(AccountZero))
	BuildTransactionMultiAgent(sender AccountAddress, payload TransactionPayload, options ...any) (*RawTransactionWithData, error)

	// BuildSignAndSubmitTransaction Convenience function to do all three in one
	// for more configuration, please use them separately
	//
	//	sender := NewEd25519Account()
	//	txnPayload := TransactionPayload{
	//		Payload: &EntryFunction{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name: "aptos_account",
	//			},
	//			Function: "transfer",
	//			ArgTypes: []TypeTag{},
	//			Args: [][]byte{
	//				dest[:],
	//				amountBytes,
	//			},
	//		}
	//	}
	//	submitResponse, err := client.BuildSignAndSubmitTransaction(sender, txnPayload)
	BuildSignAndSubmitTransaction(sender TransactionSigner, payload TransactionPayload, options ...any) (*api.SubmitTransactionResponse, error)

	// View Runs a view function on chain returning a list of return values.
	//
	//	 address := AccountOne
	//		payload := &ViewPayload{
	//			Module: ModuleId{
	//				Address: AccountOne,
	//				Name:    "coin",
	//			},
	//			Function: "balance",
	//			ArgTypes: []TypeTag{AptosCoinTypeTag},
	//			Args:     [][]byte{address[:]},
	//		}
	//		vals, err := client.aptosClient.View(payload)
	//		balance := StrToU64(vals.(any[])[0].(string))
	View(payload *ViewPayload, ledgerVersion ...uint64) ([]any, error)

	// EstimateGasPrice Retrieves the gas estimate from the network.
	EstimateGasPrice() (EstimateGasInfo, error)

	// AccountAPTBalance retrieves the APT balance in the account
	AccountAPTBalance(address AccountAddress, ledgerVersion ...uint64) (uint64, error)

	// NodeAPIHealthCheck checks if the node is within durationSecs of the current time, if not provided the node default is used
	NodeAPIHealthCheck(durationSecs ...uint64) (api.HealthCheckResponse, error)
}

// AptosFaucetClient is an interface for all functionality on the Client that is Faucet related.  Its main implementation
// is [FaucetClient]
type AptosFaucetClient interface {
	// Fund Uses the faucet to fund an address, only applies to non-production networks
	Fund(address AccountAddress, amount uint64) error
}

// AptosIndexerClient is an interface for all functionality on the Client that is Indexer related.  Its main implementation
// is [IndexerClient]
type AptosIndexerClient interface {
	// QueryIndexer queries the indexer using GraphQL to fill the `query` struct with data.  See examples in the indexer client on how to make queries
	//
	//	var out []CoinBalance
	//	var q struct {
	//		Current_coin_balances []struct {
	//			CoinType     string `graphql:"coin_type"`
	//			Amount       uint64
	//			OwnerAddress string `graphql:"owner_address"`
	//		} `graphql:"current_coin_balances(where: {owner_address: {_eq: $address}})"`
	//	}
	//	variables := map[string]any{
	//		"address": address.StringLong(),
	//	}
	//	err := client.QueryIndexer(&q, variables)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	for _, coin := range q.Current_coin_balances {
	//		out = append(out, CoinBalance{
	//			CoinType: coin.CoinType,
	//			Amount:   coin.Amount,
	//	})
	//	}
	//
	//	return out, nil
	QueryIndexer(query any, variables map[string]any, options ...graphql.Option) error

	// GetProcessorStatus returns the ledger version up to which the processor has processed
	GetProcessorStatus(processorName string) (uint64, error)

	// GetCoinBalances gets the balances of all coins associated with a given address
	GetCoinBalances(address AccountAddress) ([]CoinBalance, error)
}

// Client is a facade over the multiple types of underlying clients, as the user doesn't actually care where the data
// comes from.  It will be then handled underneath
//
// To create a new client, please use [NewClient].  An example below for Devnet:
//
//	client := NewClient(DevnetConfig)
//
// Implements AptosClient
type Client struct {
	nodeClient    *NodeClient
	faucetClient  *FaucetClient
	indexerClient *IndexerClient
}

// NewClient Creates a new client with a specific network config that can be extended in the future
func NewClient(config NetworkConfig, options ...any) (*Client, error) {
	var httpClient *http.Client
	for i, arg := range options {
		switch value := arg.(type) {
		case *http.Client:
			if httpClient != nil {
				return nil, errors.New("NewClient only accepts one http.Client")
			}
			httpClient = value
		default:
			return nil, fmt.Errorf("NewClient arg %d bad type %T", i+1, arg)
		}
	}
	var err error
	var nodeClient *NodeClient
	if httpClient == nil {
		nodeClient, err = NewNodeClient(config.NodeUrl, config.ChainId)
	} else {
		nodeClient, err = NewNodeClientWithHttpClient(config.NodeUrl, config.ChainId, httpClient)
	}
	if err != nil {
		return nil, err
	}
	// Indexer may not be present
	var indexerClient *IndexerClient
	if config.IndexerUrl != "" {
		indexerClient = NewIndexerClient(nodeClient.client, config.IndexerUrl)
	}

	// Faucet may not be present
	var faucetClient *FaucetClient
	if config.FaucetUrl != "" {
		faucetClient, err = NewFaucetClient(nodeClient, config.FaucetUrl)
		if err != nil {
			return nil, err
		}
	}

	// Fetch the chain Id if it isn't in the config
	if config.ChainId == 0 {
		_, _ = nodeClient.GetChainId()
	}

	return &Client{
		nodeClient,
		faucetClient,
		indexerClient,
	}, nil
}

// SetTimeout adjusts the HTTP client timeout
//
//	client.SetTimeout(5 * time.Millisecond)
func (client *Client) SetTimeout(timeout time.Duration) {
	client.nodeClient.SetTimeout(timeout)
}

// SetHeader sets the header for all future requests
//
//	client.SetHeader("Authorization", "Bearer abcde")
func (client *Client) SetHeader(key string, value string) {
	client.nodeClient.SetHeader(key, value)
}

// RemoveHeader removes the header from being automatically set all future requests.
//
//	client.RemoveHeader("Authorization")
func (client *Client) RemoveHeader(key string) {
	client.nodeClient.RemoveHeader(key)
}

// Info Retrieves the node info about the network and it's current state
func (client *Client) Info() (NodeInfo, error) {
	return client.nodeClient.Info()
}

// Account Retrieves information about the account such as [SequenceNumber] and [crypto.AuthenticationKey]
func (client *Client) Account(address AccountAddress, ledgerVersion ...uint64) (AccountInfo, error) {
	return client.nodeClient.Account(address, ledgerVersion...)
}

// AccountResource Retrieves a single resource given its struct name.
//
//	address := AccountOne
//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore")
//
// Can also fetch at a specific ledger version
//
//	address := AccountOne
//	dataMap, _ := client.AccountResource(address, "0x1::coin::CoinStore", 1)
func (client *Client) AccountResource(address AccountAddress, resourceType string, ledgerVersion ...uint64) (map[string]any, error) {
	return client.nodeClient.AccountResource(address, resourceType, ledgerVersion...)
}

// AccountResources fetches resources for an account into a JSON-like map[string]any in AccountResourceInfo.Data
// For fetching raw Move structs as BCS, See #AccountResourcesBCS
//
//	address := AccountOne
//	dataMap, _ := client.AccountResources(address)
//
// Can also fetch at a specific ledger version
//
//	address := AccountOne
//	dataMap, _ := client.AccountResource(address, 1)
func (client *Client) AccountResources(address AccountAddress, ledgerVersion ...uint64) ([]AccountResourceInfo, error) {
	return client.nodeClient.AccountResources(address, ledgerVersion...)
}

// AccountResourcesBCS fetches account resources as raw Move struct BCS blobs in AccountResourceRecord.Data []byte
func (client *Client) AccountResourcesBCS(address AccountAddress, ledgerVersion ...uint64) ([]AccountResourceRecord, error) {
	return client.nodeClient.AccountResourcesBCS(address, ledgerVersion...)
}

// BlockByHeight fetches a block by height
//
//	block, _ := client.BlockByHeight(1, false)
//
// Can also fetch with transactions
//
//	block, _ := client.BlockByHeight(1, true)
func (client *Client) BlockByHeight(blockHeight uint64, withTransactions bool) (*api.Block, error) {
	return client.nodeClient.BlockByHeight(blockHeight, withTransactions)
}

// BlockByVersion fetches a block by ledger version
//
//	block, _ := client.BlockByVersion(123, false)
//
// Can also fetch with transactions
//
//	block, _ := client.BlockByVersion(123, true)
func (client *Client) BlockByVersion(ledgerVersion uint64, withTransactions bool) (*api.Block, error) {
	return client.nodeClient.BlockByVersion(ledgerVersion, withTransactions)
}

// TransactionByHash gets info on a transaction
// The transaction may be pending or recently committed.
//
//	data, err := client.TransactionByHash("0xabcd")
//	if err != nil {
//		if httpErr, ok := err.(aptos.HttpError) {
//			if httpErr.StatusCode == 404 {
//				// if we're sure this has been submitted, assume it is still pending elsewhere in the mempool
//			}
//		}
//	} else {
//		if data["type"] == "pending_transaction" {
//			// known to local mempool, but not committed yet
//		}
//	}
func (client *Client) TransactionByHash(txnHash string) (*api.Transaction, error) {
	return client.nodeClient.TransactionByHash(txnHash)
}

// WaitTransactionByHash waits for a transaction to complete and returns it's data when finished.
func (client *Client) WaitTransactionByHash(txnHash string) (*api.Transaction, error) {
	return client.nodeClient.WaitTransactionByHash(txnHash)
}

// TransactionByVersion gets info on a transaction from its LedgerVersion.  It must have been
// committed to have a ledger version
//
//	data, err := client.TransactionByVersion("0xabcd")
//	if err != nil {
//		if httpErr, ok := err.(aptos.HttpError) {
//			if httpErr.StatusCode == 404 {
//				// if we're sure this has been submitted, the full node might not be caught up to this version yet
//			}
//		}
//	}
func (client *Client) TransactionByVersion(version uint64) (*api.CommittedTransaction, error) {
	return client.nodeClient.TransactionByVersion(version)
}

// PollForTransaction Waits up to 10 seconds for as single transaction to be done, polling at 10Hz
func (client *Client) PollForTransaction(hash string, options ...any) (*api.UserTransaction, error) {
	return client.nodeClient.PollForTransaction(hash, options...)
}

// PollForTransactions Waits up to 10 seconds for transactions to be done, polling at 10Hz
// Accepts options PollPeriod and PollTimeout which should wrap time.Duration values.
//
//	hashes := []string{"0x1234", "0x4567"}
//	err := client.PollForTransactions(hashes)
//
// Can additionally configure different options
//
//	hashes := []string{"0x1234", "0x4567"}
//	err := client.PollForTransactions(hashes, PollPeriod(500 * time.Milliseconds), PollTimeout(5 * time.Seconds))
func (client *Client) PollForTransactions(txnHashes []string, options ...any) error {
	return client.nodeClient.PollForTransactions(txnHashes, options...)
}

// WaitForTransaction Do a long-GET for one transaction and wait for it to complete
//
//	data, err := client.WaitForTransaction("0x1234")
func (client *Client) WaitForTransaction(txnHash string, options ...any) (*api.UserTransaction, error) {
	return client.nodeClient.WaitForTransaction(txnHash, options...)
}

// Transactions Get recent transactions.
// Start is a version number. Nil for most recent transactions.
// Limit is a number of transactions to return. 'about a hundred' by default.
//
//	client.Transactions(0, 2)   // Returns 2 transactions
//	client.Transactions(1, 100) // Returns 100 transactions
func (client *Client) Transactions(start *uint64, limit *uint64) ([]*api.CommittedTransaction, error) {
	return client.nodeClient.Transactions(start, limit)
}

// AccountTransactions Get transactions associated with an account.
// Start is a version number. Nil for most recent transactions.
// Limit is a number of transactions to return. 'about a hundred' by default.
//
//	client.AccountTransactions(AccountOne, 0, 2)   // Returns 2 transactions for 0x1
//	client.AccountTransactions(AccountOne, 1, 100) // Returns 100 transactions for 0x1
func (client *Client) AccountTransactions(address AccountAddress, start *uint64, limit *uint64) ([]*api.CommittedTransaction, error) {
	return client.nodeClient.AccountTransactions(address, start, limit)
}

// EventsByHandle Get events by handle and field name for an account.
// Start is a sequence number. Nil for most recent events.
// Limit is a number of events to return, 100 by default.
//
//	client.EventsByHandle(AccountOne, "0x2", "transfer", 0, 2)   // Returns 2 events
//	client.EventsByHandle(AccountOne, "0x2", "transfer", 1, 100) // Returns 100 events
func (client *Client) EventsByHandle(account AccountAddress, eventHandle string, fieldName string, start *uint64, limit *uint64) ([]*api.Event, error) {
	return client.nodeClient.EventsByHandle(account, eventHandle, fieldName, start, limit)
}

// EventsByCreationNumber Get events by creation number for an account.
// Start is a sequence number. Nil for most recent events.
// Limit is a number of events to return, 100 by default.
//
//	client.EventsByCreationNumber(AccountOne, "123", nil, 2)   // Returns 2 events
//	client.EventsByCreationNumber(AccountOne, "123", 1, 100) // Returns 100 events
func (client *Client) EventsByCreationNumber(account AccountAddress, creationNumber string, start *uint64, limit *uint64) ([]*api.Event, error) {
	return client.nodeClient.EventsByCreationNumber(account, creationNumber, start, limit)
}

// SubmitTransaction Submits an already signed transaction to the blockchain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "aptos_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
//	signedTxn, _ := sender.SignTransaction(rawTxn)
//	submitResponse, err := client.SubmitTransaction(signedTxn)
func (client *Client) SubmitTransaction(signedTransaction *SignedTransaction) (*api.SubmitTransactionResponse, error) {
	return client.nodeClient.SubmitTransaction(signedTransaction)
}

// BatchSubmitTransaction submits a collection of signed transactions to the network in a single request
//
// It will return the responses in the same order as the input transactions that failed.  If the response is empty, then
// all transactions succeeded.
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "aptos_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
//	signedTxn, _ := sender.SignTransaction(rawTxn)
//	submitResponse, err := client.BatchSubmitTransaction([]*SignedTransaction{signedTxn})
func (client *Client) BatchSubmitTransaction(signedTxns []*SignedTransaction) (*api.BatchSubmitTransactionResponse, error) {
	return client.nodeClient.BatchSubmitTransaction(signedTxns)
}

// SimulateTransaction Simulates a raw transaction without sending it to the blockchain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "aptos_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, _ := client.BuildTransaction(sender.AccountAddress(), txnPayload)
//	simResponse, err := client.SimulateTransaction(rawTxn, sender)
func (client *Client) SimulateTransaction(rawTxn *RawTransaction, sender TransactionSigner, options ...any) ([]*api.UserTransaction, error) {
	return client.nodeClient.SimulateTransaction(rawTxn, sender, options...)
}

// SimulateTransactionMultiAgent simulates a transaction as fee payer or multi agent
func (client *Client) SimulateTransactionMultiAgent(rawTxn *RawTransactionWithData, sender TransactionSigner, options ...any) ([]*api.UserTransaction, error) {
	return client.nodeClient.SimulateTransactionMultiAgent(rawTxn, sender, options...)
}

// GetChainId Retrieves the ChainId of the network
// Note this will be cached forever, or taken directly from the config
func (client *Client) GetChainId() (uint8, error) {
	return client.nodeClient.GetChainId()
}

// Fund Uses the faucet to fund an address, only applies to non-production networks
func (client *Client) Fund(address AccountAddress, amount uint64) error {
	return client.faucetClient.Fund(address, amount)
}

// BuildTransaction Builds a raw transaction from the payload and fetches any necessary information from on-chain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "aptos_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, err := client.BuildTransaction(sender.AccountAddress(), txnPayload)
func (client *Client) BuildTransaction(sender AccountAddress, payload TransactionPayload, options ...any) (*RawTransaction, error) {
	return client.nodeClient.BuildTransaction(sender, payload, options...)
}

// BuildTransactionMultiAgent Builds a raw transaction for MultiAgent or FeePayer from the payload and fetches any necessary information from on-chain
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "aptos_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	rawTxn, err := client.BuildTransactionMultiAgent(sender.AccountAddress(), txnPayload, FeePayer(AccountZero))
func (client *Client) BuildTransactionMultiAgent(sender AccountAddress, payload TransactionPayload, options ...any) (*RawTransactionWithData, error) {
	return client.nodeClient.BuildTransactionMultiAgent(sender, payload, options...)
}

// BuildSignAndSubmitTransaction Convenience function to do all three in one
// for more configuration, please use them separately
//
//	sender := NewEd25519Account()
//	txnPayload := TransactionPayload{
//		Payload: &EntryFunction{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name: "aptos_account",
//			},
//			Function: "transfer",
//			ArgTypes: []TypeTag{},
//			Args: [][]byte{
//				dest[:],
//				amountBytes,
//			},
//		}
//	}
//	submitResponse, err := client.BuildSignAndSubmitTransaction(sender, txnPayload)
func (client *Client) BuildSignAndSubmitTransaction(sender TransactionSigner, payload TransactionPayload, options ...any) (*api.SubmitTransactionResponse, error) {
	return client.nodeClient.BuildSignAndSubmitTransaction(sender, payload, options...)
}

// View Runs a view function on chain returning a list of return values.
//
//	 address := AccountOne
//		payload := &ViewPayload{
//			Module: ModuleId{
//				Address: AccountOne,
//				Name:    "coin",
//			},
//			Function: "balance",
//			ArgTypes: []TypeTag{AptosCoinTypeTag},
//			Args:     [][]byte{address[:]},
//		}
//		vals, err := client.aptosClient.View(payload)
//		balance := StrToU64(vals.(any[])[0].(string))
func (client *Client) View(payload *ViewPayload, ledgerVersion ...uint64) ([]any, error) {
	return client.nodeClient.View(payload, ledgerVersion...)
}

// EstimateGasPrice Retrieves the gas estimate from the network.
func (client *Client) EstimateGasPrice() (EstimateGasInfo, error) {
	return client.nodeClient.EstimateGasPrice()
}

// AccountAPTBalance retrieves the APT balance in the account
func (client *Client) AccountAPTBalance(address AccountAddress, ledgerVersion ...uint64) (uint64, error) {
	return client.nodeClient.AccountAPTBalance(address, ledgerVersion...)
}

// QueryIndexer queries the indexer using GraphQL to fill the `query` struct with data.  See examples in the indexer client on how to make queries
//
//	var out []CoinBalance
//	var q struct {
//		Current_coin_balances []struct {
//			CoinType     string `graphql:"coin_type"`
//			Amount       uint64
//			OwnerAddress string `graphql:"owner_address"`
//		} `graphql:"current_coin_balances(where: {owner_address: {_eq: $address}})"`
//	}
//	variables := map[string]any{
//		"address": address.StringLong(),
//	}
//	err := client.QueryIndexer(&q, variables)
//	if err != nil {
//		return nil, err
//	}
//
//	for _, coin := range q.Current_coin_balances {
//		out = append(out, CoinBalance{
//			CoinType: coin.CoinType,
//			Amount:   coin.Amount,
//	})
//	}
//
//	return out, nil
func (client *Client) QueryIndexer(query any, variables map[string]any, options ...graphql.Option) error {
	return client.indexerClient.Query(query, variables, options...)
}

// GetProcessorStatus returns the ledger version up to which the processor has processed
func (client *Client) GetProcessorStatus(processorName string) (uint64, error) {
	return client.indexerClient.GetProcessorStatus(processorName)
}

// GetCoinBalances gets the balances of all coins associated with a given address
func (client *Client) GetCoinBalances(address AccountAddress) ([]CoinBalance, error) {
	return client.indexerClient.GetCoinBalances(address)
}

// NodeAPIHealthCheck checks if the node is within durationSecs of the current time, if not provided the node default is used
func (client *Client) NodeAPIHealthCheck(durationSecs ...uint64) (api.HealthCheckResponse, error) {
	return client.nodeClient.NodeAPIHealthCheck(durationSecs...)
}

func (client *Client) AccountModule(address AccountAddress, moduleName string, ledgerVersion ...uint64) (*api.MoveBytecode, error) {
	return client.nodeClient.AccountModule(address, moduleName, ledgerVersion...)
}

func (client *Client) EntryFunctionWithArgs(address AccountAddress, moduleName string, functionName string, typeArgs []any, args []any, options ...any) (*EntryFunction, error) {
	return client.nodeClient.EntryFunctionWithArgs(address, moduleName, functionName, typeArgs, args, options...)
}
