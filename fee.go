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
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/core/contract/abi"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/engine"
	"github.com/oneroot-network/onerootchain/core/contract/native/prime"
	"github.com/oneroot-network/onerootchain/core/types"
	"math/big"
)

//calculate MakerFee,TakerFee,MakerChannelFee,TakerChannelFee,MakerSysFee,TakerSysFee
func countFee(ref common.ContractRef, globalParams GlobalParams, maker *engine.Order, taker *engine.Order, clear *engine.Clear) {
	tradeAmount := new(big.Int).SetUint64(clear.TradeAmount)
	tradeQuoteAmount := new(big.Int).SetUint64(clear.TradeQuoteAmount)
	if taker.IsSell() {
		clear.MakerFee, clear.TakerFee, clear.MakerChannelFee, clear.TakerChannelFee, clear.MakerSysFee, clear.TakerSysFee =
			doCountFee(ref, globalParams, maker, taker, tradeQuoteAmount, tradeAmount)
	} else {
		clear.MakerFee, clear.TakerFee, clear.MakerChannelFee, clear.TakerChannelFee, clear.MakerSysFee, clear.TakerSysFee =
			doCountFee(ref, globalParams, maker, taker, tradeAmount, tradeQuoteAmount)
	}
}

func doCountFee(ref common.ContractRef, globalParams GlobalParams, maker *engine.Order, taker *engine.Order, takerGet, makerGive *big.Int) (uint64, uint64, uint64, uint64, uint64, uint64) {
	var makerFee, takerFee, makerChannelFee, takerChannelFee, makerSysFee, takerSysFee uint64
	res := big.NewInt(1)
	//count taker sys fee
	if !isPrime(ref, taker.User) {
		takerSysFee = res.Mul(takerGet, new(big.Int).SetUint64(globalParams.TakerSysFeeRate)).
			Div(res, big.NewInt(10000)).Uint64()
	} else {
		ref.Logger().Debug("taker is prime")
		//multiply discount
		takerSysFee = res.Mul(takerGet, new(big.Int).SetUint64(globalParams.TakerSysFeeRate)).
			Mul(res, new(big.Int).SetUint64(globalParams.PrimeFeeDiscountPercent)).
			Div(res, big.NewInt(10000*100)).Uint64()
	}
	//count maker sys fee
	res = big.NewInt(1)
	if !isPrime(ref, maker.User) {
		makerSysFee = res.Mul(makerGive, new(big.Int).SetUint64(globalParams.MakerSysFeeRate)).
			Div(res, big.NewInt(10000)).Uint64()
	} else {
		ref.Logger().Debug("maker is prime")
		makerSysFee = res.Mul(makerGive, new(big.Int).SetUint64(globalParams.MakerSysFeeRate)).
			Mul(res, new(big.Int).SetUint64(globalParams.PrimeFeeDiscountPercent)).
			Div(res, big.NewInt(10000*100)).Uint64()
	}
	//count channel fee
	if taker.TakerFeeRate > 0 {
		res := big.NewInt(1)
		takerChannelFee = res.Mul(takerGet, new(big.Int).SetUint64(uint64(taker.TakerFeeRate))).Div(res, big.NewInt(10000)).Uint64()
	}
	if maker.MakerFeeRate > 0 {
		res := big.NewInt(1)
		makerChannelFee = res.Mul(makerGive, new(big.Int).SetUint64(uint64(maker.MakerFeeRate))).Div(res, big.NewInt(10000)).Uint64()
	}
	makerFee = makerSysFee + makerChannelFee
	takerFee = takerSysFee + takerChannelFee
	return makerFee, takerFee, makerChannelFee, takerChannelFee, makerSysFee, takerSysFee
}

//check prime
func isPrime(ref common.ContractRef, acc *types.Account) bool {
	if acc == nil {
		return false
	}
	encoder := abi.NewEncoder()
	err := encoder.Encode(acc)
	if err != nil {
		return false
	}
	engine, cErr := ref.NewExecuteEngine(ncom.PrimeNameCtrAccount, prime.IsPrimeUser, encoder.Bytes())
	if cErr != errors.ErrOK {
		return false
	}
	res, cErr := engine.Invoke()
	if cErr != errors.ErrOK {
		return false
	}
	if res == nil {
		return false
	}
	return res.(bool)
}
