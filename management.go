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

///about permission:
//relay:relay permission is set by admin.method called:`trade`,`cancel`,`delegateWithdraw`
///common:all users can access
package dex

import (
	"github.com/oneroot-network/onerootchain/common/buffer"
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/common/serialization"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/facade"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"github.com/oneroot-network/onerootchain/core/contract/native/global_params"
	"github.com/oneroot-network/onerootchain/core/states"
	"github.com/oneroot-network/onerootchain/core/types"
	"strconv"
)

//only admin is allowed to set the relay
func (p *DEXProtocol) SetRelay(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	reader := buffer.NewBuffer(args)
	arg := new(facade.SetterArgs)
	err := arg.Deserialize(reader)
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	if !ref.CheckWitness(arg.From) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	fromAddr := arg.From.GetAddress()
	if !isOperator(ref, fromAddr) {
		return nil, errors.ErrDexUnAuthorized
	}
	targetAddr := arg.Target.GetAddress()
	key := utils.GetAccountKey(utils.KeyPrefixRelay, targetAddr)
	if arg.Value {
		state, err := ref.GetStateSet().GetOrAddBool(key)
		if err != nil {
			return nil, errors.ErrStore.SetMsg(err.Error())
		}
		state.Value = true
	} else {
		err = ref.GetStateSet().Delete(key)
		if err != nil {
			ref.Logger().Error("delete key error", "error", err)
			return nil, errors.ErrStore.SetMsg(err.Error())
		}
	}
	ref.AddEventLog([]string{
		EvtLogSetRelay,
		arg.Target.String(),
		strconv.FormatBool(arg.Value),
	})
	return nil, errors.ErrOK
}

//return all relays
func (p *DEXProtocol) Relays(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	prefixKey := utils.GetPrefixKey(utils.KeyPrefixRelay)
	prefixKeyLen := len(prefixKey)
	var relays []string
	finds, err := ref.GetStateSet().Find(prefixKey, new(states.BoolState))
	if err != nil {
		return relays, errors.ErrStore
	}
	for k := range finds {
		pp := []byte(k)
		admin, err := types.AddressFromBytes(pp[prefixKeyLen:])
		if err == nil {
			relays = append(relays, admin.ToBase58())
		}
	}
	return relays, errors.ErrOK
}

func (p *DEXProtocol) EpochEnd(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	if !ref.CheckWitness(ncom.GovernanceCtrAccount) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	currentRound, err := serialization.ReadUint32(buffer.NewBuffer(args))
	if err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	//set current round
	err = ref.GetStateSet().Set(utils.GetCurrentRoundKey(), &states.Uint64State{Value: uint64(currentRound)})
	if err != nil {
		return nil, errors.ErrCtrExecute.SetMsg("set current round error:%s", err)
	}
	return nil, errors.ErrOK
}

func (p *DEXProtocol) ClaimSpProfit(ref common.ContractRef, args []byte) (interface{}, errors.Error) {
	if !ref.CheckWitness(ncom.GovernanceCtrAccount) {
		return nil, errors.ErrCtrInvalidateAuth
	}
	asset := types.NewAccount()
	if err := asset.Deserialize(buffer.NewBuffer(args)); err != nil {
		return nil, errors.ErrCtrInvalidArgs
	}
	var profit uint64
	currentRound, err := ref.GetStateSet().GetUint64(utils.GetCurrentRoundKey())
	if err != nil {
		return profit, errors.ErrCtrExecute.SetMsg("get current round error:%s", err)
	}
	assetAddr := asset.GetAddress()
	spProfitObj, err := ref.GetStateSet().GetObject(utils.GetSpProfitKey(assetAddr), new(SpProfit))
	if err != nil {
		return profit, errors.ErrCtrExecute.SetMsg("get spProfit error:%s", err)
	}
	spProfit := spProfitObj.(*SpProfit)
	if spProfit.LatestRound == uint32(currentRound.Value) {
		profit = spProfit.HistoryProfit
		spProfit.HistoryProfit = 0
	} else {
		profit = spProfit.HistoryProfit + spProfit.LatestProfit
		spProfit.HistoryProfit = 0
		spProfit.LatestProfit = 0
		spProfit.LatestRound = uint32(currentRound.Value)
	}
	if profit == 0 {
		return profit, errors.ErrOK
	}
	_, cErr := DoWithdraw(ref, &ncom.AssetArgs{
		Asset:  asset,
		From:   ncom.GovernanceCtrAccount,
		To:     ncom.GovernanceCtrAccount,
		Amount: profit,
	})
	if cErr != errors.ErrOK {
		return uint64(0), cErr
	}
	if err := ref.GetStateSet().Set(utils.GetSpProfitKey(assetAddr), spProfit); err != nil {
		return uint64(0), errors.ErrCtrExecute.SetMsg("set spProfit error:%s", err)
	}
	if spProfit.HistoryProfit+spProfit.LatestProfit == 0 {
		if err := ref.GetStateSet().Delete(utils.GetSpProfitKey(assetAddr)); err != nil {
			return uint64(0), errors.ErrCtrExecute.SetMsg("delete spProfit error:%s", err)
		}
	}
	ref.Logger().Debug("governance get fee", "asset", assetAddr.ToBase58(), "amount", profit)
	return profit, errors.ErrOK
}

func isOperator(ref common.ContractRef, user types.Address) bool {
	operator, cErr := getOperator(ref)
	if cErr != errors.ErrOK {
		ref.Logger().Warn("get global operator error", "error", cErr)
		return false
	}
	addr := operator.GetAddress()
	return user.Equal(addr)
}
func isRelay(ref common.ContractRef, user types.Address) bool {
	//check relay
	key := utils.GetAccountKey(utils.KeyPrefixRelay, user)
	res, err := ref.GetStateSet().GetBool(key)
	if err != nil {
		return false
	}
	return res.Value
	//return true
}

func getOperator(ref common.ContractRef) (*types.Account, errors.Error) {
	engine, cErr := ref.NewExecuteEngine(ncom.GlobalParamCtrAccount, global_params.GetOperator, nil)
	if cErr != errors.ErrOK {
		return nil, cErr
	}
	ret, cErr := engine.Invoke()
	if cErr != errors.ErrOK {
		return nil, cErr
	}
	return ret.(*types.Account), errors.ErrOK
}
