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
	"github.com/oneroot-network/onerootchain/common/buffer"
	"github.com/oneroot-network/onerootchain/common/serialization"
	"github.com/oneroot-network/onerootchain/core/states"
)

type PrepareWithdrawState struct {
	Time   uint32 //apply time
	Amount uint64 //total applied amount
}

func (s *PrepareWithdrawState) Serialize(buf *buffer.Buffer) error {
	err := serialization.WriteUint32(buf, s.Time)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, s.Amount)
	if err != nil {
		return err
	}
	return nil
}
func (s *PrepareWithdrawState) Deserialize(buf *buffer.Buffer) error {
	t, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}
	s.Time = t
	amount, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	s.Amount = amount
	return nil
}
func (s *PrepareWithdrawState) Copy() states.StateObject {
	return &PrepareWithdrawState{
		Time:   s.Time,
		Amount: s.Amount,
	}
}
func (s *PrepareWithdrawState) DataSize() int {
	var size int
	size += serialization.GetUint32Size(s.Time)
	size += serialization.GetUint64Size(s.Amount)
	return size
}

type SpProfit struct {
	LatestRound   uint32
	LatestProfit  uint64
	HistoryProfit uint64
}

func (s *SpProfit) Serialize(buf *buffer.Buffer) error {
	err := serialization.WriteUint32(buf, s.LatestRound)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, s.LatestProfit)
	if err != nil {
		return err
	}
	return serialization.WriteUint64(buf, s.HistoryProfit)
}

func (s *SpProfit) Deserialize(buf *buffer.Buffer) error {
	latestRound, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}
	s.LatestRound = latestRound
	latestProfit, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	s.LatestProfit = latestProfit
	historyProfit, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	s.HistoryProfit = historyProfit
	return nil
}

func (s *SpProfit) Copy() states.StateObject {
	return &SpProfit{
		LatestRound:   s.LatestRound,
		LatestProfit:  s.LatestProfit,
		HistoryProfit: s.HistoryProfit,
	}
}

func (s *SpProfit) DataSize() int {
	var size int
	size += serialization.GetUint32Size(s.LatestRound)
	size += serialization.GetUint64Size(s.LatestProfit)
	size += serialization.GetUint64Size(s.HistoryProfit)
	return size
}
