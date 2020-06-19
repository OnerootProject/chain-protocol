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
	"github.com/oneroot-network/onerootchain/core/contract/common"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/engine"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/facade"
	dexutil "github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"github.com/oneroot-network/onerootchain/core/types"
	"strconv"
)

const (
	EvtLogDeposit             = "deposit" //deposit event log
	EvtLogWithdraw            = "withdraw"
	EvtLogPrepareWithdraw     = "prepareWithdraw"
	EvtLogCommitWithdraw      = "commitWithdraw"
	EvtLogTrade               = "trade"
	EvtLogDelegateWithdraw    = "delegateWithdraw"
	EvtLogCancelOrder         = "cancel"
	EvtLogSetRelay            = "setRelay"
	EvtLogDelegateCancelOrder = "delegateCancel"
)

func AddTransferEvtLog(ref common.ContractRef, evtLogName string, asset *ncom.AssetArgs, balance uint64) {
	ref.AddEventLog([]string{
		evtLogName,
		asset.Asset.String(),
		asset.From.String(),
		asset.To.String(),
		strconv.FormatUint(asset.Amount, 10),
		strconv.FormatUint(balance, 10),
	})
}
func AddTradeEvtLog(ref common.ContractRef, clear *engine.Clear, maker *engine.Order, taker *engine.Order) {
	var makerFee, takerFee, makerChannelFee, takerChannelFee string
	if taker.Side == "sell" {
		takerFee = dexutil.Uint64ToDecimal(clear.TakerFee, taker.QuoteDecimal)
		makerFee = dexutil.Uint64ToDecimal(clear.MakerFee, taker.BaseDecimal)
		takerChannelFee = dexutil.Uint64ToDecimal(clear.TakerChannelFee, taker.QuoteDecimal)
		makerChannelFee = dexutil.Uint64ToDecimal(clear.MakerChannelFee, taker.BaseDecimal)
	} else {
		takerFee = dexutil.Uint64ToDecimal(clear.TakerFee, taker.BaseDecimal)
		makerFee = dexutil.Uint64ToDecimal(clear.MakerFee, taker.QuoteDecimal)
		takerChannelFee = dexutil.Uint64ToDecimal(clear.TakerChannelFee, taker.BaseDecimal)
		makerChannelFee = dexutil.Uint64ToDecimal(clear.MakerChannelFee, taker.QuoteDecimal)
	}
	ref.AddEventLog([]string{
		EvtLogTrade,
		hex.EncodeToString(maker.OrderId),
		hex.EncodeToString(taker.OrderId),
		dexutil.Uint64ToDecimal(clear.Price, 8),
		dexutil.Uint64ToDecimal(clear.TradeAmount, taker.BaseDecimal),
		dexutil.Uint64ToDecimal(clear.TradeQuoteAmount, taker.QuoteDecimal),
		makerFee,
		takerFee,
		makerChannelFee,
		takerChannelFee,
	})
}

func AddDWithdrawEvtLog(ref common.ContractRef, args *facade.DWithdrawArgs, balance uint64) {
	ref.AddEventLog([]string{
		EvtLogDelegateWithdraw,
		args.Asset.String(),
		args.Relay.String(),
		args.From.String(),
		args.To.String(),
		args.Extra,
		strconv.FormatUint(args.Amount-args.Fee, 10),
		strconv.FormatUint(args.Fee, 10),
		strconv.FormatUint(balance, 10),
	})
}
func AddCancelOrderEvtLog(ref common.ContractRef, from *types.Account, ids string) {
	ref.AddEventLog([]string{
		EvtLogCancelOrder,
		from.String(),
		ids,
	})
}

func AddDelegateCancelOrderEvtLog(ref common.ContractRef, from *types.Account, user *types.Account, number uint64) {
	ref.AddEventLog([]string{
		EvtLogDelegateCancelOrder,
		from.String(),
		user.String(),
		strconv.FormatUint(number, 10),
	})
}
