// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 接收到连接时处理
// Authors: jdh99 <jdh821@163.com>

package dcom

// gRxCon 接收到连接帧时处理函数
func gRxCon(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	logInfo("rx con.token:%d", frame.controlWord.token)
	resp, err := gCallback(protocol, pipe, srcIA, frame.controlWord.rid, frame.payload)

	// NON不需要应答
	if frame.controlWord.code == gCodeNon {
		return
	}

	if err != SystemOK {
		logInfo("service send err:0x%x token:%d", err, frame.controlWord.token)
		gSendRstFrame(protocol, pipe, srcIA, err, frame.controlWord.rid, frame.controlWord.token)
		return
	}

	if len(resp) > gSingleFrameSizeMax {
		// 长度过长启动块传输
		logInfo("service send too long:%d.start block tx.token:%d", len(resp), frame.controlWord.token)
		gBlockTx(protocol, pipe, srcIA, gCodeAck, frame.controlWord.rid, frame.controlWord.token, resp)
		return
	}

	var ackFrame tFrame
	ackFrame.controlWord.code = gCodeAck
	ackFrame.controlWord.blockFlag = 0
	ackFrame.controlWord.rid = frame.controlWord.rid
	ackFrame.controlWord.token = frame.controlWord.token
	ackFrame.controlWord.payloadLen = len(resp)
	ackFrame.payload = append(ackFrame.payload, resp...)
	gSend(protocol, pipe, srcIA, &ackFrame)
}
