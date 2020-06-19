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

package facade

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/oneroot-network/onerootchain/common/buffer"
	errors2 "github.com/oneroot-network/onerootchain/common/errors"
	"github.com/oneroot-network/onerootchain/common/serialization"
	"github.com/oneroot-network/onerootchain/core/contract/common"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/engine"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"github.com/oneroot-network/onerootchain/core/types"
	cryptocom "github.com/oneroot-network/onerootchain/crypto/common"
	"math"
	"strconv"
	"strings"
)

type RawOrderData struct {
	//chainId
	ChainId uint32
	//user's address
	User *types.Account
	//order pair such as ETH_USD,where ETH is symbol of base token and USD is symbol of quote token
	Pair string
	//'buy' or 'sell'
	Side string
	//the price of quote currency,8 decimals at most.will be multiplied 1E8 and stored in uint64 internally
	Price string
	//the amount user would like to buy/sell,8 decimals at most and max number is 100 billion .stored in uint64 internally
	Amount string
	//channel who collects the order for relay
	Channel *types.Account
	//fee rate needs payed for channel as maker.MakerFeeRate/10000
	MakerFeeRate uint32
	//fee rate needs payed for channel as taker.TakerFeeRate/10000
	TakerFeeRate uint32
	//expire time in unix second.default 0 means never expired
	Expire uint32
	//the random number to make the id unique
	Salt uint64
}

func (a *RawOrderData) OrderId() ([]byte, error) {
	//amount=&chain_id=&channel=&expire=&maker_fee_rate&pair=&price=&salt=&side=&taker_fee_rate=&user=
	var buffer bytes.Buffer
	buffer.WriteString("amount=")
	buffer.WriteString(a.Amount)
	buffer.WriteString("&chain_id=")
	buffer.WriteString(strconv.FormatInt(int64(a.ChainId), 10))
	buffer.WriteString("&channel=")
	buffer.WriteString(a.Channel.Address.ToBase58())
	buffer.WriteString("&expire=")
	buffer.WriteString(strconv.FormatInt(int64(a.Expire), 10))
	buffer.WriteString("&maker_fee_rate=")
	buffer.WriteString(strconv.FormatInt(int64(a.MakerFeeRate), 10))
	buffer.WriteString("&pair=")
	buffer.WriteString(a.Pair)
	buffer.WriteString("&price=")
	buffer.WriteString(a.Price)
	buffer.WriteString("&salt=")
	buffer.WriteString(strconv.FormatInt(int64(a.Salt), 10))
	buffer.WriteString("&side=")
	buffer.WriteString(a.Side)
	buffer.WriteString("&taker_fee_rate=")
	buffer.WriteString(strconv.FormatInt(int64(a.TakerFeeRate), 10))
	buffer.WriteString("&user=")
	buffer.WriteString(a.User.Address.ToBase58())
	res := sha256.Sum256(buffer.Bytes())
	return res[:], nil
}
func (a *RawOrderData) Serialize(buf *buffer.Buffer) error {
	err := serialization.WriteUint32(buf, a.ChainId)
	if err != nil {
		return err
	}
	err = a.User.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteString(buf, a.Pair)
	if err != nil {
		return err
	}

	err = serialization.WriteString(buf, a.Side)
	if err != nil {
		return err
	}

	err = serialization.WriteString(buf, a.Price)
	if err != nil {
		return err
	}
	err = serialization.WriteString(buf, a.Amount)
	if err != nil {
		return err
	}
	err = a.Channel.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteUint32(buf, a.MakerFeeRate)
	if err != nil {
		return err
	}
	err = serialization.WriteUint32(buf, a.TakerFeeRate)
	if err != nil {
		return err
	}
	err = serialization.WriteUint32(buf, a.Expire)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, a.Salt)
	if err != nil {
		return err
	}
	return nil
}
func (a *RawOrderData) Deserialize(buf *buffer.Buffer) error {
	by, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}
	a.ChainId = by
	user := new(types.Account)
	err = user.Deserialize(buf)
	if err != nil {
		return err
	}
	a.User = user
	pair, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.Pair = pair

	side, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.Side = side

	price, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.Price = price

	amount, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.Amount = amount

	channel := new(types.Account)
	err = channel.Deserialize(buf)
	if err != nil {
		return err
	}
	a.Channel = channel

	mfr, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}
	a.MakerFeeRate = mfr
	tfr, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}
	a.TakerFeeRate = tfr
	expire, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}
	a.Expire = expire

	salt, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	a.Salt = salt
	return nil
}

type OrderData struct {
	RawOrderData
	//signature of order data
	Sig *types.Sig
}

func (a *OrderData) Serialize(buf *buffer.Buffer) error {
	err := a.RawOrderData.Serialize(buf)
	if err != nil {
		return err
	}
	err = a.Sig.Serialize(buf)
	if err != nil {
		return err
	}
	return nil
}

func (a *OrderData) Deserialize(buf *buffer.Buffer) error {
	data := new(RawOrderData)
	err := data.Deserialize(buf)
	if err != nil {
		return err
	}
	a.RawOrderData = *data
	sig := new(types.Sig)
	err = sig.Deserialize(buf)
	if err != nil {
		return err
	}
	a.Sig = sig
	return nil
}

func (a *OrderData) String() string {
	by, _ := json.Marshal(a)
	return string(by)
}
func (a *OrderData) ToOrder(ref common.ContractRef) (*engine.Order, errors2.Error) {
	or := new(engine.Order)
	//var hash types.H256 = bcommon.Hash(buf.Bytes())
	oId, err := a.OrderId()
	if err != nil {
		return nil, errors2.ErrCtrInvalidArgs
	}
	or.OrderId = oId
	or.Side = a.Side
	base, quote, err := a.pairToAccount()
	if err != nil {
		return nil, errors2.ErrDexParsePairError
	}
	or.Base = base
	or.Quote = quote
	bp, err2 := utils.GetTokenDecimal(ref, base)
	if err2 != errors2.ErrOK {
		return nil, err2
	}
	or.BaseDecimal = bp
	or.BasePrecision = uint64(math.Pow10(int(bp)))
	qp, err2 := utils.GetTokenDecimal(ref, quote)
	if err2 != errors2.ErrOK {
		return nil, err2
	}
	or.QuoteDecimal = qp
	or.QuotePrecision = uint64(math.Pow10(int(qp)))

	or.Price, err = utils.DecimalToUint64(a.Price, 8)
	if err != nil {
		return nil, errors2.ErrInvalidNumber
	}
	or.Amount, err = utils.DecimalToUint64(a.Amount, bp)
	if err != nil {
		return nil, errors2.ErrInvalidNumber
	}
	or.User = a.User
	or.Filled = 0
	or.Surplus = or.Amount
	or.Channel = a.Channel
	or.MakerFeeRate = a.MakerFeeRate
	or.TakerFeeRate = a.TakerFeeRate
	if ref != nil {
		//get order filled from the state set
		orderKey := utils.GetOrderIdKey(or.OrderId)
		or.OrderIdKey = []byte(orderKey)
		res, err := ref.GetStateSet().GetOrAddObject(orderKey, &engine.OrderState{})
		if err != nil {
			return nil, errors2.ErrStore
		}
		or.Filled = res.(*engine.OrderState).Filled
		or.Surplus -= or.Filled
	}
	return or, errors2.ErrOK
}
func (a *OrderData) SignOrder(keys []cryptocom.PublicKey, pris []cryptocom.PrivateKey) (*types.Sig, error) {
	oId, err := a.OrderId()
	if err != nil {
		return nil, err
	}
	a.Sig = new(types.Sig)
	a.Sig.PublicKeys = keys
	a.Sig.M = uint8(len(pris))
	a.Sig.SigData = [][]byte{}
	for _, pri := range pris {
		sData, err := pri.Sign(oId)
		if err != nil {
			return nil, err
		}
		a.Sig.SigData = append(a.Sig.SigData, sData)
	}

	//if a.Sig == nil {
	//	a.Sig = &types.Sig{
	//		PublicKeys: []cryptocom.PublicKey{pri.PublicKey()},
	//		M:          1,
	//		SigData:    [][]byte{sData},
	//	}
	//} else {
	//	a.Sig.PublicKeys = append(a.Sig.PublicKeys, pri.PublicKey())
	//	a.Sig.SigData = append(a.Sig.SigData, sData)
	//	a.Sig.M++
	//}
	return a.Sig, nil
}

//convert string trade pair to token account.$base_$quote pattern like:BTC_USD
func (a *OrderData) pairToAccount() (*types.Account, *types.Account, error) {
	strs := strings.Split(a.Pair, "_")
	if len(strs) != 2 {
		return nil, nil, errors.New("pair error")
	}
	base, err := types.AccountFromString(strs[0])
	if err != nil {
		return nil, nil, err
	}
	quote, err := types.AccountFromString(strs[1])
	return base, quote, err
}

type RelayArgs struct {
	From        *types.Account
	TradeAmount string
	MakerFee    string
	TakerFee    string
}

func (a *RelayArgs) Serialize(buf *buffer.Buffer) error {
	err := a.From.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteString(buf, a.TradeAmount)
	if err != nil {
		return err
	}
	err = serialization.WriteString(buf, a.MakerFee)
	if err != nil {
		return err
	}
	err = serialization.WriteString(buf, a.TakerFee)
	if err != nil {
		return err
	}
	return nil
}
func (a *RelayArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	a.From = from
	ta, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.TradeAmount = ta
	mf, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.MakerFee = mf
	tf, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	a.TakerFee = tf
	return nil
}
func (a *RelayArgs) String() string {
	by, _ := json.Marshal(a)
	return string(by)
}
func (a *RelayArgs) ToRelay(isTakerSell bool, basePre, quotePre uint8) (*engine.Relay, errors2.Error) {
	r := new(engine.Relay)
	ta, err := utils.DecimalToUint64(a.TradeAmount, basePre)
	if err != nil {
		return nil, errors2.ErrInvalidNumber
	}
	r.TradeAmount = ta
	if isTakerSell {
		r.MakerFee, err = utils.DecimalToUint64(a.MakerFee, basePre)
		if err != nil {
			return nil, errors2.ErrInvalidNumber
		}
		r.TakerFee, err = utils.DecimalToUint64(a.TakerFee, quotePre)
		if err != nil {
			return nil, errors2.ErrInvalidNumber
		}
	} else {
		r.MakerFee, err = utils.DecimalToUint64(a.MakerFee, quotePre)
		if err != nil {
			return nil, errors2.ErrInvalidNumber
		}
		r.TakerFee, err = utils.DecimalToUint64(a.TakerFee, basePre)
		if err != nil {
			return nil, errors2.ErrInvalidNumber
		}
	}
	r.From = a.From
	return r, errors2.ErrOK
}

type TradeArgs struct {
	Maker *OrderData
	Taker *OrderData
	Relay *RelayArgs
}

func NewTradeArgs() *TradeArgs {
	return &TradeArgs{
		Maker: &OrderData{},
		Taker: &OrderData{},
		Relay: &RelayArgs{},
	}
}
func (a *TradeArgs) Serialize(buf *buffer.Buffer) error {
	if a.Maker == nil {
		return errors.New("null error")
	}
	err := a.Maker.Serialize(buf)
	if err != nil {
		return err
	}
	if a.Taker == nil {
		return errors.New("null error")
	}
	err = a.Taker.Serialize(buf)
	if err != nil {
		return err
	}
	if a.Relay == nil {
		return errors.New("null error")
	}
	err = a.Relay.Serialize(buf)
	if err != nil {
		return err
	}
	return nil
}
func (a *TradeArgs) Deserialize(buf *buffer.Buffer) error {
	err := a.Maker.Deserialize(buf)
	if err != nil {
		return err
	}
	err = a.Taker.Deserialize(buf)
	if err != nil {
		return err
	}
	err = a.Relay.Deserialize(buf)
	if err != nil {
		return err
	}
	return nil
}

func (a *TradeArgs) String() string {
	by, _ := json.Marshal(a)
	return string(by)
}

//delegate withdraw args
type DWithdrawArgs struct {
	Asset  *types.Account //asset
	From   *types.Account //sender
	To     *types.Account //receiver of asset
	Amount uint64         //amount to send
	Fee    uint64         //fee of user would like to pay
	Salt   uint64
	Extra  string         //extra info added
	Sig    *types.Sig     //signature of from
	Relay  *types.Account //relay(delegate) address
}

func (arg *DWithdrawArgs) Serialize(buf *buffer.Buffer) error {

	err := arg.Asset.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.To.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, arg.Amount)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, arg.Fee)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, arg.Salt)
	if err != nil {
		return err
	}
	err = serialization.WriteString(buf, arg.Extra)
	if err != nil {
		return err
	}
	err = arg.Sig.Serialize(buf)
	if err != nil {
		return err
	}
	return arg.Relay.Serialize(buf)
}
func (arg *DWithdrawArgs) Deserialize(buf *buffer.Buffer) error {
	asset := new(types.Account)
	err := asset.Deserialize(buf)
	if err != nil {
		return err
	}
	from := new(types.Account)
	err = from.Deserialize(buf)
	if err != nil {
		return err
	}
	to := new(types.Account)
	err = to.Deserialize(buf)
	if err != nil {
		return err
	}
	amount, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	fee, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	salt, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	extra, err := serialization.ReadString(buf)
	if err != nil {
		return err
	}
	sig := new(types.Sig)
	err = sig.Deserialize(buf)
	if err != nil {
		return err
	}
	relay := new(types.Account)
	err = relay.Deserialize(buf)
	if err != nil {
		return err
	}

	arg.Asset = asset
	arg.From = from
	arg.To = to
	arg.Amount = amount
	arg.Fee = fee
	arg.Salt = salt
	arg.Extra = extra
	arg.Sig = sig
	arg.Relay = relay
	return nil
}

func (arg *DWithdrawArgs) SignWithdraw(keys []cryptocom.PublicKey, pris []cryptocom.PrivateKey) (*types.Sig, error) {
	hash, err := arg.HashParams()
	if err != nil {
		return nil, err
	}
	arg.Sig = new(types.Sig)
	arg.Sig.PublicKeys = keys
	arg.Sig.M = uint8(len(pris))
	arg.Sig.SigData = [][]byte{}
	for _, pri := range pris {
		sData, err := pri.Sign(hash)
		if err != nil {
			return nil, err
		}
		arg.Sig.SigData = append(arg.Sig.SigData, sData)
	}
	return arg.Sig, nil
}

func (arg *DWithdrawArgs) HashParams() ([]byte, error) {
	//amount=&asset=&extra=&fee=&from=&salt=&to
	var buffer bytes.Buffer
	buffer.WriteString("amount=")
	buffer.WriteString(strconv.FormatInt(int64(arg.Amount), 10))
	buffer.WriteString("&asset=")
	buffer.WriteString(arg.Asset.Address.ToBase58())
	buffer.WriteString("&extra=")
	buffer.WriteString(arg.Extra)
	buffer.WriteString("&fee=")
	buffer.WriteString(strconv.FormatInt(int64(arg.Fee), 10))
	buffer.WriteString("&from=")
	buffer.WriteString(arg.From.Address.ToBase58())
	buffer.WriteString("&salt=")
	buffer.WriteString(strconv.FormatInt(int64(arg.Salt), 10))
	buffer.WriteString("&to=")
	buffer.WriteString(arg.To.Address.ToBase58())
	res := sha256.Sum256(buffer.Bytes())
	return res[:], nil
}

type ListAssetArgs struct {
	From  *types.Account
	Base  *types.Account
	Quote *types.Account
}

func (arg *ListAssetArgs) Serialize(buf *buffer.Buffer) error {
	err := arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.Base.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.Quote.Serialize(buf)
	if err != nil {
		return err
	}
	return nil
}
func (arg *ListAssetArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	base := new(types.Account)
	err = base.Deserialize(buf)
	if err != nil {
		return err
	}
	quote := new(types.Account)
	err = quote.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.From = from
	arg.Base = base
	arg.Quote = quote
	return nil
}
func (arg *ListAssetArgs) EncodePair(prefix string) []byte {
	pLen := len(prefix)
	size := types.AddressSize + pLen
	key := make([]byte, size)
	copy(key[0:], ncom.DexAddress[:])
	copy(key[types.AddressSize:], []byte(prefix))
	return key
}

type CancelOrderArgs struct {
	RawOrderData
}

type DCancelArgs struct {
	From   *types.Account
	User   *types.Account
	Number uint64 //orders number this timestamp will be invalid
}

func (arg *DCancelArgs) Serialize(buf *buffer.Buffer) error {
	err := arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.User.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, arg.Number)
	if err != nil {
		return err
	}
	return nil
}
func (arg *DCancelArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.From = from
	user := new(types.Account)
	err = user.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.User = user
	t, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	arg.Number = t
	return nil
}

type SetterArgs struct {
	From   *types.Account
	Target *types.Account
	Value  bool
}

func (arg *SetterArgs) Serialize(buf *buffer.Buffer) error {
	err := arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.Target.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteBool(buf, arg.Value)
	if err != nil {
		return err
	}
	return nil
}
func (arg *SetterArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.From = from
	target := new(types.Account)
	err = target.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.Target = target
	v, err := serialization.ReadBool(buf)
	if err != nil {
		return err
	}
	arg.Value = v
	return nil
}

type CommitWithdrawArgs struct {
	From  *types.Account
	Asset *types.Account
}

func (arg *CommitWithdrawArgs) Serialize(buf *buffer.Buffer) error {
	err := arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	return arg.Asset.Serialize(buf)
}

func (arg *CommitWithdrawArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.From = from
	asset := new(types.Account)
	err = asset.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.Asset = asset
	return nil
}

type AnchorArgs struct {
	From  *types.Account
	Asset *types.Account
}

func (arg *AnchorArgs) Serialize(buf *buffer.Buffer) error {
	err := arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	return arg.Asset.Serialize(buf)
}

func (arg *AnchorArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.From = from
	asset := new(types.Account)
	err = asset.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.Asset = asset
	return nil
}

type AssetFee struct {
	Asset  *types.Account
	Amount uint64
}

func (a *AssetFee) Serialize(buf *buffer.Buffer) error {
	err := a.Asset.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteUint64(buf, a.Amount)
	if err != nil {
		return err
	}
	return nil
}
func (a *AssetFee) Deserialize(buf *buffer.Buffer) error {
	asset := new(types.Account)
	err := asset.Deserialize(buf)
	if err != nil {
		return err
	}
	a.Asset = asset
	amount, err := serialization.ReadUint64(buf)
	if err != nil {
		return err
	}
	a.Amount = amount
	return nil
}

type LinkMarketArgs struct {
	From   *types.Account
	Asset  *types.Account
	Target *types.Account
	Value  bool
}

func (arg *LinkMarketArgs) Serialize(buf *buffer.Buffer) error {
	err := arg.From.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.Asset.Serialize(buf)
	if err != nil {
		return err
	}
	err = arg.Target.Serialize(buf)
	if err != nil {
		return err
	}
	err = serialization.WriteBool(buf, arg.Value)
	if err != nil {
		return err
	}
	return nil
}
func (arg *LinkMarketArgs) Deserialize(buf *buffer.Buffer) error {
	from := new(types.Account)
	err := from.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.From = from
	asset := new(types.Account)
	err = asset.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.Asset = asset
	target := new(types.Account)
	err = target.Deserialize(buf)
	if err != nil {
		return err
	}
	arg.Target = target
	v, err := serialization.ReadBool(buf)
	if err != nil {
		return err
	}
	arg.Value = v
	return nil
}
