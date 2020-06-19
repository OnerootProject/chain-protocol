/*
 * Copyright (C) 2019 The oneroot-network Authors
 * This file is part of The onerootchain library.
 *
 * The onerootchain is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The onerootchain is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The onerootchain.  If not, see <http://www.gnu.org/licenses/>.
 */

package dex

import (
	"encoding/hex"
	"github.com/oneroot-network/onerootchain/common/buffer"
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/common/serialization"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/engine"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/facade"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"github.com/oneroot-network/onerootchain/core/types"
	"time"
)

//deposit asset to dex from `From` account and `To` account will receive balance in dex
func (p *DEXProtocol) Deposit(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	asset := new(ncom.AssetArgs)
	err := asset.Deserialize(reader)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(asset.From) {
		return nil, errors.ErrCtrInvalidateAuth
	}

	//do transfer asset to dex
	balance, cErr := DoDeposit(ref, asset)
	if cErr != errors.ErrOK {
		return nil, cErr
	}
	//emit log
	AddTransferEvtLog(ref, EvtLogDeposit, asset, balance)

	return nil, errors.ErrOK
}

// withdraw asset from `From` account in dex and transferred to `To` account wallet
func (p *DEXProtocol) Withdraw(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	asset := new(ncom.AssetArgs)
	err := asset.Deserialize(reader)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(asset.From) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	if !withdrawList(asset.From) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	//do transfer asset from dex
	balance, cErr := DoWithdraw(ref, asset)
	if cErr != errors.ErrOK {
		return nil, cErr
	}
	//emit log
	AddTransferEvtLog(ref, EvtLogWithdraw, asset, balance)
	return nil, errors.ErrOK
}

func withdrawList(from *types.Account) bool {
	return from.Equal(ncom.StakingCtrAccount) ||
		from.Equal(ncom.GovernanceCtrAccount) ||
		from.Equal(ncom.NameServiceCtrAccount) ||
		from.Equal(ncom.DexCtrAccount)
}

//delegateWithdraw for relay to help the user with from dex
func (p *DEXProtocol) DelegateWithdraw(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	withdrawArgs := new(facade.DWithdrawArgs)
	err := withdrawArgs.Deserialize(reader)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	if withdrawArgs.Fee > withdrawArgs.Amount {
		return nil, errors.ErrDexOverWithdrawFee
	}
	if !ref.CheckWitness(withdrawArgs.Relay) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	wRelayAddr := withdrawArgs.Relay.GetAddress()
	//verify relay authorized
	if !isRelay(ref, wRelayAddr) {
		return nil, errors.ErrDexUnAuthorized
	}

	hash, err := withdrawArgs.HashParams()
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	//verify withdraw done
	key := utils.GetDWithdrawKey(hash)
	withdrawn, err := ref.GetStateSet().GetBool(key)
	if err != nil {
		return nil, errors.ErrStore.SetMsg(err.Error())
	}
	if withdrawn.Value {
		return nil, errors.ErrDexWithdrawSubmitted
	}
	//verify user
	wFromAddr := withdrawArgs.From.GetAddress()
	if !VerifySigUser(wFromAddr, withdrawArgs.Sig) {
		return nil, errors.ErrDexVerifySigUserError
	}
	//verify signature of withdraw params
	if !VerifySig(hash, withdrawArgs.Sig) {
		return nil, errors.ErrDexVerifySigError
	}

	//do transfer asset from dex
	balance, cErr := DoDelegateWithdraw(ref, withdrawArgs)
	if cErr != errors.ErrOK {
		return nil, cErr
	}
	//write withdrawn to avoid double withdraw
	withdrawn, err = ref.GetStateSet().GetOrAddBool(key)
	if err != nil {
		ref.Logger().Warn("update delegate with state", "error", err)
		return nil, errors.ErrStore.SetMsg(err.Error())
	}
	withdrawn.Value = true
	//emit log
	AddDWithdrawEvtLog(ref, withdrawArgs, balance)
	return nil, errors.ErrOK
}

//PrepareWithdraw will record the applied withdraw asset amount and time
//return total applied asset amount(maybe > balance in dex)
func (p *DEXProtocol) PrepareWithdraw(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	withdrawArgs := new(ncom.AssetArgs)
	err := withdrawArgs.Deserialize(reader)
	if err != nil {
		return 0, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(withdrawArgs.From) {
		return 0, errors.ErrCtrInvalidateAuth
	}
	if !withdrawArgs.From.Equal(withdrawArgs.To) {
		//can only apply withdraw to self
		return 0, errors.ErrCtrInvalidArgs
	}
	//lock asset in apply withdraw state
	balance, cErr := DoApplyWithdraw(ref, withdrawArgs)
	if cErr != errors.ErrOK {
		return 0, cErr
	}
	//emit log
	AddTransferEvtLog(ref, EvtLogPrepareWithdraw, withdrawArgs, balance)
	return balance, errors.ErrOK
}

//CommitWithdraw will withdraw total applied asset to user's wallet from dex.
//Notice:total applied amount maybe >= balance in dex
func (p *DEXProtocol) CommitWithdraw(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	commitArgs := new(facade.CommitWithdrawArgs)
	err := commitArgs.Deserialize(reader)
	if err != nil {
		return 0, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(commitArgs.From) {
		return 0, errors.ErrCtrInvalidateAuth
	}

	cFromAddr := commitArgs.From.GetAddress()
	cAssetAddr := commitArgs.Asset.GetAddress()

	globalParams, cErr := GetGlobalParams(ref)
	if cErr != errors.ErrOK {
		ref.Logger().Error("get global params error", "error", cErr.String())
		return 0, cErr
	}
	// get applied withdraw state
	applyWithdrawKey := utils.GetPreparedWithdrawKey(cFromAddr, cAssetAddr)
	res, err := ref.GetStateSet().GetOrAddObject(applyWithdrawKey, &PrepareWithdrawState{})
	if err != nil {
		ref.Logger().Error("get apply withdraw state error", "from", commitArgs.From.String(), "asset", commitArgs.Asset.String(), "error", err)
		return 0, errors.ErrStore
	}

	ws := res.(*PrepareWithdrawState)
	if ref.GetContext().Timestamp < ws.Time+uint32(globalParams.WithdrawApplyWaitTime) {
		return 0, errors.ErrApplyWaitNotEnough
	}
	withdrawAmount := ws.Amount
	if withdrawAmount == 0 {
		return 0, errors.ErrWithdrawZero
	}

	//get balance in dex
	balanceKey := utils.GetBalanceKey(cFromAddr, cAssetAddr)
	balance, err := ref.GetStateSet().GetUint64(balanceKey)
	if err != nil {
		ref.Logger().Error("get balance state error", "from", commitArgs.From.String(), "asset", commitArgs.Asset.String(), "error", err)
		return 0, errors.ErrStore
	}
	if balance.Value < withdrawAmount {
		//
		withdrawAmount = balance.Value
	}
	//delete PrepareWithdrawState(total applied should be 0 after confirm withdraw)
	err = ref.GetStateSet().Delete(applyWithdrawKey)
	if err != nil {
		ref.Logger().Error("delete key error", "error", err)
		return 0, errors.ErrStore
	}

	assetArgs := &ncom.AssetArgs{
		Asset:  commitArgs.Asset,
		From:   commitArgs.From,
		To:     commitArgs.From,
		Amount: withdrawAmount,
	}
	remain, err2 := DoWithdraw(ref, assetArgs)
	if err2 != errors.ErrOK {
		return 0, err2
	}
	//emit log
	AddTransferEvtLog(ref, EvtLogCommitWithdraw, assetArgs, remain)
	return remain, errors.ErrOK
}
func (p *DEXProtocol) GetPrepareWithdrawState(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	r := buffer.NewBuffer(args)
	from := types.NewAccount()
	err := from.Deserialize(r)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	fromAddr := from.GetAddress()
	asset := types.NewAccount()
	err = asset.Deserialize(r)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	assetAddr := asset.GetAddress()
	state, err := ref.GetStateSet().GetObject(utils.GetPreparedWithdrawKey(fromAddr, assetAddr), &PrepareWithdrawState{})
	if err != nil {
		ref.Logger().Error("get apply withdraw state error", "from", from.String(), "asset", asset.String(), "error", err)
		return nil, errors.ErrStore.SetMsg(err.Error())
	}
	return state.(*PrepareWithdrawState), errors.ErrOK
}

// get the balance of account's asset
// returns balance or error
func (p *DEXProtocol) BalanceOf(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	r := buffer.NewBuffer(args)
	acc := types.NewAccount()
	err := acc.Deserialize(r)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	addr := acc.GetAddress()
	assetAcc := types.NewAccount()
	err = assetAcc.Deserialize(r)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	assetAddr := assetAcc.GetAddress()
	balance, err := ref.GetStateSet().GetUint64(utils.GetBalanceKey(addr, assetAddr))
	if err != nil {
		ref.Logger().Error("dex get balance error", "from", acc.String(), "asset", assetAcc.String(), "error", err)
		return nil, errors.ErrStore.SetMsg(err.Error())
	}
	return balance.Value, errors.ErrOK
}

func (p *DEXProtocol) Trade(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	b := time.Now().UnixNano()
	reader := buffer.NewBuffer(args)
	tradeArgs := facade.NewTradeArgs()
	err := tradeArgs.Deserialize(reader)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(tradeArgs.Relay.From) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	rFromAddr := tradeArgs.Relay.From.GetAddress()
	//verify relay authorized
	if !isRelay(ref, rFromAddr) {
		return nil, errors.ErrDexUnAuthorized
	}

	//do verify
	makerOrder, takerOrder, relay, cErr := verify(ref, tradeArgs)
	if cErr != errors.ErrOK {
		ref.Logger().Warn("verify error", "error", cErr.String())
		return nil, cErr
	}
	//do match
	clear, cErr := engine.MatchOrder(makerOrder, takerOrder, relay)
	if cErr != errors.ErrOK {
		ref.Logger().Warn("match error", "error", cErr.String())
		return nil, cErr
	}
	globalParams, cErr := GetGlobalParams(ref)
	if cErr != errors.ErrOK {
		return nil, cErr
	}
	countFee(ref, globalParams, makerOrder, takerOrder, clear)
	ref.Logger().Debug("clear info", "clear", clear)
	//do settlement
	cErr = settle(ref, makerOrder, takerOrder, relay, clear)
	if cErr != errors.ErrOK {
		ref.Logger().Warn("settle error", "error", cErr.String())
		return nil, cErr
	}
	AddTradeEvtLog(ref, clear, makerOrder, takerOrder)
	e := time.Now().UnixNano()
	ref.Logger().Debug("Dex Trade", "time", (e-b)/1e6)
	return nil, errors.ErrOK
}

//user can cancel the order by itself
func (p *DEXProtocol) CancelOrder(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	cancelArgs := new(facade.CancelOrderArgs)
	err := cancelArgs.Deserialize(reader)
	if err != nil {
		return false, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(cancelArgs.User) { //very important
		return false, errors.ErrCtrInvalidateAuth
	}
	id, err := cancelArgs.OrderId()
	if err != nil {
		ref.Logger().Error("get order id error", "error", err)
		return false, errors.ErrCtrInvalidArgs
	}
	key := utils.GetOrderIdKey(id)
	res, err := ref.GetStateSet().GetOrAddObject(key, &engine.OrderState{})
	if err != nil {
		return false, errors.ErrStore
	}
	obj := res.(*engine.OrderState)
	obj.Canceled = true
	obj.User = cancelArgs.User
	ids := hex.EncodeToString(id)
	//emit log
	AddCancelOrderEvtLog(ref, cancelArgs.User, ids)

	return true, errors.ErrOK
}

//cancel order by relay
func (p *DEXProtocol) DelegateCancelOrder(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	cancelArgs := new(facade.DCancelArgs)
	err := cancelArgs.Deserialize(reader)
	if err != nil {
		return false, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(cancelArgs.From) {
		return false, errors.ErrCtrInvalidateAuth
	}
	fromAddr := cancelArgs.From.GetAddress()
	//verify relay authorized
	if !isRelay(ref, fromAddr) {
		return false, errors.ErrDexUnAuthorized
	}
	userAddr := cancelArgs.User.GetAddress()
	res, err := ref.GetStateSet().GetOrAddUint64(utils.GetAccountKey(utils.KeyPrefixDCancelOrder, userAddr))
	if err != nil {
		return false, errors.ErrStore
	}
	if cancelArgs.Number <= res.Value {
		return false, errors.ErrCtrInvalidArgs
	}
	res.Value = cancelArgs.Number
	//emit log
	AddDelegateCancelOrderEvtLog(ref, cancelArgs.From, cancelArgs.User, cancelArgs.Number)
	return true, errors.ErrOK
}

//return the order state
func (p *DEXProtocol) GetOrderState(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	str, err := serialization.ReadString(reader)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	oId, err := hex.DecodeString(str)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	key := utils.GetOrderIdKey(oId)
	res, err := ref.GetStateSet().GetObject(key, &engine.OrderState{})
	if err != nil {
		return nil, errors.ErrStore.SetMsg(err.Error())
	}
	obj := res.(*engine.OrderState)
	return obj, errors.ErrOK
}

func settle(ref common.ContractRef, maker *engine.Order, taker *engine.Order, relay *engine.Relay, clear *engine.Clear) errors.Error {
	//update the state of order
	err := updateOrderState(ref, maker, taker)
	if err != errors.ErrOK {
		return err
	}
	//update balance of maker,taker,relay
	err = updateBalance(ref, maker, taker, clear)

	if err != errors.ErrOK {
		return err
	}
	return errors.ErrOK
}

//update the order status to state set
func updateOrderState(ref common.ContractRef, maker *engine.Order, taker *engine.Order) errors.Error {
	makerS, err := ref.GetStateSet().GetOrAddObject(string(maker.OrderIdKey), &engine.OrderState{})
	if err != nil {
		ref.Logger().Error("update dex order ", maker.OrderId, " error:", err)
		return errors.ErrStore
	}
	m := makerS.(*engine.OrderState)
	m.Filled = maker.Filled
	m.User = maker.User
	takerS, err := ref.GetStateSet().GetOrAddObject(string(taker.OrderIdKey), &engine.OrderState{})
	if err != nil {
		ref.Logger().Error("update dex order ", taker.OrderId, " error:", err)
		return errors.ErrStore
	}
	t := takerS.(*engine.OrderState)
	t.Filled = taker.Filled
	t.User = taker.User
	return errors.ErrOK
}

//update balance of related users
func updateBalance(ref common.ContractRef, maker *engine.Order, taker *engine.Order, clear *engine.Clear) errors.Error {
	//update maker taker and relay balance
	base := taker.Base
	quote := taker.Quote
	if taker.IsSell() {
		_, err := BalanceSub(ref.GetStateSet(), taker.User, base, clear.TradeAmount)
		if err != errors.ErrOK {
			return err
		}
		_, err = BalanceAdd(ref.GetStateSet(), taker.User, quote, clear.TradeQuoteAmount-clear.TakerFee)
		if err != errors.ErrOK {
			return err
		}

		_, err = BalanceAdd(ref.GetStateSet(), maker.User, base, clear.TradeAmount-clear.MakerFee)
		if err != errors.ErrOK {
			return err
		}
		_, err = BalanceSub(ref.GetStateSet(), maker.User, quote, clear.TradeQuoteAmount)
		if err != errors.ErrOK {
			return err
		}
		if clear.TakerChannelFee > 0 {
			_, err = BalanceAdd(ref.GetStateSet(), taker.Channel, quote, clear.TakerChannelFee)
			if err != errors.ErrOK {
				return err
			}
		}
		if clear.MakerChannelFee > 0 {
			_, err = BalanceAdd(ref.GetStateSet(), maker.Channel, base, clear.MakerChannelFee)
			if err != errors.ErrOK {
				return err
			}
		}
		//sys fee is for governance contract
		if clear.TakerSysFee > 0 {
			err = AccountForGovernance(ref, quote, clear.TakerSysFee)
			if err != errors.ErrOK {
				return err
			}
		}
		if clear.MakerSysFee > 0 {
			err = AccountForGovernance(ref, base, clear.MakerSysFee)
			if err != errors.ErrOK {
				return err
			}
		}
	} else {
		_, err := BalanceAdd(ref.GetStateSet(), taker.User, base, clear.TradeAmount-clear.TakerFee)
		if err != errors.ErrOK {
			return err
		}
		_, err = BalanceSub(ref.GetStateSet(), taker.User, quote, clear.TradeQuoteAmount)
		if err != errors.ErrOK {
			return err
		}
		_, err = BalanceSub(ref.GetStateSet(), maker.User, base, clear.TradeAmount)
		if err != errors.ErrOK {
			return err
		}
		_, err = BalanceAdd(ref.GetStateSet(), maker.User, quote, clear.TradeQuoteAmount-clear.MakerFee)
		if err != errors.ErrOK {
			return err
		}
		if clear.TakerChannelFee > 0 {
			_, err = BalanceAdd(ref.GetStateSet(), taker.Channel, base, clear.TakerChannelFee)
			if err != errors.ErrOK {
				return err
			}
		}
		if clear.MakerChannelFee > 0 {
			_, err = BalanceAdd(ref.GetStateSet(), maker.Channel, quote, clear.MakerChannelFee)
			if err != errors.ErrOK {
				return err
			}
		}
		if clear.TakerSysFee > 0 {
			err = AccountForGovernance(ref, base, clear.TakerSysFee)
			if err != errors.ErrOK {
				return err
			}
		}
		if clear.MakerSysFee > 0 {
			err = AccountForGovernance(ref, quote, clear.MakerSysFee)
			if err != errors.ErrOK {
				return err
			}
		}
	}
	return errors.ErrOK
}
