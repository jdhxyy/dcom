// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 发送模块
// Authors: jdh99 <jdh821@163.com>

package dcom

// gSend 发送数据
func gSend(protocol int, port int, dstIA uint64, frame *tFrame) {
	if frame == nil || gParam.IsAllowSend(port) == false {
		return
	}
	gParam.Send(protocol, port, dstIA, gFrameToBytes(frame))
}

// gBlockSend 块传输发送数据
func gBlockSend(protocol int, port int, dstIA uint64, frame *tBlockFrame) {
	if frame == nil || gParam.IsAllowSend(port) == false {
		return
	}
	gParam.Send(protocol, port, dstIA, gBlockFrameToBytes(frame))
}

// gSendRstFrame 发送错误码
func gSendRstFrame(protocol int, port int, dstIA uint64, errorCode ErrorCode, rid int, token int) {
	var frame tFrame
	frame.controlWord.code = gCodeRst
	frame.controlWord.blockFlag = 0
	frame.controlWord.rid = rid
	frame.controlWord.token = token
	frame.controlWord.payloadLen = 1
	frame.payload = make([]uint8, 1)
	frame.payload[0] = uint8(errorCode) | 0x80
	gSend(protocol, port, dstIA, &frame)
}
