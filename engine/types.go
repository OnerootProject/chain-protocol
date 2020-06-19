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
	"encoding/json"
	"github.com/oneroot-network/onerootchain/common/buffer"
	"github.com/oneroot-network/onerootchain/common/serialization"
	"github.com/oneroot-network/onerootchain/core/states"
	"github.com/oneroot-network/onerootchain/core/types"
)

///internal types of dex

type Order struct {
	User           *types.Account
	Channel        *types.Account
	OrderId        []byte
	Price          uint64
	Amount         uint64 //amount of base currency
	MakerFeeRate   uint32 // maker fee rate
	TakerFeeRate   uint32 //taker fee rate
	Side           string
	Base           *types.Account
	Quote          *types.Account
	Filled         uint64 //filled amount of baseToken
	Surplus        uint64 //surplus amount of base currency
	BasePrecision  uint64
	QuotePrecision uint64
	BaseDecimal    uint8
	QuoteDecimal   uint8
	//the order id key in state set
	OrderIdKey []byte
}

func NewBuyOrder(base, quote *types.Account, price uint64, amount uint64) *Order {
	return &Order{
		Price:          price,
		Amount:         amount,
		Side:           "buy",
		Base:           base,
		Quote:          quote,
		Filled:         0,
		Surplus:        amount,
		BasePrecision:  1,
		QuotePrecision: 1,
		BaseDecimal:    0,
		QuoteDecimal:   0,
	}
}
func NewSellOrder(base, quote *types.Account, price uint64, amount uint64) *Order {
	return &Order{
		Price:          price,
		Amount:         amount,
		Side:           "sell",
		Base:           base,
		Quote:          quote,
		Filled:         0,
		Surplus:        amount,
		BasePrecision:  1,
		QuotePrecision: 1,
		BaseDecimal:    0,
		QuoteDecimal:   0,
	}
}

func (a *Order) IsSell() bool {
	if a.Side == "sell" {
		return true
	} else {
		return false
	}
}

func (a *Order) String() string {
	by, _ := json.Marshal(a)
	return string(by)
}

type Relay struct {
	From        *types.Account
	TradeAmount uint64
	MakerFee    uint64
	TakerFee    uint64
}

func (a *Relay) String() string {
	by, _ := json.Marshal(a)
	return string(by)
}

type Clear struct {
	Price            uint64
	TradeAmount      uint64 //amount of base currency
	TradeQuoteAmount uint64 //amount of quote currency
	MakerFee         uint64 //MakerFee=MakerChannelFee+MakerSysFee
	TakerFee         uint64
	MakerChannelFee  uint64
	TakerChannelFee  uint64
	MakerSysFee      uint64
	TakerSysFee      uint64
}

type OrderState struct {
	User     *types.Account //user of the order
	Filled   uint64         // amount filled of the order
	Canceled bool           // indicate cancel or not.default:false
}

func (s *OrderState) Serialize(buf *buffer.Buffer) error {
	err := s.User.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, s.Filled)
	if err != nil {
		return err
	}
	err = serialization.WriteBool(buf, s.Canceled)
	if err != nil {
		return err
	}
	return nil
}
func (s *OrderState) Deserialize(buf *buffer.Buffer) error {
	user := new(types.Account)
	err := user.Deserialize(buf)
	if err != nil {
		return err
	}
	s.User = user
	filled, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	s.Filled = filled
	canceled, err := serialization.ReadBool(buf)
	if err != nil {
		return err
	}
	s.Canceled = canceled
	return nil
}

func (s *OrderState) Copy() states.StateObject {
	return &OrderState{
		User:     s.User,
		Filled:   s.Filled,
		Canceled: s.Canceled,
	}
}
func (s *OrderState) DataSize() int {
	var size int
	size += s.User.DataSize()
	size += serialization.GetUint64Size(s.Filled)
	size += serialization.GetBoolSize(s.Canceled)
	return size
}
