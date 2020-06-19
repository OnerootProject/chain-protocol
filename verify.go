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

//verifications for dex protocol
package dex

import (
	"github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/engine"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/facade"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"github.com/oneroot-network/onerootchain/core/types"
)

//basic verification
func verify(ref common.ContractRef, tradeArgs *facade.TradeArgs) (*engine.Order, *engine.Order, *engine.Relay, errors.Error) {
	//verify chainID
	if ref.GetContext().ChainID != tradeArgs.Maker.ChainId || ref.GetContext().ChainID != tradeArgs.Taker.ChainId {
		return nil, nil, nil, errors.ErrDexChainIdError
	}
	//verify order expired or not
	if IsExpired(ref, tradeArgs.Maker.Expire) || IsExpired(ref, tradeArgs.Taker.Expire) {
		return nil, nil, nil, errors.ErrOrderExpired
	}
	//check fee
	if tradeArgs.Maker.MakerFeeRate > 10000 ||
		tradeArgs.Maker.TakerFeeRate > 10000 ||
		tradeArgs.Taker.TakerFeeRate > 10000 ||
		tradeArgs.Taker.MakerFeeRate > 10000 {
		return nil, nil, nil, errors.ErrFeeIllegal
	}
	//check trade side
	if !((tradeArgs.Taker.Side == Buy && tradeArgs.Maker.Side == Sell) ||
		(tradeArgs.Taker.Side == Sell && tradeArgs.Maker.Side == Buy)) {
		return nil, nil, nil, errors.ErrDexSideError
	}

	//convert order data to inner order
	makerOrder, err2 := tradeArgs.Maker.ToOrder(ref)
	if err2 != errors.ErrOK {
		ref.Logger().Error("convert maker order", "error", err2.String())
		return nil, nil, nil, err2
	}
	takerOrder, err2 := tradeArgs.Taker.ToOrder(ref)
	if err2 != errors.ErrOK {
		ref.Logger().Error("convert taker order", "error", err2.String())
		return nil, nil, nil, err2
	}
	relay, err2 := tradeArgs.Relay.ToRelay(takerOrder.IsSell(), takerOrder.BaseDecimal, takerOrder.QuoteDecimal)
	if err2 != errors.ErrOK {
		ref.Logger().Error("convert relay", "error", err2.String())
		return nil, nil, nil, err2
	}

	//check same pair
	if !(makerOrder.Base.Equal(takerOrder.Base) && makerOrder.Quote.Equal(takerOrder.Quote)) {
		return nil, nil, nil, errors.ErrDexPairError
	}
	//verify pair listed
	//if !IsPairListed(ref, makerOrder.Base, makerOrder.Quote) {
	//	return nil, nil, nil, errors.ErrPairUnList
	//}
	//verify order canceled by user
	if IsCanceled(ref, makerOrder.OrderId) || IsCanceled(ref, takerOrder.OrderId) {
		return nil, nil, nil, errors.ErrDexOrderCanceled
	}

	makerUserAddr := makerOrder.User.GetAddress()
	takerUserAddr := takerOrder.User.GetAddress()

	//verify order canceled by relay
	if IsCanceledByRelay(ref, makerUserAddr, tradeArgs.Maker.Salt) || IsCanceledByRelay(ref, takerUserAddr, tradeArgs.Taker.Salt) {
		return nil, nil, nil, errors.ErrDexOrderCanceled
	}

	//verify user from order and sig is same
	if !VerifySigUser(makerUserAddr, tradeArgs.Maker.Sig) {
		return nil, nil, nil, errors.ErrDexVerifySigUserError
	}
	if !VerifySigUser(takerUserAddr, tradeArgs.Taker.Sig) {
		return nil, nil, nil, errors.ErrDexVerifySigUserError
	}
	//verify sig of maker and order
	if !VerifySig(makerOrder.OrderId, tradeArgs.Maker.Sig) {
		return nil, nil, nil, errors.ErrDexVerifySigError
	}
	if !VerifySig(takerOrder.OrderId, tradeArgs.Taker.Sig) {
		return nil, nil, nil, errors.ErrDexVerifySigError
	}
	return makerOrder, takerOrder, relay, errors.ErrOK
}

func IsCanceled(ref common.ContractRef, oId []byte) bool {
	key := utils.GetOrderIdKey(oId)
	res, err := ref.GetStateSet().GetObject(key, &engine.OrderState{})
	if err != nil {
		ref.Logger().Warn("get cancel state error", "error", err)
		return true
	}
	return res.(*engine.OrderState).Canceled
}
func IsCanceledByRelay(ref common.ContractRef, user types.Address, sequence uint64) bool {
	res, err := ref.GetStateSet().GetOrAddUint64(utils.GetAccountKey(utils.KeyPrefixDCancelOrder, user))
	if err != nil {
		ref.Logger().Warn("get delegate cancel state error", "error", err)
		return true
	}
	return sequence <= res.Value
}

func IsExpired(ref common.ContractRef, expire uint32) bool {
	if expire == 0 || ref.GetContext().Timestamp < expire {
		return false
	} else {
		return true
	}
}

func VerifySig(data []byte, sig *types.Sig) bool {
	if err := sig.Verify(data); err != nil {
		return false
	} else {
		return true
	}
}
func VerifySigUser(user types.Address, sig *types.Sig) bool {
	addr, err := types.AddressFromMultiPublicKeys(sig.PublicKeys, sig.M)
	if err != nil {
		return false
	}
	return user.Equal(addr)
}
