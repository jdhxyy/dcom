// Copyright 2021-2021 The TZIOT Authors. All rights reserved.
// 发送模块
// Authors: jdh99 <jdh821@163.com>

package dcom

// gSend 发送数据
func gSend(port int, dstIA uint64, frame *tFrame) {
    if frame == nil || gParam.IsAllowSend(port) == false {
        return
    }
    bytes := gFrameToBytes(frame)
    gParam.Send(port, dstIA, bytes)
}

// gBlockSend 块传输发送数据
func gBlockSend(port int, dstIA uint64, frame *tBlockFrame) {
    if frame == nil || gParam.IsAllowSend(port) == false {
        return
    }
    bytes := gBlockFrameToBytes(frame)
    gParam.Send(port, dstIA, bytes)
}

// gSendRstFrame 发送错误码
// controlWord 当前会话控制字
// 返回true时发送成功
func gSendRstFrame(port int, dstIA uint64, errorCode ErrorCode, rid int, token int) {
    var frame tFrame
    frame.controlWord.code = gCodeRst
    frame.controlWord.blockFlag = 0
    frame.controlWord.rid = rid
    frame.controlWord.token = token
    frame.controlWord.payloadLen = 1
    frame.payload = make([]uint8, 1)
    frame.payload[0] = uint8(errorCode) | 0x80
    gSend(port, dstIA, &frame)
}
