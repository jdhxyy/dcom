// Copyright 2021-2021 The TZIOT Authors. All rights reserved.
// 接收模块
// Authors: jdh99 <jdh821@163.com>

package dcom

// gRxLoad 模块载入
func gRxLoad() {
    gBlockRxSetCallback(dealRecv)
}

func dealRecv(port int, srcIA uint64, frame *tFrame) {
    if frame.controlWord.code == gCodeCon || frame.controlWord.code == gCodeNon {
        gRxCon(port, srcIA, frame)
        return
    }
    if frame.controlWord.code == gCodeAck {
        gRxAckFrame(port, srcIA, frame)
        return
    }
    if frame.controlWord.code == gCodeBack {
        gBlockRxBackFrame(port, srcIA, frame)
        return
    }
    if frame.controlWord.code == gCodeRst {
        if len(frame.payload) != 1 || frame.controlWord.payloadLen != 1 {
            return
        }
        gRxRstFrame(port, srcIA, frame)
        gBlockRxDealRstFrame(port, srcIA, frame)
        gBlockTxDealRstFrame(port, srcIA, frame)
        return
    }
}

// Receive 接收数据
// 应用模块接收到数据后需调用本函数
// 本函数接收帧的格式为DCOM协议数据
func Receive(port int, srcIA uint64, bytes []uint8) {
    frame := gByetsToFrame(bytes)
    if frame == nil {
        return
    }

    if frame.controlWord.blockFlag == 0 {
        dealRecv(port, srcIA, frame)
    } else {
        blockFrame := gByetsToBlockFrame(bytes)
        gBlockRxReceive(port, srcIA, blockFrame)
    }
}
