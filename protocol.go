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
	"fmt"
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	gp "github.com/oneroot-network/onerootchain/core/contract/native/global_params"
	"github.com/oneroot-network/onerootchain/core/contract/native/utils"
	"github.com/oneroot-network/onerootchain/core/types"
)

const (
	Buy  = "buy"
	Sell = "sell"

	Deposit            = "deposit"
	Trade              = "trade"
	Cancel             = "cancel"
	DelegateCancel     = "delegateCancel"
	DelegateWithdraw   = "delegateWithdraw"
	SetRelay           = "setRelay"
	Relays             = "relays"
	OrderState         = "orderState"
	PrepareWithdraw    = "prepareWithdraw"
	CommitWithdraw     = "commitWithdraw"
	GetPrepareWithdraw = "getPrepareWithdrawState" //query the prepared withdraws
)

//system configs
const (
	//define fee params
	MakerSysFeeRate         = "makerSysFeeRate"         // maker sys fee rate.DIV(10000)
	TakerSysFeeRate         = "takerSysFeeRate"         // taker sys fee rate .DIV(10000)
	PrimeFeeDiscountPercent = "primeFeeDiscountPercent" //fee discount for prime user
	WithdrawApplyWaitTime   = "withdrawApplyWaitTime"   //apply wait time in 2pc withdraw
)

func init() {
	gp.RegisterParam(gp.NewValidateParam(MakerSysFeeRate, "0", gp.FeeRateValidator))
	gp.RegisterParam(gp.NewValidateParam(TakerSysFeeRate, "3", gp.FeeRateValidator))
	gp.RegisterParam(gp.NewValidateParam(PrimeFeeDiscountPercent, "80", gp.PercentValidator))
	gp.RegisterParam(gp.NewValidateParam(WithdrawApplyWaitTime, "10", gp.PositiveIntValidator))
}

type GlobalParams struct {
	MakerSysFeeRate         uint64 // maker sys fee rate.DIV(10000)
	TakerSysFeeRate         uint64 // taker sys fee rate .DIV(10000)
	PrimeFeeDiscountPercent uint64 //fee discount for prime user
	WithdrawApplyWaitTime   uint64 //apply wait time in 2pc withdraw
}

//the implementation of dex
type DEXProtocol struct {
	Address types.Address
}

func NewDexProtocol() *DEXProtocol {
	return &DEXProtocol{
		Address: ncom.DexAddress,
	}
}

func (p DEXProtocol) GetAddress() types.Address { return p.Address }

func (p *DEXProtocol) Invoke(ref common.ContractRef, method string, args []byte) (interface{}, errors.Error) {
	switch method {
	case ncom.BalanceOf:
		return p.BalanceOf(ref, args)
	case Deposit:
		return p.Deposit(ref, args)
	case ncom.Withdraw:
		return p.Withdraw(ref, args)
	case DelegateWithdraw:
		return p.DelegateWithdraw(ref, args)
	case PrepareWithdraw:
		return p.PrepareWithdraw(ref, args)
	case CommitWithdraw:
		return p.CommitWithdraw(ref, args)
	case GetPrepareWithdraw:
		return p.GetPrepareWithdrawState(ref, args)
	case Cancel:
		return p.CancelOrder(ref, args)
	case DelegateCancel:
		return p.DelegateCancelOrder(ref, args)
	case Trade:
		return p.Trade(ref, args)
	case OrderState:
		return p.GetOrderState(ref, args)
	case SetRelay:
		return p.SetRelay(ref, args)
	case Relays:
		return p.Relays(ref, args)
	case ncom.EpochEnd:
		return p.EpochEnd(ref, args)
	case ncom.ClaimSpProfit:
		return p.ClaimSpProfit(ref, args)
	default:
		return nil, errors.ErrCtrServiceNotFound
	}
}

func GetGlobalParams(ref common.ContractRef) (GlobalParams, errors.Error) {
	globalParams, cErr := utils.GetGlobalParams(ref,
		MakerSysFeeRate,
		TakerSysFeeRate,
		PrimeFeeDiscountPercent,
		WithdrawApplyWaitTime,
	)

	if cErr != errors.ErrOK {
		return GlobalParams{}, cErr
	}
	var params GlobalParams
	var err error
	params.MakerSysFeeRate, err = globalParams[0].GetUint64()
	if err != nil {
		return params, errors.ErrCtrExecute.SetMsg(fmt.Sprintf("get global param error:%s", err))
	}
	params.TakerSysFeeRate, err = globalParams[1].GetUint64()
	if err != nil {
		return params, errors.ErrCtrExecute.SetMsg(fmt.Sprintf("get global param error:%s", err))
	}
	params.PrimeFeeDiscountPercent, err = globalParams[2].GetUint64()
	if err != nil {
		return params, errors.ErrCtrExecute.SetMsg(fmt.Sprintf("get global param error:%s", err))
	}

	params.WithdrawApplyWaitTime, err = globalParams[3].GetUint64()
	if err != nil {
		return params, errors.ErrCtrExecute.SetMsg(fmt.Sprintf("get global param error:%s", err))
	}
	return params, errors.ErrOK

}
