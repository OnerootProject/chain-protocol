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

package engine

import (
	"github.com/oneroot-network/onerootchain/common/errors"
	"math"
	"math/big"
)

var Division = new(big.Int).SetUint64(1e8)

//match the price and clear.
//return Clear data for the up layer to update the states of both orders
func MatchOrder(maker *Order, taker *Order, relay *Relay) (*Clear, errors.Error) {

	if !priceMatch(maker, taker) {
		return nil, errors.ErrDexPriceNotMatch
	}
	tradeAmount := new(big.Int).SetUint64(relay.TradeAmount) //amount of base token
	if maker.Surplus < tradeAmount.Uint64() || taker.Surplus < tradeAmount.Uint64() {
		return nil, errors.ErrDexSurplusNotEnough
	}
	price := new(big.Int).SetUint64(maker.Price)
	res := big.NewInt(1)
	basePre := new(big.Int).SetUint64(uint64(maker.BasePrecision))
	quotePre := new(big.Int).SetUint64(uint64(maker.QuotePrecision))
	//overflow problem,use big.Int
	tradeQuoteAmount := res.Mul(price, tradeAmount).Mul(res, quotePre).Div(res, Division).Div(res, basePre)
	if tradeQuoteAmount.Uint64() == 0 {
		return nil, errors.ErrDexQuoteTradeAmountZero
	}
	//check overflow
	if tradeQuoteAmount.Cmp(new(big.Int).SetUint64(math.MaxUint64)) > 0 {
		return nil, errors.ErrCtrOverflow
	}
	maker.Surplus -= tradeAmount.Uint64()
	taker.Surplus -= tradeAmount.Uint64()
	maker.Filled += tradeAmount.Uint64()
	taker.Filled += tradeAmount.Uint64()
	clear := &Clear{
		Price:            price.Uint64(),
		TradeAmount:      tradeAmount.Uint64(),
		TradeQuoteAmount: tradeQuoteAmount.Uint64(),
	}
	return clear, errors.ErrOK
}

//match the price of the left &right orders
func priceMatch(left *Order, right *Order) bool {
	if left.IsSell() {
		return left.Price <= right.Price
	} else {
		return left.Price >= right.Price
	}
}
