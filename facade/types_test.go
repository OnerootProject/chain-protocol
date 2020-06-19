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
	"fmt"
	"github.com/oneroot-network/onerootchain/core/contract/native/dex/utils"
	"math"
	"strconv"
	"testing"
)

func TestMaxNumber(t *testing.T) {
	//max 100 billion,if decimal is 8
	var aa uint64 = 1 * 1e11
	fmt.Println(aa)
}

func TestFloatConvertion(t *testing.T) {

	ss := strconv.FormatFloat(0.12345670, 'f', -1, 64)
	fmt.Println(ss)

	fl := 0.1234567821
	aa := fl * 1e8
	fmt.Println(uint64(math.Floor(aa)))
}
func TestDecimalToUint64(t *testing.T) {
	assertConvert(t, "0.1244", 8, 12440000)
	assertConvert(t, "0.1244", 1, 0)
	assertConvert(t, "0.1244", 4, 1244)
	assertConvert(t, "0.123456789", 8, 0)
	assertConvert(t, "1000000000000000000000000", 8, 0)
	assertConvert(t, "-10", 8, 0)
	assertConvert(t, "20.08", 4, 200800)
	assertConvert(t, "20.ae0", 4, 0)
	assertConvert(t, "1.022", 4, 10220)
	assertConvert(t, "akio", 4, 0)
	assertConvert(t, "12e.98", 4, 0)
}
func TestUint64ToDecimal(t *testing.T) {
	assertToDecimal(t, 10220, 4, "1.0220")
	assertToDecimal(t, 200800, 4, "20.0800")
	assertToDecimal(t, 1, 4, "0.0001")
	assertToDecimal(t, 10007, 4, "1.0007")
	assertToDecimal(t, 10000000070, 4, "1000000.0070")
	assertToDecimal(t, 0, 4, "0")

}
func assertConvert(t *testing.T, num string, decimal int, exp uint64) {
	res, err := utils.DecimalToUint64(num, uint8(decimal))
	if err != nil {
		if exp != 0 {
			t.Fatal(err)
		}
		return
	}
	if res != exp {
		t.Fatal("un expected value,exp=", exp, ",real=", res, "params:", num, decimal)
	}
}

func assertToDecimal(t *testing.T, v uint64, decimal int, exp string) {
	s := utils.Uint64ToDecimal(v, uint8(decimal))
	if s != exp {
		t.Fatal("to decimal error:", v, decimal, "exp=", exp, "real=", s)
	}
}
