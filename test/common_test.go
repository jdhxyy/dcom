package dcom

import (
    "dcom"
    "fmt"
    "testing"
)

func TestControlWordToBytes(t *testing.T) {
    var word dcom.tControlWord
    word.Code = 1
    word.BlockFlag = 1
    word.Rid = 2
    word.Token = 3
    word.PayloadLen = 4
    bytes := dcom.gControlWordToBytes(&word)
    fmt.Println(word)
    for _,v := range bytes {
        fmt.Printf("0x%02x ", v)
    }
    fmt.Println()
}

func TestFrameToBytes(t *testing.T) {
    var frame dcom.tFrame
    frame.Word.Code = 1
    frame.Word.BlockFlag = 1
    frame.Word.Rid = 2
    frame.Word.Token = 3
    frame.Word.PayloadLen = 4
    frame.Payload = make([]uint8, 10)
    for i := 0; i < 10; i++ {
        frame.Payload[i] = uint8(i)
    }

    bytes := dcom.gFrameToBytes(&frame)
    for _,v := range bytes {
        fmt.Printf("0x%02x ", v)
    }
    fmt.Println()
}
