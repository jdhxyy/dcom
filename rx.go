// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 接收模块
// Authors: jdh99 <jdh821@163.com>

package dcom

// gRxLoad 模块载入
func gRxLoad() {
	gBlockRxSetCallback(dealRecv)
}

func dealRecv(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	if frame.controlWord.code == gCodeCon || frame.controlWord.code == gCodeNon {
		gRxCon(protocol, pipe, srcIA, frame)
		return
	}
	if frame.controlWord.code == gCodeAck {
		gRxAckFrame(protocol, pipe, srcIA, frame)
		return
	}
	if frame.controlWord.code == gCodeBack {
		gBlockRxBackFrame(protocol, pipe, srcIA, frame)
		return
	}
	if frame.controlWord.code == gCodeRst {
		if len(frame.payload) != 1 || frame.controlWord.payloadLen != 1 {
			return
		}
		gRxRstFrame(protocol, pipe, srcIA, frame)
		gBlockRxDealRstFrame(protocol, pipe, srcIA, frame)
		gBlockTxDealRstFrame(protocol, pipe, srcIA, frame)
		return
	}
}

// Receive 接收数据
// 应用模块接收到数据后需调用本函数
// 本函数接收帧的格式为DCOM协议数据
func Receive(protocol int, pipe uint64, srcIA uint64, bytes []uint8) {
	frame := gBytesToFrame(bytes)
	if frame == nil {
		return
	}

	if frame.controlWord.blockFlag == 0 {
		dealRecv(protocol, pipe, srcIA, frame)
	} else {
		blockFrame := gByetsToBlockFrame(bytes)
		if blockFrame == nil {
			return
		}
		gBlockRxReceive(protocol, pipe, srcIA, blockFrame)
	}
}
