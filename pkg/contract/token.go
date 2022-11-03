// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"burnAmount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"upgrade\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"upgradeAgent\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"upgradeMaster\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getUpgradeState\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"canUpgrade\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalUpgraded\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"agent\",\"type\":\"address\"}],\"name\":\"setUpgradeAgent\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"remaining\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isToken\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"BURN_ADDRESS\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"master\",\"type\":\"address\"}],\"name\":\"setUpgradeMaster\",\"outputs\":[],\"payable\":false,\"type\":\"function\"},{\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_symbol\",\"type\":\"string\"},{\"name\":\"_totalSupply\",\"type\":\"uint256\"},{\"name\":\"_decimals\",\"type\":\"uint256\"}],\"payable\":false,\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Upgrade\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"agent\",\"type\":\"address\"}],\"name\":\"UpgradeAgentSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"burner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"burnedAmount\",\"type\":\"uint256\"}],\"name\":\"Burned\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"}]",
	Bin: "0x6060604052346100005760405162000fb338038062000fb383398101604090815281516020830151918301516060840151608085015192949384019391909101915b845b60038054600160a060020a031916600160a060020a0383161790555b508360069080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106100ac57805160ff19168380011785556100d9565b828001600101855582156100d9579182015b828111156100d95782518255916020019190600101906100be565b5b506100fa9291505b808211156100f657600081556001016100e2565b5090565b50508260079080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061014857805160ff1916838001178555610175565b82800160010185558215610175579182015b8281111561017557825182559160200191906001019061015a565b5b506101969291505b808211156100f657600081556001016100e2565b5090565b505060008281556008829055600160a060020a03861681526001602052604090208290555b50505050505b610de280620001d16000396000f300606060405236156100f65763ffffffff60e060020a60003504166306fdde0381146100fb578063095ea7b31461018857806318160ddd146101b857806323b872dd146101d7578063313ce5671461020d57806342966c681461022c57806345977d031461023e5780635de4ccb014610250578063600440cb1461027957806370a08231146102a25780638444b391146102cd57806395d89b41146102fb5780639738968c14610388578063a9059cbb146103a9578063c752ff62146103d9578063d7e7088a146103f8578063dd62ed3e14610413578063eefa597b14610444578063fccc281314610465578063ffeb7d751461048e575b610000565b34610000576101086104a9565b60408051602080825283518183015283519192839290830191850190808383821561014e575b80518252602083111561014e57601f19909201916020918201910161012e565b505050905090810190601f16801561017a5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34610000576101a4600160a060020a0360043516602435610537565b604080519115158252519081900360200190f35b34610000576101c56105dd565b60408051918252519081900360200190f35b34610000576101a4600160a060020a03600435811690602435166044356105e3565b604080519115158252519081900360200190f35b34610000576101c56106e6565b60408051918252519081900360200190f35b346100005761023c6004356106ec565b005b346100005761023c6004356107c4565b005b346100005761025d610927565b60408051600160a060020a039092168252519081900360200190f35b346100005761025d610936565b60408051600160a060020a039092168252519081900360200190f35b34610000576101c5600160a060020a0360043516610945565b60408051918252519081900360200190f35b34610000576102da610964565b6040518082600481116100005760ff16815260200191505060405180910390f35b34610000576101086109b1565b60408051602080825283518183015283519192839290830191850190808383821561014e575b80518252602083111561014e57601f19909201916020918201910161012e565b505050905090810190601f16801561017a5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34610000576101a4610a3f565b604080519115158252519081900360200190f35b34610000576101a4600160a060020a0360043516602435610a45565b604080519115158252519081900360200190f35b34610000576101c5610b0e565b60408051918252519081900360200190f35b346100005761023c600160a060020a0360043516610b14565b005b34610000576101c5600160a060020a0360043581169060243516610cd2565b60408051918252519081900360200190f35b34610000576101a4610cff565b604080519115158252519081900360200190f35b346100005761025d610d04565b60408051600160a060020a039092168252519081900360200190f35b346100005761023c600160a060020a0360043516610d09565b005b6006805460408051602060026001851615610100026000190190941693909304601f8101849004840282018401909252818152929183018282801561052f5780601f106105045761010080835404028352916020019161052f565b820191906000526020600020905b81548152906001019060200180831161051257829003601f168201915b505050505081565b6000811580159061056c5750600160a060020a0333811660009081526002602090815260408083209387168352929052205415155b1561057657610000565b600160a060020a03338116600081815260026020908152604080832094881680845294825291829020869055815186815291517f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259281900390910190a35060015b92915050565b60005481565b600160a060020a0380841660009081526002602090815260408083203385168452825280832054938616835260019091528120549091906106249084610d65565b600160a060020a0380861660009081526001602052604080822093909355908716815220546106539084610d8d565b600160a060020a0386166000908152600160205260409020556106768184610d8d565b600160a060020a038087166000818152600260209081526040808320338616845282529182902094909455805187815290519288169391927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef929181900390910190a3600191505b509392505050565b60085481565b33600160a060020a0381166000908152600160205260409020546107109083610d8d565b600160a060020a038216600090815260016020526040812091909155546107379083610d8d565b60005560408051600160a060020a03831681526020810184905281517f696de425f79f4a40bc6d2122ca50507f0efbeabbff86a84871b7196ab8ea8df7929181900390910190a1604080518381529051600091600160a060020a038416917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a35b5050565b60006107ce610964565b9050600381600481116100005714806107ef57506004816004811161000057145b15156107fa57610000565b81151561080657610000565b600160a060020a0333166000908152600160205260409020546108299083610d8d565b600160a060020a033316600090815260016020526040812091909155546108509083610d8d565b6000556005546108609083610d65565b60055560048054604080517f753e88e5000000000000000000000000000000000000000000000000000000008152600160a060020a0333811694820194909452602481018690529051929091169163753e88e59160448082019260009290919082900301818387803b156100005760325a03f115610000575050600454604080518581529051600160a060020a03928316935033909216917f7e5c344a8141a805725cb476f76c6953b842222b967edd1f78ddb6e8b3f397ac9181900360200190a35b5050565b600454600160a060020a031681565b600354600160a060020a031681565b600160a060020a0381166000908152600160205260409020545b919050565b600061096e610a3f565b151561097c575060016109ab565b600454600160a060020a03161515610996575060026109ab565b60055415156109a7575060036109ab565b5060045b5b5b5b90565b6007805460408051602060026001851615610100026000190190941693909304601f8101849004840282018401909252818152929183018282801561052f5780601f106105045761010080835404028352916020019161052f565b820191906000526020600020905b81548152906001019060200180831161051257829003601f168201915b505050505081565b60015b90565b600060406044361015610a5757610000565b600160a060020a033316600090815260016020526040902054610a7a9084610d8d565b600160a060020a033381166000908152600160205260408082209390935590861681522054610aa99084610d65565b600160a060020a038086166000818152600160209081526040918290209490945580518781529051919333909316927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef92918290030190a3600191505b5b5092915050565b60055481565b610b1c610a3f565b1515610b2757610000565b600160a060020a0381161515610b3c57610000565b60035433600160a060020a03908116911614610b5757610000565b6004610b61610964565b60048111610000571415610b7457610000565b6004805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0383811691909117808355604080516000602091820181905282517f61d3d7a6000000000000000000000000000000000000000000000000000000008152925193909416946361d3d7a69483820194929383900390910190829087803b156100005760325a03f1156100005750506040515115159050610c1557610000565b600054600460009054906101000a9004600160a060020a0316600160a060020a0316634b2ba0dd6000604051602001526040518163ffffffff1660e060020a028152600401809050602060405180830381600087803b156100005760325a03f11561000057505060405151919091149050610c8f57610000565b60045460408051600160a060020a039092168252517f7845d5aa74cc410e35571258d954f23b82276e160fe8c188fa80566580f279cc9181900360200190a15b50565b600160a060020a038083166000908152600260209081526040808320938516835292905220545b92915050565b600181565b600081565b600160a060020a0381161515610d1e57610000565b60035433600160a060020a03908116911614610d3957610000565b6003805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0383161790555b50565b6000828201610d82848210801590610d7d5750838210155b610da6565b8091505b5092915050565b6000610d9b83831115610da6565b508082035b92915050565b801515610ccf57610000565b5b505600a165627a7a723058207f245bfa3c64bf13794a0e7c04d38c15c9b39b6cca49410784290397af00ed690029",
}

// TokenABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenMetaData.ABI instead.
var TokenABI = TokenMetaData.ABI

// TokenBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TokenMetaData.Bin instead.
var TokenBin = TokenMetaData.Bin

// DeployToken deploys a new Ethereum contract, binding an instance of Token to it.
func DeployToken(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address, _name string, _symbol string, _totalSupply *big.Int, _decimals *big.Int) (common.Address, *types.Transaction, *Token, error) {
	parsed, err := TokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TokenBin), backend, _owner, _name, _symbol, _totalSupply, _decimals)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Token{TokenCaller: TokenCaller{contract: contract}, TokenTransactor: TokenTransactor{contract: contract}, TokenFilterer: TokenFilterer{contract: contract}}, nil
}

// Token is an auto generated Go binding around an Ethereum contract.
type Token struct {
	TokenCaller     // Read-only binding to the contract
	TokenTransactor // Write-only binding to the contract
	TokenFilterer   // Log filterer for contract events
}

// TokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenSession struct {
	Contract     *Token            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenCallerSession struct {
	Contract *TokenCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenTransactorSession struct {
	Contract     *TokenTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenRaw struct {
	Contract *Token // Generic contract binding to access the raw methods on
}

// TokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenCallerRaw struct {
	Contract *TokenCaller // Generic read-only contract binding to access the raw methods on
}

// TokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenTransactorRaw struct {
	Contract *TokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewToken creates a new instance of Token, bound to a specific deployed contract.
func NewToken(address common.Address, backend bind.ContractBackend) (*Token, error) {
	contract, err := bindToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Token{TokenCaller: TokenCaller{contract: contract}, TokenTransactor: TokenTransactor{contract: contract}, TokenFilterer: TokenFilterer{contract: contract}}, nil
}

// NewTokenCaller creates a new read-only instance of Token, bound to a specific deployed contract.
func NewTokenCaller(address common.Address, caller bind.ContractCaller) (*TokenCaller, error) {
	contract, err := bindToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCaller{contract: contract}, nil
}

// NewTokenTransactor creates a new write-only instance of Token, bound to a specific deployed contract.
func NewTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenTransactor, error) {
	contract, err := bindToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenTransactor{contract: contract}, nil
}

// NewTokenFilterer creates a new log filterer instance of Token, bound to a specific deployed contract.
func NewTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenFilterer, error) {
	contract, err := bindToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenFilterer{contract: contract}, nil
}

// bindToken binds a generic wrapper to an already deployed contract.
func bindToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.TokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.TokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Token *TokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Token.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Token *TokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Token.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Token *TokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Token.Contract.contract.Transact(opts, method, params...)
}

// BURNADDRESS is a free data retrieval call binding the contract method 0xfccc2813.
//
// Solidity: function BURN_ADDRESS() returns(address)
func (_Token *TokenCaller) BURNADDRESS(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "BURN_ADDRESS")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BURNADDRESS is a free data retrieval call binding the contract method 0xfccc2813.
//
// Solidity: function BURN_ADDRESS() returns(address)
func (_Token *TokenSession) BURNADDRESS() (common.Address, error) {
	return _Token.Contract.BURNADDRESS(&_Token.CallOpts)
}

// BURNADDRESS is a free data retrieval call binding the contract method 0xfccc2813.
//
// Solidity: function BURN_ADDRESS() returns(address)
func (_Token *TokenCallerSession) BURNADDRESS() (common.Address, error) {
	return _Token.Contract.BURNADDRESS(&_Token.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) returns(uint256 remaining)
func (_Token *TokenCaller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) returns(uint256 remaining)
func (_Token *TokenSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _Token.Contract.Allowance(&_Token.CallOpts, _owner, _spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) returns(uint256 remaining)
func (_Token *TokenCallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _Token.Contract.Allowance(&_Token.CallOpts, _owner, _spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) returns(uint256 balance)
func (_Token *TokenCaller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) returns(uint256 balance)
func (_Token *TokenSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _Token.Contract.BalanceOf(&_Token.CallOpts, _owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) returns(uint256 balance)
func (_Token *TokenCallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _Token.Contract.BalanceOf(&_Token.CallOpts, _owner)
}

// CanUpgrade is a free data retrieval call binding the contract method 0x9738968c.
//
// Solidity: function canUpgrade() returns(bool)
func (_Token *TokenCaller) CanUpgrade(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "canUpgrade")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CanUpgrade is a free data retrieval call binding the contract method 0x9738968c.
//
// Solidity: function canUpgrade() returns(bool)
func (_Token *TokenSession) CanUpgrade() (bool, error) {
	return _Token.Contract.CanUpgrade(&_Token.CallOpts)
}

// CanUpgrade is a free data retrieval call binding the contract method 0x9738968c.
//
// Solidity: function canUpgrade() returns(bool)
func (_Token *TokenCallerSession) CanUpgrade() (bool, error) {
	return _Token.Contract.CanUpgrade(&_Token.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint256)
func (_Token *TokenCaller) Decimals(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint256)
func (_Token *TokenSession) Decimals() (*big.Int, error) {
	return _Token.Contract.Decimals(&_Token.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint256)
func (_Token *TokenCallerSession) Decimals() (*big.Int, error) {
	return _Token.Contract.Decimals(&_Token.CallOpts)
}

// GetUpgradeState is a free data retrieval call binding the contract method 0x8444b391.
//
// Solidity: function getUpgradeState() returns(uint8)
func (_Token *TokenCaller) GetUpgradeState(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "getUpgradeState")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetUpgradeState is a free data retrieval call binding the contract method 0x8444b391.
//
// Solidity: function getUpgradeState() returns(uint8)
func (_Token *TokenSession) GetUpgradeState() (uint8, error) {
	return _Token.Contract.GetUpgradeState(&_Token.CallOpts)
}

// GetUpgradeState is a free data retrieval call binding the contract method 0x8444b391.
//
// Solidity: function getUpgradeState() returns(uint8)
func (_Token *TokenCallerSession) GetUpgradeState() (uint8, error) {
	return _Token.Contract.GetUpgradeState(&_Token.CallOpts)
}

// IsToken is a free data retrieval call binding the contract method 0xeefa597b.
//
// Solidity: function isToken() returns(bool)
func (_Token *TokenCaller) IsToken(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "isToken")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsToken is a free data retrieval call binding the contract method 0xeefa597b.
//
// Solidity: function isToken() returns(bool)
func (_Token *TokenSession) IsToken() (bool, error) {
	return _Token.Contract.IsToken(&_Token.CallOpts)
}

// IsToken is a free data retrieval call binding the contract method 0xeefa597b.
//
// Solidity: function isToken() returns(bool)
func (_Token *TokenCallerSession) IsToken() (bool, error) {
	return _Token.Contract.IsToken(&_Token.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (_Token *TokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (_Token *TokenSession) Name() (string, error) {
	return _Token.Contract.Name(&_Token.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (_Token *TokenCallerSession) Name() (string, error) {
	return _Token.Contract.Name(&_Token.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (_Token *TokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (_Token *TokenSession) Symbol() (string, error) {
	return _Token.Contract.Symbol(&_Token.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (_Token *TokenCallerSession) Symbol() (string, error) {
	return _Token.Contract.Symbol(&_Token.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() returns(uint256)
func (_Token *TokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() returns(uint256)
func (_Token *TokenSession) TotalSupply() (*big.Int, error) {
	return _Token.Contract.TotalSupply(&_Token.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() returns(uint256)
func (_Token *TokenCallerSession) TotalSupply() (*big.Int, error) {
	return _Token.Contract.TotalSupply(&_Token.CallOpts)
}

// TotalUpgraded is a free data retrieval call binding the contract method 0xc752ff62.
//
// Solidity: function totalUpgraded() returns(uint256)
func (_Token *TokenCaller) TotalUpgraded(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "totalUpgraded")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalUpgraded is a free data retrieval call binding the contract method 0xc752ff62.
//
// Solidity: function totalUpgraded() returns(uint256)
func (_Token *TokenSession) TotalUpgraded() (*big.Int, error) {
	return _Token.Contract.TotalUpgraded(&_Token.CallOpts)
}

// TotalUpgraded is a free data retrieval call binding the contract method 0xc752ff62.
//
// Solidity: function totalUpgraded() returns(uint256)
func (_Token *TokenCallerSession) TotalUpgraded() (*big.Int, error) {
	return _Token.Contract.TotalUpgraded(&_Token.CallOpts)
}

// UpgradeAgent is a free data retrieval call binding the contract method 0x5de4ccb0.
//
// Solidity: function upgradeAgent() returns(address)
func (_Token *TokenCaller) UpgradeAgent(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "upgradeAgent")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UpgradeAgent is a free data retrieval call binding the contract method 0x5de4ccb0.
//
// Solidity: function upgradeAgent() returns(address)
func (_Token *TokenSession) UpgradeAgent() (common.Address, error) {
	return _Token.Contract.UpgradeAgent(&_Token.CallOpts)
}

// UpgradeAgent is a free data retrieval call binding the contract method 0x5de4ccb0.
//
// Solidity: function upgradeAgent() returns(address)
func (_Token *TokenCallerSession) UpgradeAgent() (common.Address, error) {
	return _Token.Contract.UpgradeAgent(&_Token.CallOpts)
}

// UpgradeMaster is a free data retrieval call binding the contract method 0x600440cb.
//
// Solidity: function upgradeMaster() returns(address)
func (_Token *TokenCaller) UpgradeMaster(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Token.contract.Call(opts, &out, "upgradeMaster")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UpgradeMaster is a free data retrieval call binding the contract method 0x600440cb.
//
// Solidity: function upgradeMaster() returns(address)
func (_Token *TokenSession) UpgradeMaster() (common.Address, error) {
	return _Token.Contract.UpgradeMaster(&_Token.CallOpts)
}

// UpgradeMaster is a free data retrieval call binding the contract method 0x600440cb.
//
// Solidity: function upgradeMaster() returns(address)
func (_Token *TokenCallerSession) UpgradeMaster() (common.Address, error) {
	return _Token.Contract.UpgradeMaster(&_Token.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_Token *TokenTransactor) Approve(opts *bind.TransactOpts, _spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "approve", _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_Token *TokenSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Approve(&_Token.TransactOpts, _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_Token *TokenTransactorSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Approve(&_Token.TransactOpts, _spender, _value)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 burnAmount) returns()
func (_Token *TokenTransactor) Burn(opts *bind.TransactOpts, burnAmount *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "burn", burnAmount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 burnAmount) returns()
func (_Token *TokenSession) Burn(burnAmount *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Burn(&_Token.TransactOpts, burnAmount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 burnAmount) returns()
func (_Token *TokenTransactorSession) Burn(burnAmount *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Burn(&_Token.TransactOpts, burnAmount)
}

// SetUpgradeAgent is a paid mutator transaction binding the contract method 0xd7e7088a.
//
// Solidity: function setUpgradeAgent(address agent) returns()
func (_Token *TokenTransactor) SetUpgradeAgent(opts *bind.TransactOpts, agent common.Address) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "setUpgradeAgent", agent)
}

// SetUpgradeAgent is a paid mutator transaction binding the contract method 0xd7e7088a.
//
// Solidity: function setUpgradeAgent(address agent) returns()
func (_Token *TokenSession) SetUpgradeAgent(agent common.Address) (*types.Transaction, error) {
	return _Token.Contract.SetUpgradeAgent(&_Token.TransactOpts, agent)
}

// SetUpgradeAgent is a paid mutator transaction binding the contract method 0xd7e7088a.
//
// Solidity: function setUpgradeAgent(address agent) returns()
func (_Token *TokenTransactorSession) SetUpgradeAgent(agent common.Address) (*types.Transaction, error) {
	return _Token.Contract.SetUpgradeAgent(&_Token.TransactOpts, agent)
}

// SetUpgradeMaster is a paid mutator transaction binding the contract method 0xffeb7d75.
//
// Solidity: function setUpgradeMaster(address master) returns()
func (_Token *TokenTransactor) SetUpgradeMaster(opts *bind.TransactOpts, master common.Address) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "setUpgradeMaster", master)
}

// SetUpgradeMaster is a paid mutator transaction binding the contract method 0xffeb7d75.
//
// Solidity: function setUpgradeMaster(address master) returns()
func (_Token *TokenSession) SetUpgradeMaster(master common.Address) (*types.Transaction, error) {
	return _Token.Contract.SetUpgradeMaster(&_Token.TransactOpts, master)
}

// SetUpgradeMaster is a paid mutator transaction binding the contract method 0xffeb7d75.
//
// Solidity: function setUpgradeMaster(address master) returns()
func (_Token *TokenTransactorSession) SetUpgradeMaster(master common.Address) (*types.Transaction, error) {
	return _Token.Contract.SetUpgradeMaster(&_Token.TransactOpts, master)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_Token *TokenTransactor) Transfer(opts *bind.TransactOpts, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "transfer", _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_Token *TokenSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Transfer(&_Token.TransactOpts, _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_Token *TokenTransactorSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Transfer(&_Token.TransactOpts, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_Token *TokenTransactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "transferFrom", _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_Token *TokenSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.TransferFrom(&_Token.TransactOpts, _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_Token *TokenTransactorSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.TransferFrom(&_Token.TransactOpts, _from, _to, _value)
}

// Upgrade is a paid mutator transaction binding the contract method 0x45977d03.
//
// Solidity: function upgrade(uint256 value) returns()
func (_Token *TokenTransactor) Upgrade(opts *bind.TransactOpts, value *big.Int) (*types.Transaction, error) {
	return _Token.contract.Transact(opts, "upgrade", value)
}

// Upgrade is a paid mutator transaction binding the contract method 0x45977d03.
//
// Solidity: function upgrade(uint256 value) returns()
func (_Token *TokenSession) Upgrade(value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Upgrade(&_Token.TransactOpts, value)
}

// Upgrade is a paid mutator transaction binding the contract method 0x45977d03.
//
// Solidity: function upgrade(uint256 value) returns()
func (_Token *TokenTransactorSession) Upgrade(value *big.Int) (*types.Transaction, error) {
	return _Token.Contract.Upgrade(&_Token.TransactOpts, value)
}

// TokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Token contract.
type TokenApprovalIterator struct {
	Event *TokenApproval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenApproval represents a Approval event raised by the Token contract.
type TokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Token *TokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*TokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Token.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &TokenApprovalIterator{contract: _Token.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Token *TokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *TokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Token.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenApproval)
				if err := _Token.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Token *TokenFilterer) ParseApproval(log types.Log) (*TokenApproval, error) {
	event := new(TokenApproval)
	if err := _Token.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenBurnedIterator is returned from FilterBurned and is used to iterate over the raw logs and unpacked data for Burned events raised by the Token contract.
type TokenBurnedIterator struct {
	Event *TokenBurned // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenBurnedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenBurned)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenBurned)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenBurnedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenBurnedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenBurned represents a Burned event raised by the Token contract.
type TokenBurned struct {
	Burner       common.Address
	BurnedAmount *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBurned is a free log retrieval operation binding the contract event 0x696de425f79f4a40bc6d2122ca50507f0efbeabbff86a84871b7196ab8ea8df7.
//
// Solidity: event Burned(address burner, uint256 burnedAmount)
func (_Token *TokenFilterer) FilterBurned(opts *bind.FilterOpts) (*TokenBurnedIterator, error) {

	logs, sub, err := _Token.contract.FilterLogs(opts, "Burned")
	if err != nil {
		return nil, err
	}
	return &TokenBurnedIterator{contract: _Token.contract, event: "Burned", logs: logs, sub: sub}, nil
}

// WatchBurned is a free log subscription operation binding the contract event 0x696de425f79f4a40bc6d2122ca50507f0efbeabbff86a84871b7196ab8ea8df7.
//
// Solidity: event Burned(address burner, uint256 burnedAmount)
func (_Token *TokenFilterer) WatchBurned(opts *bind.WatchOpts, sink chan<- *TokenBurned) (event.Subscription, error) {

	logs, sub, err := _Token.contract.WatchLogs(opts, "Burned")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenBurned)
				if err := _Token.contract.UnpackLog(event, "Burned", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBurned is a log parse operation binding the contract event 0x696de425f79f4a40bc6d2122ca50507f0efbeabbff86a84871b7196ab8ea8df7.
//
// Solidity: event Burned(address burner, uint256 burnedAmount)
func (_Token *TokenFilterer) ParseBurned(log types.Log) (*TokenBurned, error) {
	event := new(TokenBurned)
	if err := _Token.contract.UnpackLog(event, "Burned", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Token contract.
type TokenTransferIterator struct {
	Event *TokenTransfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenTransfer represents a Transfer event raised by the Token contract.
type TokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Token *TokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*TokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Token.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TokenTransferIterator{contract: _Token.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Token *TokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *TokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Token.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenTransfer)
				if err := _Token.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Token *TokenFilterer) ParseTransfer(log types.Log) (*TokenTransfer, error) {
	event := new(TokenTransfer)
	if err := _Token.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenUpgradeIterator is returned from FilterUpgrade and is used to iterate over the raw logs and unpacked data for Upgrade events raised by the Token contract.
type TokenUpgradeIterator struct {
	Event *TokenUpgrade // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenUpgradeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenUpgrade)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenUpgrade)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenUpgradeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenUpgradeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenUpgrade represents a Upgrade event raised by the Token contract.
type TokenUpgrade struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterUpgrade is a free log retrieval operation binding the contract event 0x7e5c344a8141a805725cb476f76c6953b842222b967edd1f78ddb6e8b3f397ac.
//
// Solidity: event Upgrade(address indexed _from, address indexed _to, uint256 _value)
func (_Token *TokenFilterer) FilterUpgrade(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*TokenUpgradeIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _Token.contract.FilterLogs(opts, "Upgrade", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &TokenUpgradeIterator{contract: _Token.contract, event: "Upgrade", logs: logs, sub: sub}, nil
}

// WatchUpgrade is a free log subscription operation binding the contract event 0x7e5c344a8141a805725cb476f76c6953b842222b967edd1f78ddb6e8b3f397ac.
//
// Solidity: event Upgrade(address indexed _from, address indexed _to, uint256 _value)
func (_Token *TokenFilterer) WatchUpgrade(opts *bind.WatchOpts, sink chan<- *TokenUpgrade, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _Token.contract.WatchLogs(opts, "Upgrade", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenUpgrade)
				if err := _Token.contract.UnpackLog(event, "Upgrade", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgrade is a log parse operation binding the contract event 0x7e5c344a8141a805725cb476f76c6953b842222b967edd1f78ddb6e8b3f397ac.
//
// Solidity: event Upgrade(address indexed _from, address indexed _to, uint256 _value)
func (_Token *TokenFilterer) ParseUpgrade(log types.Log) (*TokenUpgrade, error) {
	event := new(TokenUpgrade)
	if err := _Token.contract.UnpackLog(event, "Upgrade", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenUpgradeAgentSetIterator is returned from FilterUpgradeAgentSet and is used to iterate over the raw logs and unpacked data for UpgradeAgentSet events raised by the Token contract.
type TokenUpgradeAgentSetIterator struct {
	Event *TokenUpgradeAgentSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenUpgradeAgentSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenUpgradeAgentSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenUpgradeAgentSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenUpgradeAgentSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenUpgradeAgentSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenUpgradeAgentSet represents a UpgradeAgentSet event raised by the Token contract.
type TokenUpgradeAgentSet struct {
	Agent common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterUpgradeAgentSet is a free log retrieval operation binding the contract event 0x7845d5aa74cc410e35571258d954f23b82276e160fe8c188fa80566580f279cc.
//
// Solidity: event UpgradeAgentSet(address agent)
func (_Token *TokenFilterer) FilterUpgradeAgentSet(opts *bind.FilterOpts) (*TokenUpgradeAgentSetIterator, error) {

	logs, sub, err := _Token.contract.FilterLogs(opts, "UpgradeAgentSet")
	if err != nil {
		return nil, err
	}
	return &TokenUpgradeAgentSetIterator{contract: _Token.contract, event: "UpgradeAgentSet", logs: logs, sub: sub}, nil
}

// WatchUpgradeAgentSet is a free log subscription operation binding the contract event 0x7845d5aa74cc410e35571258d954f23b82276e160fe8c188fa80566580f279cc.
//
// Solidity: event UpgradeAgentSet(address agent)
func (_Token *TokenFilterer) WatchUpgradeAgentSet(opts *bind.WatchOpts, sink chan<- *TokenUpgradeAgentSet) (event.Subscription, error) {

	logs, sub, err := _Token.contract.WatchLogs(opts, "UpgradeAgentSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenUpgradeAgentSet)
				if err := _Token.contract.UnpackLog(event, "UpgradeAgentSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgradeAgentSet is a log parse operation binding the contract event 0x7845d5aa74cc410e35571258d954f23b82276e160fe8c188fa80566580f279cc.
//
// Solidity: event UpgradeAgentSet(address agent)
func (_Token *TokenFilterer) ParseUpgradeAgentSet(log types.Log) (*TokenUpgradeAgentSet, error) {
	event := new(TokenUpgradeAgentSet)
	if err := _Token.contract.UnpackLog(event, "UpgradeAgentSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
