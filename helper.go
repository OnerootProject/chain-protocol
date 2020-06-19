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
	common2 "github.com/oneroot-network/onerootchain/common"
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/core/contract/abi"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/oneroot_token"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/facade"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"github.com/oneroot-network/onerootchain/core/states"
	"github.com/oneroot-network/onerootchain/core/types"
)

//do transfer asset using transferAgs
//returns balance of `To` after deposit or error
func DoDeposit(ref common.ContractRef, asset *ncom.AssetArgs) (uint64, errors.Error) {
	transferAgs := &ncom.TransferArgs{
		From:   asset.From,
		To:     ncom.DexCtrAccount,
		Amount: asset.Amount,
	}
	cErr := DoTransfer(ref, asset.Asset, transferAgs)
	if cErr != errors.ErrOK {
		return 0, cErr
	}
	toAddr := asset.To.GetAddress()
	assetAddr := asset.Asset.GetAddress()
	//modify state of dex
	balanceKey := utils.GetBalanceKey(toAddr, assetAddr)
	balance, err := ref.GetStateSet().GetOrAddUint64(balanceKey)
	if err != nil {
		ref.Logger().Error("deposit to dex error", "token", asset.Asset.String(), "from", asset.From.String(), "to", asset.To.String(), "amount", asset.Amount, "error", err)
		return 0, errors.ErrStore
	}
	v, overflow := common2.SafeAdd(balance.Value, asset.Amount)
	if overflow {
		return 0, errors.ErrCtrOverflow
	}
	balance.Value = v
	return balance.Value, cErr
}

//do transfer asset using transferAgs
//returns balance of `From` after withdraw or error
func DoWithdraw(ref common.ContractRef, asset *ncom.AssetArgs) (uint64, errors.Error) {
	transferAgs := &ncom.TransferArgs{
		From:   ncom.DexCtrAccount,
		To:     asset.To,
		Amount: asset.Amount,
	}
	//modify state of dex
	stateSet := ref.GetStateSet()
	cErr := DoTransfer(ref, asset.Asset, transferAgs)
	if cErr != errors.ErrOK {
		return 0, cErr
	}
	return BalanceSub(stateSet, asset.From, asset.Asset, asset.Amount)
}

//delegate withdraw asset
//fee will be paid to relay
func DoDelegateWithdraw(ref common.ContractRef, args *facade.DWithdrawArgs) (uint64, errors.Error) {
	transferAgs := &ncom.TransferArgs{
		From:   ncom.DexCtrAccount,
		To:     args.To,
		Amount: args.Amount - args.Fee,
	}
	//transfer asset to `To`
	cErr := DoTransfer(ref, args.Asset, transferAgs)
	if cErr != errors.ErrOK {
		return 0, cErr
	}
	//update balance state
	//add fee to relay
	_, cErr = BalanceAdd(ref.GetStateSet(), args.Relay, args.Asset, args.Fee)
	if cErr != errors.ErrOK {
		return 0, cErr
	}
	//sub amount of `From`
	return BalanceSub(ref.GetStateSet(), args.From, args.Asset, args.Amount)
}

func DoApplyWithdraw(ref common.ContractRef, asset *ncom.AssetArgs) (uint64, errors.Error) {
	//get balance
	stateSet := ref.GetStateSet()
	fromAddr := asset.From.GetAddress()
	assetAddr := asset.Asset.GetAddress()
	balance, err := stateSet.GetUint64(utils.GetBalanceKey(fromAddr, assetAddr))
	if err != nil {
		ref.Logger().Error("dex get balance error", "from", asset.From.String(), "asset", asset.Asset.String(), "error", err)
		return 0, errors.ErrStore
	}
	if balance.Value < asset.Amount {
		return 0, errors.ErrCtrBalanceNotEnough
	}
	//add to apply state
	key := utils.GetPreparedWithdrawKey(fromAddr, assetAddr)
	res, err := stateSet.GetOrAddObject(key, &PrepareWithdrawState{})
	if err != nil {
		ref.Logger().Error("get apply withdraw state error", "from", asset.From.String(), "asset", asset.Asset.String(), "error", err)
		return 0, errors.ErrStore
	}
	ws := res.(*PrepareWithdrawState)
	am, overflow := common2.SafeAdd(ws.Amount, asset.Amount)
	if overflow {
		return 0, errors.ErrCtrOverflow
	}
	ws.Amount = am
	ws.Time = ref.GetContext().Timestamp
	return ws.Amount, errors.ErrOK
}

//transfer token by call token contract `transfer` method
func DoTransfer(ref common.ContractRef, token *types.Account, transferAgs *ncom.TransferArgs) errors.Error {
	args := []*ncom.TransferArgs{transferAgs}
	encoder := abi.NewEncoder()
	err := encoder.Encode(args)
	if err != nil {
		return errors.ErrCtrExecute
	}
	engine, cErr := ref.NewExecuteEngine(token, oneroot_token.Transfer, encoder.Bytes())
	if cErr != errors.ErrOK {
		return cErr
	}
	_, cErr = engine.Invoke()
	return cErr
}

func BalanceAdd(state states.StateSet, acc *types.Account, assetAcc *types.Account, amount uint64) (uint64, errors.Error) {
	user := acc.GetAddress()
	asset := assetAcc.GetAddress()
	balanceKey := utils.GetBalanceKey(user, asset)
	balance, err := state.GetOrAddUint64(balanceKey)
	if err != nil {
		return 0, errors.ErrStore
	}
	v, overflow := common2.SafeAdd(balance.Value, amount)
	if overflow {
		return 0, errors.ErrCtrOverflow
	}
	balance.Value = v
	return balance.Value, errors.ErrOK
}
func BalanceSub(state states.StateSet, acc *types.Account, assetAcc *types.Account, amount uint64) (uint64, errors.Error) {
	user := acc.GetAddress()
	asset := assetAcc.GetAddress()
	balanceKey := utils.GetBalanceKey(user, asset)
	balance, err := state.GetOrAddUint64(balanceKey)
	if err != nil {
		return 0, errors.ErrStore
	}
	if balance.Value < amount {
		return 0, errors.ErrCtrBalanceNotEnough
	}
	balance.Value -= amount
	if balance.Value == 0 {
		err = state.Delete(balanceKey)
		if err != nil {
			return 0, errors.ErrStore
		}
	}
	return balance.Value, errors.ErrOK
}

func AccountForGovernance(ref common.ContractRef, asset *types.Account, amount uint64) errors.Error {
	_, cErr := BalanceAdd(ref.GetStateSet(), ncom.GovernanceCtrAccount, asset, amount)
	if cErr != errors.ErrOK {
		return cErr
	}
	currentRound, err := ref.GetStateSet().GetUint64(utils.GetCurrentRoundKey())
	if err != nil {
		return errors.ErrCtrExecute.SetMsg("get current round error:%s", err)
	}
	assetAddr := asset.GetAddress()
	spProfitObj, err := ref.GetStateSet().GetOrAddObject(utils.GetSpProfitKey(assetAddr), new(SpProfit))
	if err != nil {
		return errors.ErrCtrExecute.SetMsg("get spProfit error:%s", err)
	}
	spProfit := spProfitObj.(*SpProfit)
	if spProfit.LatestRound == uint32(currentRound.Value) {
		spProfit.LatestProfit += amount
	} else {
		spProfit.HistoryProfit += spProfit.LatestProfit
		spProfit.LatestProfit = amount
		spProfit.LatestRound = uint32(currentRound.Value)
	}
	ref.Logger().Debug("governance get fee", "asset", assetAddr.ToBase58(), "amount", amount)
	return errors.ErrOK
}
