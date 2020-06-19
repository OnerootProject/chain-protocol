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
	"fmt"
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/core/types"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPriceAmountMultiply(t *testing.T) {
	price := uint64(0.00000001 * 1e8)
	amount := uint64(0.01 * 1e8)

	fmt.Println(uint64(price*amount) / 1e8)
	assert.True(t, uint64(price*amount)/1e8 == 0, "not 0")

	price = 0.00000001 * 1e8
	amount = 1.2 * 1e8
	assert.True(t, uint64(price*amount)/1e8 == 1, "not 1")

	price = 0.00000001 * 1e8
	amount = 12 * 1e8
	assert.True(t, uint64(price*amount)/1e8 == 12, "not 12")
	//will overflow
	price = 1 * 1e8
	amount = 1e4 * 1e8
	fmt.Println(price * amount)

	//skip overflow,use big int
	bp := new(big.Int).SetUint64(price)
	ap := new(big.Int).SetUint64(amount)
	res := new(big.Int).Mul(bp, ap)
	res = new(big.Int).Div(res, new(big.Int).SetUint64(1e8))

	fmt.Println(res.Uint64())
	assert.True(t, res.Uint64() == 1e4*1e8, "overflow")
}

func TestMatchOrder(t *testing.T) {

	//taker buy condition
	res := checkMatched(
		0.1, 10,
		0.2, 10,
		10,
		true,
		0.1, 1)
	assert.True(t, res, "taker buy:price cross,full fill")
	res = checkMatched(
		0.1, 10,
		0.2, 10,
		5,
		true,
		0.1, 0.5)
	assert.True(t, res, "taker buy:price cross,partial fill")

	//taker sell condition
	res = checkMatched(
		0.1, 10,
		0.2, 10,
		10,
		false,
		0.2, 2)
	assert.True(t, res, "taker sell:price cross,full fill")
	res = checkMatched(
		0.1, 10,
		0.2, 10,
		5,
		false,
		0.2, 1)
	assert.True(t, res, "taker sell:price cross,partial fill")

	//check big int
	res = checkMatched(
		0.1, 1e9,
		0.1, 1e10,
		1e9,
		false,
		0.1, 1e8)
	assert.True(t, res, "amount big int match")
	//price *tradeAmount<=1E11
	res = checkMatched(
		1e2, 1e9,
		1e2, 1e10,
		1e9,
		false,
		1e2, 1e11)
	assert.True(t, res, "price big int match")

	checkMatchException(t)

}

//func TestPrecisionMatch(t *testing.T) {
//	base, _ := types.AddressFromHexString("0ddc425383c5bbf19b0be15192c18c4f033b2a76")
//	quote, _ := types.AddressFromHexString("0eec425383c5bbf19b0be15192c18c4f033b2a76")
//	quotePre := 1e0
//	basePre := 1e8
//	relay := Relay{
//		TradeAmount: uint64(10 * basePre),
//	}
//	buy := NewSellOrder(base, quote, uint64(0.1*1E8), uint64(10*1E8)), NewBuyOrder(base, quote, uint64(buyPrice*1E8), uint64(buyAmount*1E8))
//}

func checkMatchException(t *testing.T) {
	/**
	the following is to check exceptions
	*/
	//not match
	res := checkMatched(
		0.2, 10,
		0.1, 10,
		10,
		false,
		0.2, 2)
	assert.True(t, !res, "price spread,no match")

	//big int overflow.make sure:price*amount<=1E11
	//price *tradeAmount>1E11
	res = checkMatched(
		1e3, 1e9,
		1e3, 1e10,
		1e9,
		false,
		1e3, 1e12)
	assert.True(t, !res, "price big int overflow")
}
func checkMatched(sellPrice float64, sellAmount float64, buyPrice float64, buyAmount float64, trade float64, makerSell bool, expPrice float64, expQuote float64) bool {
	clear, error := match(
		sellPrice, sellAmount,
		buyPrice, buyAmount,
		trade,
		makerSell)
	//fmt.Println(clear.Price, clear.TradeQuoteAmount)
	if error != errors.ErrOK {
		fmt.Println(error)
		return false
	}
	if clear.Price == uint64(expPrice*1e8) && clear.TradeQuoteAmount == uint64(expQuote*1e8) {
		return true
	} else {
		return false
	}
}
func match(sellPrice float64, sellAmount float64, buyPrice float64, buyAmount float64, trade float64, makerSell bool) (*Clear, error) {
	maker, taker, relay := makeOrder(
		sellPrice, sellAmount,
		buyPrice, buyAmount,
		trade,
		makerSell)
	return doMatch(maker, taker, relay)
}
func doMatch(maker *Order, taker *Order, relay *Relay) (*Clear, error) {
	return MatchOrder(maker, taker, relay)

}

func makeOrder(sellPrice float64, sellAmount float64, buyPrice float64, buyAmount float64, trade float64, makerSell bool) (*Order, *Order, *Relay) {
	baseAddr, _ := types.AddressFromHexString("0ddc425383c5bbf19b0be15192c18c4f033b2a76")
	quoteAddr, _ := types.AddressFromHexString("0eec425383c5bbf19b0be15192c18c4f033b2a76")
	relay := &Relay{
		TradeAmount: uint64(trade * 1e8),
	}
	base := types.AccountFromAddress(baseAddr)
	quote := types.AccountFromAddress(quoteAddr)
	if makerSell {
		return NewSellOrder(base, quote, uint64(sellPrice*1e8), uint64(sellAmount*1e8)), NewBuyOrder(base, quote, uint64(buyPrice*1e8), uint64(buyAmount*1e8)), relay
	} else {
		return NewBuyOrder(base, quote, uint64(buyPrice*1e8), uint64(buyAmount*1e8)), NewSellOrder(base, quote, uint64(sellPrice*1e8), uint64(sellAmount*1e8)), relay
	}
}
func TestFee(t *testing.T) {
	var fr uint64 = 10
	var amount uint64 = 8977667788

	fmt.Println(amount * fr / 10000)
}
