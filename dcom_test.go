package dcom

import (
	"fmt"
	"testing"
)

func TestCase1(t *testing.T) {
	testLoad()
	arr := []uint8{1, 2, 3}
	resp, err := Call(0, 0, 0x1234, 1, 3000, arr)
	fmt.Print(err, resp)
}

func testLoad() {
	var param LoadParam
	param.BlockRetryMaxNum = 5
	param.BlockRetryInterval = 1000
	param.IsAllowSend = testIsAllowSend
	param.Send = testSend
	Load(&param)
}

func testIsAllowSend(port int) bool {
	return true
}

func testSend(protocol int, port int, dstIA uint64, bytes []uint8) {
	fmt.Printf("protocol:%d dstIA:%x, port:%d send:", protocol, dstIA, port)
	testPrintHex(bytes)
}

func testPrintHex(bytes []uint8) {
	for _, v := range bytes {
		fmt.Printf("0x%02x ", v)
	}
	fmt.Println()
}

func TestCase2(t *testing.T) {
	testLoad()
	arr := []uint8{1, 2, 3}
	resp := CallAsync(0, 0, 0x1234, 1, 3000, arr)
	<-resp.Done
	fmt.Print(resp)
}

func TestCase3(t *testing.T) {
	testLoad()
	arr := []uint8{1, 2, 3}
	resp, err := Call(0, 0, 0x1234, 1, 0, arr)
	fmt.Print(err, resp)
}

func TestCase4(t *testing.T) {
	testLoad()
	arr := []uint8{1, 2, 3}
	resp := CallAsync(1, 0, 0x1234, 1, 0, arr)
	<-resp.Done
	fmt.Print(resp)
}

func TestCase5(t *testing.T) {
	testLoad()
	arr := make([]uint8, 501)
	for i := 0; i < 501; i++ {
		arr[i] = uint8(i)
	}
	resp, err := Call(0, 0, 0x1234, 1, 0, arr)
	fmt.Print(err, resp)
}

func TestCase6(t *testing.T) {
	testLoad()
	arr := make([]uint8, 501)
	for i := 0; i < 501; i++ {
		arr[i] = uint8(i)
	}
	resp, err := Call(0, 0, 0x1234, 1, 3000, arr)
	fmt.Print(err, resp)
}

func TestCase7(t *testing.T) {
	testLoad1()
	arr := []uint8{1, 2, 3}
	resp, err := Call(0, 0, 0x1234, 1, 3000, arr)
	fmt.Print(err, resp)
}

func testLoad1() {
	var param LoadParam
	param.BlockRetryMaxNum = 5
	param.BlockRetryInterval = 1000
	param.IsAllowSend = testIsAllowSend
	param.Send = testSend1
	Load(&param)
}

func testSend1(protocol int, port int, dstIA uint64, bytes []uint8) {
	fmt.Printf("dstIA:%x, port:%d send:", dstIA, port)
	testPrintHex(bytes)

	var arr []uint8
	arr = append(arr, 0x40)
	arr = append(arr, 0x04)
	arr = append(arr, 0x01)
	arr = append(arr, 0x05)
	arr = append(arr, 0x1)
	arr = append(arr, 0x2)
	arr = append(arr, 0x3)
	arr = append(arr, 0x4)
	arr = append(arr, 0x5)
	Receive(protocol, port, dstIA, arr)
}
