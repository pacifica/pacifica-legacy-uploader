package common

import (
	"testing"
)

// Tests Encoder based upon base64 from the webssite
// http://www.motobit.com/util/base64-decoder-encoder.asp?charset=utf-8&acharset=
func Test_HelloWorldEncoded(t *testing.T) {
	var expected = "SGVsbG8gd29ybGQgd2hhdCBoYXZlIHlvdSBkb25lIQ=="
	var input = "Hello world what have you done!"

	var result = FsEncodeUser(input)

	if result != expected {
		t.Error("expected (%v)\nbut got (%v)", expected, result)
	}
}

// Tests Decoder based upon base64 from the webssite
// http://www.motobit.com/util/base64-decoder-encoder.asp?charset=utf-8&acharset=
func Test_HelloWorldDecoded(t *testing.T) {
	var expected = "Hello world what have you done!"
	var input = "SGVsbG8gd29ybGQgd2hhdCBoYXZlIHlvdSBkb25lIQ=="

	var result = FsDecodeUser(input)

	if result != expected {
		t.Error("expected (%v)\nbut got (%v)", expected, result)
	}
}

// Tests Encoder and Decoder based upon base64 from the webssite
// http://www.motobit.com/util/base64-decoder-encoder.asp?charset=utf-8&acharset=
func Test_EncodeDecodePair(t *testing.T) {
	var first = "23%23 5~!~@#!62 934258~* Abc_PQ"
	var second = "MjMlMjMgNX4hfkAjITYyIDkzNDI1OH4qIEFiY19QUQ=="

	var result = FsEncodeUser(first)

	if result != second {
		t.Error("FsEncodeUser :: expected (%v)\ngot (%v)", second, result)
	}

	result = FsDecodeUser(second)

	if result != first {
		t.Error("FsDecodeUser :: expected (%v)\ngot (%v)", first, result)
	}
}
