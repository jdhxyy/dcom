// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 发送模块
// Authors: jdh99 <jdh821@163.com>

package dcom

// gSend 发送数据
func gSend(protocol int, pipe uint64, dstIA uint64, frame *tFrame) {
	if frame == nil {
		return
	}
	if gParam.IsAllowSend(pipe) == false {
		logWarn("send failed!pipe:0x%x is not allow send.token:%d", pipe, frame.controlWord.token)
		return
	}
	logInfo("send frame.token:%d protocol:%d pipe:0x%x dst ia:0x%x", frame.controlWord.token, protocol, pipe, dstIA)
	gParam.Send(protocol, pipe, dstIA, gFrameToBytes(frame))
}

// gBlockSend 块传输发送数据
func gBlockSend(protocol int, pipe uint64, dstIA uint64, frame *tBlockFrame) {
	if frame == nil {
		return
	}
	if gParam.IsAllowSend(pipe) == false {
		logWarn("block send failed!pipe:0x%x is not allow send.token:%d", pipe, frame.controlWord.token)
		return
	}
	logInfo("block send frame.token:%d protocol:%d pipe:0x%x dst ia:0x%x offset:%d", frame.controlWord.token,
		protocol, pipe, dstIA, frame.blockHeader.offset)
	gParam.Send(protocol, pipe, dstIA, gBlockFrameToBytes(frame))
}

// gSendRstFrame 发送错误码
func gSendRstFrame(protocol int, pipe uint64, dstIA uint64, errorCode int, rid int, token int) {
	logWarn("send rst frame!token:%d protocol:%d pipe:0x%x dst ia:0x%x", token, protocol, pipe, dstIA)
	var frame tFrame
	frame.controlWord.code = gCodeRst
	frame.controlWord.blockFlag = 0
	frame.controlWord.rid = rid
	frame.controlWord.token = token
	frame.controlWord.payloadLen = 1
	frame.payload = make([]uint8, 1)
	frame.payload[0] = uint8(errorCode) | 0x80
	gSend(protocol, pipe, dstIA, &frame)
}
