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
	"fmt"
	"github.com/oneroot-network/onerootchain/common/buffer"
	"github.com/oneroot-network/onerootchain/core/contract/abi"
	ncom "github.com/oneroot-network/onerootchain/core/contract/native/common"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/facade"
	"github.com/oneroot-network/onerootchain/core/types"
	"github.com/oneroot-network/onerootchain/crypto"
	"github.com/oneroot-network/onerootchain/crypto/common"
	"github.com/oneroot-network/onerootchain/wallet"
	"github.com/oneroot-network/onerootchain/wallet/keystore"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

///Notice:before run test,make sure run oneroot chain client locally and create at least 3 accounts in wallet
const pwd = "0"
const walletPath = "~/.ks_bt"

var chainId uint32 = 3

var priKeys = []string{"012eb3e7cf9abc248f621ec4e5f56c8290a7bab1b89791001a1202f07d09ae511c", //BCzGT34XnHdX41sevKsRtaLMqYNie28gpU
	"01799b40d5f9c3a343655d08717f82bdb0df26bb974a67f300919ed3d5570789bb",                    //BFwqnoV19kUz4wbsRReW1imFEaUWJrXVFU
	"018ade9b2b7d80e078d994a47e9e09f2bd0cb7b3ec2282bf08d55152e6c25aa19f",                    //BNLxN8Nt7QSMVbR3iC8TgGgkvcd6qofAE3
	"01876f95d7a6ecb91b160fce947bac853b1d7233f2cc68615c26a66135f2fa1158",                    //B7vG5TNRR8p2gyFareZky1ZvhPmTWcMQuW
}

//var wallet = GetWallet(walletPath, pwd)
var wa = GetWalletFromPri()
var key0 = Getkey(0)
var account0 = types.AccountFromAddress(key0.Address)
var key1 = Getkey(1)
var account1 = types.AccountFromAddress(key1.Address)
var key2 = Getkey(2)

var relay = account0
var maker = account1

func GetPrivateKey(path, from string, pwd string) common.PrivateKey {
	wa := GetWallet(path, pwd)
	acc, err := types.AddressFromBase58(from)
	wa.Unlock(acc, []byte(pwd))
	if err != nil {
		panic(err)
	}
	key, err := wa.GetPrivateKey(acc)
	if err != nil {
		panic(err)
	}
	return key
}
func GetWallet(path, pwd string) *wallet.Wallet {
	ks, err := keystore.NewFileKeyStore(path)
	if err != nil {
		panic(err)
	}
	return wallet.NewWallet(ks)
}
func GetWalletFromPri() *wallet.Wallet {
	ks, err := keystore.NewFileKeyStore(walletPath)
	if err != nil {
		panic(err)
	}
	wa := wallet.NewWallet(ks)
	//import from private key
	for index, key := range priKeys {
		by, err := hex.DecodeString(key)
		if err != nil {
			panic(err)
		}
		pri, err := crypto.PrivateKeyFromBytes(by)
		if err != nil {
			panic(err)
		}
		_, _ = wa.ImportKey(pri, []byte(pwd), wallet.CryptStandard)
		acc, _ := wa.GetKeyByIndex(index)
		fmt.Println(acc.Address.ToBase58())
	}

	return wa
}

func Getkey(index int) *wallet.Key {
	acc, err := wa.GetKeyByIndex(index)
	if err != nil {
		panic(err)
	}
	return acc
}
func TestGetArgs(t *testing.T) {
	by, _ := hex.DecodeString("0d030d0c04000a01000628d70f66d43b7c19ebfc3b289db451018d419e0745423633766d50456e316261374841394659505853374e4a6758513855324a555767565f424c454359556651365331516b56334c726a436f6934564b4475376e5a554338346a07036275790703332e340705312e3938320a0100e4f138c34d6d531a4bfca58df9171366ab6e49530405040504000686016ed4d8609a0d030c010e010384f61c4ee853a24569b91ab97dd82d9cb1158b58b4208903de09dc2c4287d38102010c01093f019483bb07e090f41ea0a7a9160be16e5aef356b36d4045172c1af816c03ddcb6c722580a92d633288db3ac21dc5dc753ed167ad4abcd6b374cb05004c22170d0c04000a01000628d70f66d43b7c19ebfc3b289db451018d419e0745423633766d50456e316261374841394659505853374e4a6758513855324a555767565f424c454359556651365331516b56334c726a436f6934564b4475376e5a554338346a070473656c6c0703312e360705312e3539300a0100e4f138c34d6d531a4bfca58df9171366ab6e49530405040504000686016ed4d8624d0d030c010e010384f61c4ee853a24569b91ab97dd82d9cb1158b58b4208903de09dc2c4287d38102010c010941010e24235b4b8207f1623126624bb7fb36f6d73945a48bf1d9242c644ab3614868d8f22a2cd6b997072ab09242322d036898d94b2b842ed4d9874c6f6167d2057b0d040a01009401c675246496be0fff800887cc14529f8080970705302e373339070130070130")
	decoder := abi.NewDecoder(by)
	//decoder.GetDataType()
	//decoder.GetLen()
	fmt.Println(by)
	args := facade.NewTradeArgs()
	decoder.Decode(args)
	//args.Deserialize(buffer.NewBuffer(by))

	fmt.Println(decoder, args)
}

func TestSignOrder(t *testing.T) {
	addr := account0.Address.ToBase58()
	acc := GetPrivateKey(walletPath,
		addr,
		pwd)
	maker := facade.OrderData{
		RawOrderData: facade.RawOrderData{
			ChainId:      chainId,
			User:         account0,
			Price:        "0.00157899",
			Amount:       "100.5", //decimal 4
			Side:         "buy",   //buy
			Pair:         "ETH_USD",
			Salt:         1,
			Channel:      account0,
			MakerFeeRate: 0,
			Expire:       0,
		},
	}
	oId, err := maker.OrderId()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(oId))
	if hex.EncodeToString(oId) != "45682e95f85caa3f4d7d8877f304b9f83b4d32a62474214ba48a53605ec73de9" {
		t.Fatal("orderId error")
	}
	sig, err := maker.SignOrder([]common.PublicKey{acc.PublicKey()}, []common.PrivateKey{acc})
	if err != nil {
		t.Fatal(err)
	}
	maker.Sig = sig
	buf := buffer.NewBuffer(nil)
	sig.Serialize(buf)
	fmt.Printf("sig:%s\n", hex.EncodeToString(buf.Bytes()))
	ss := new(types.Sig)
	err = ss.Deserialize(buf)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, ss.PublicKeys[0], sig.PublicKeys[0], "deserialize sig error")
	assert.Equal(t, ss.SigData, sig.SigData, "deserialize sig error")

	err = ss.Verify(oId)
	if err != nil {
		t.Fatal(err)
	}
	add, _ := types.AddressFromPublicKey(sig.PublicKeys[0])
	fmt.Println("recover:", add.ToBase58(), sig.PublicKeys[0].String())
	if add.ToBase58() != addr {
		t.Fatal("error address")
	}
}

//test verify sig error from java sdk
func TestVerifySigFromArg1(t *testing.T) {
	user, _ := types.AccountFromString("B51ebV5UErmqJ8ZwXdLDjzREVg4kfrMapH")
	channel, _ := types.AccountFromString("BRKceqEh9Y4sE4BAzsB913m87N7m9fsetR")
	maker := facade.OrderData{
		RawOrderData: facade.RawOrderData{
			ChainId:      0,
			User:         user,
			Price:        "3.4",
			Amount:       "1.982", //decimal 4
			Side:         "buy",   //buy
			Pair:         "B63vmPEn1ba7HA9FYPXS7NJgXQ8U2JUWgV_BLECYUfQ6S1QkV3LrjCoi4VKDu7nZUC84j",
			Salt:         1575528980634,
			Channel:      channel,
			MakerFeeRate: 5,
			TakerFeeRate: 5,
			Expire:       0,
		},
	}
	oId, err := maker.OrderId()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(oId))
	assert.Equal(t, "8e8fd4f0f82ad07bdfa2d369d8f85295fee9232379dc2ee0a43b0bc43c0ffbe1", hex.EncodeToString(oId), "order id unexpected")
	maker.Sig = decodeSig("010384f61c4ee853a24569b91ab97dd82d9cb1158b58b4208903de09dc2c4287d381", "019483bb07e090f41ea0a7a9160be16e5aef356b36d4045172c1af816c03ddcb6c722580a92d633288db3ac21dc5dc753ed167ad4abcd6b374cb05004c2217")
	fmt.Println(maker)
	buf := buffer.NewBuffer(nil)
	maker.Sig.Serialize(buf)
	fmt.Printf("sig:%x\n", buf.Bytes())

	assert.True(t, VerifySigUser(user.GetAddress(), maker.Sig), "verify sig user error")
	assert.True(t, VerifySig(oId, maker.Sig), "verify sig error")
}
func TestVerifySigData(t *testing.T) {
	sigData, _ := hex.DecodeString("01010384f61c4ee853a24569b91ab97dd82d9cb1158b58b4208903de09dc2c4287d38101013f019483bb07e090f41ea0a7a9160be16e5aef356b36d4045172c1af816c03ddcb6c722580a92d633288db3ac21dc5dc753ed167ad4abcd6b374cb05004c2217")
	data, _ := hex.DecodeString("8e8fd4f0f82ad07bdfa2d369d8f85295fee9232379dc2ee0a43b0bc43c0ffbe1")
	sig := new(types.Sig)
	sig.Deserialize(buffer.NewBuffer(sigData))
	fmt.Println(sig.Verify(data))
}

func decodeSig(pub string, sig string) *types.Sig {
	data, _ := hex.DecodeString(pub)

	pk, _ := crypto.DeserializePublicKey(buffer.NewBuffer(data))
	pks := []common.PublicKey{pk}
	sigData, _ := hex.DecodeString(sig)
	sigd := [][]byte{sigData}
	return &types.Sig{
		PublicKeys: pks,
		M:          1,
		SigData:    sigd,
	}
}
func getMultiAddr() (types.Address, []common.PublicKey, []common.PrivateKey) {
	k1 := key0
	k2 := key1
	k3 := key2
	pubKeys := append([]common.PublicKey{}, k1.PublicKey, k2.PublicKey, k3.PublicKey)
	m := uint8(2)

	addr, err := types.AddressFromMultiPublicKeys(pubKeys, m)
	if err != nil {
		panic(err)
	}
	pri1 := GetPrivateKey(walletPath,
		key1.Address.ToBase58(),
		pwd)
	pri2 := GetPrivateKey(walletPath,
		key2.Address.ToBase58(),
		pwd)
	pris := []common.PrivateKey{pri1, pri2}
	return addr, pubKeys, pris
}
func TestMultiSignOrder(t *testing.T) {
	// 2/3 multi sig
	k1 := key0
	k2 := key1
	k3 := key2
	pubKeys := append([]common.PublicKey{}, k1.PublicKey, k2.PublicKey, k3.PublicKey)
	m := uint8(2)

	addr, err := types.AddressFromMultiPublicKeys(pubKeys, m)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("multi sig address:", addr.ToBase58())

	acc := types.AccountFromAddress(addr)
	pri1 := GetPrivateKey(walletPath,
		key1.Address.ToBase58(),
		pwd)
	pri2 := GetPrivateKey(walletPath,
		key2.Address.ToBase58(),
		pwd)

	order := facade.OrderData{
		RawOrderData: facade.RawOrderData{
			ChainId:      chainId,
			User:         acc,
			Price:        "0.00157899",
			Amount:       "100.5", //decimal 4
			Side:         "buy",   //buy
			Pair:         "ETH_USD",
			Salt:         1,
			Channel:      acc,
			MakerFeeRate: 0,
			Expire:       0,
		},
	}
	oId, err := order.OrderId()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.EncodeToString(oId))
	if hex.EncodeToString(oId) != "37d3b2670a3a9022af1393e4b017ba37c832944d0c847e881120a5029f486244" {
		t.Fatal("orderId error")
	}
	sig, err := order.SignOrder(pubKeys, []common.PrivateKey{pri1, pri2})
	if err != nil {
		t.Fatal(err)
	}
	order.Sig = sig
	buf := buffer.NewBuffer(nil)
	sig.Serialize(buf)
	fmt.Printf("sig:%s\n", hex.EncodeToString(buf.Bytes()))
	ss := new(types.Sig)
	err = ss.Deserialize(buf)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, ss.PublicKeys[0], sig.PublicKeys[0], "deserialize sig error")
	assert.Equal(t, ss.PublicKeys[1], sig.PublicKeys[1], "deserialize sig error")
	assert.Equal(t, ss.PublicKeys[2], sig.PublicKeys[2], "deserialize sig error")
	assert.Equal(t, ss.M, sig.M, "deserialize sig error")
	assert.Equal(t, ss.SigData, sig.SigData, "deserialize sig error")

	err = ss.Verify(oId)
	if err != nil {
		t.Fatal(err)
	}

	add, _ := types.AddressFromMultiPublicKeys(ss.PublicKeys, ss.M)
	fmt.Println("recover:", add.ToBase58())
	if !add.Equal(addr) {
		t.Fatal("error address")
	}
}
func TestSignWithdraw(t *testing.T) {
	fmt.Println("--->", ncom.onerootTokenAddress.ToBase58())
	from := maker
	asset, _ := types.AccountFromString("BCzGT34XnHdX41sevKsRtaLMqYNie28gpU")
	pri := GetPrivateKey(walletPath,
		from.String(),
		pwd)
	args := &facade.DWithdrawArgs{
		Relay:  relay,
		Asset:  asset,
		From:   maker,
		To:     maker,
		Amount: 1,
		Fee:    19898921,
		Salt:   1557729851,
		Extra:  "{\"recipientAddresst\":\"ATGdpatRjPZES5KhNays3UDfyZxcsWERFGHJ\"}",
	}
	wId, _ := args.HashParams()
	fmt.Println("wId:", hex.EncodeToString(wId))
	if hex.EncodeToString(wId) != "1fe59a5052e6978cedef7b04acfbf94c3d060a3aa6ba2661eba2f37a97b5104b" {
		t.Fatal("withdrawId error")
	}
	sig1, _ := args.SignWithdraw([]common.PublicKey{pri.PublicKey()}, []common.PrivateKey{pri})
	err := sig1.Verify(wId)
	if err != nil {
		t.Fatal("verify sig error")
	}
	args.Sig = sig1
	buf := buffer.NewBuffer(nil)
	args.Sig.Serialize(buf)
	fmt.Printf("%x,%x\n", wId, buf.Bytes())

	buf = buffer.NewBuffer(nil)
	err = args.Serialize(buf)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("serialize DWithdrawArgs:%x\n", buf.Bytes())
	withdrawArgs := new(facade.DWithdrawArgs)
	err = withdrawArgs.Deserialize(buf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, withdrawArgs.Sig.PublicKeys[0], withdrawArgs.Sig.PublicKeys[0], "deserialize sig error")
	assert.Equal(t, withdrawArgs.Sig.SigData, withdrawArgs.Sig.SigData, "deserialize sig error")
}

func TestSignWithdrawMultisig(t *testing.T) {
	relay, _ := types.AccountFromString("BCzGT34XnHdX41sevKsRtaLMqYNie28gpU")
	asset, _ := types.AccountFromString("BCzGT34XnHdX41sevKsRtaLMqYNie28gpU")
	addr, pubKeys, pris := getMultiAddr()

	acc := types.AccountFromAddress(addr)
	args := &facade.DWithdrawArgs{
		Relay:  relay,
		Asset:  asset,
		From:   acc,
		To:     acc,
		Amount: 1,
		Fee:    19898921,
		Salt:   1557729851,
		Extra:  "{\"recipientAddresst\":\"ATGdpatRjPZES5KhNays3UDfyZxcsWERFGHJ\"}",
	}
	wId, _ := args.HashParams()
	fmt.Println("wId:", hex.EncodeToString(wId))
	if hex.EncodeToString(wId) != "2d3beec2c1e597ec7e04f94e76d966caa8c339591f47f7bfed2dfb7244115248" {
		t.Fatal("withdrawId error")
	}
	sig1, _ := args.SignWithdraw(pubKeys, pris)
	args.Sig = sig1
	buf := buffer.NewBuffer(nil)
	args.Sig.Serialize(buf)
	fmt.Printf("%x,%x\n", wId, buf.Bytes())

	buf = buffer.NewBuffer(nil)
	err := args.Serialize(buf)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("serialize DWithdrawArgs:%x\n", buf.Bytes())
	withdrawArgs := new(facade.DWithdrawArgs)
	err = withdrawArgs.Deserialize(buf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, withdrawArgs.Sig.PublicKeys[0], withdrawArgs.Sig.PublicKeys[0], "deserialize sig error")
	assert.Equal(t, withdrawArgs.Sig.SigData, withdrawArgs.Sig.SigData, "deserialize sig error")

	if err = withdrawArgs.Sig.Verify(wId); err != nil {
		t.Fatal(err)
	}
}

func TestOther(t *testing.T) {
	y, m, d := time.Now().Date()

	fmt.Println(time.Now().Truncate(time.Hour * 24))
	fmt.Println(y*1e4 + int(m)*1e2 + d)
	fmt.Println(ncom.DexAddress.ToBase58())
}
