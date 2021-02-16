// Copyright 2021-2021 The TZIOT Authors. All rights reserved.
// 块传输发送模块
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
    "container/list"
    "gitee.com/jdhxyy/crc16"
    "sync"
    "time"
)

type tBlockTxItem struct {
    port  int
    dstIA uint64
    code  int
    rid   int
    token int

    // 第一帧需要重发控制
    isFirstFrame        bool
    firstFrameRetryTime int64
    firstFrameRetryNum  int

    lastRxAckTime int64

    crc16 uint16
    data  []uint8
}

var blockTxItems list.List
var blockTxItemsMutex sync.Mutex

// gThreadBlockTxRun 块传输发送模块运行线程
func gThreadBlockTxRun() {
    blockTxItemsMutex.Lock()

    node := blockTxItems.Front()
    var nextNode *list.Element
    for {
        if node == nil {
            break
        }
        nextNode = node.Next()
        checkTimeoutAndRetrySendFirstFrame(node)
        node = nextNode
    }

    blockTxItemsMutex.Unlock()

    time.Sleep(gInterval)
}

// checkTimeoutAndRetrySendFirstFrame 检查超时节点和重发首帧
func checkTimeoutAndRetrySendFirstFrame(node *list.Element) {
    item := node.Value.(*tBlockTxItem)
    now := time.Now().Unix()
    if item.isFirstFrame == false {
        // 非首帧
        if now-item.lastRxAckTime > int64(gParam.BlockRetryInterval*gParam.BlockRetryMaxNum*1000) {
            blockTxItems.Remove(node)
        }
        return
    }

    // 首帧处理
    if now-item.firstFrameRetryTime < int64(gParam.BlockRetryInterval*1000) {
        return
    }

    if item.firstFrameRetryNum > gParam.BlockRetryMaxNum {
        blockTxItems.Remove(node)
    } else {
        blockTxSendFrame(item, 0)
        item.firstFrameRetryNum++
        item.firstFrameRetryTime = now
    }
}

func blockTxSendFrame(item *tBlockTxItem, offset int) {
    delta := len(item.data) - offset
    payloadLen := gSingleFrameSizeMax - gBlockHeaderLen
    if payloadLen > delta {
        payloadLen = delta
    }

    var frame tBlockFrame
    frame.controlWord.code = item.code
    frame.controlWord.blockFlag = 1
    frame.controlWord.rid = item.rid
    frame.controlWord.token = item.token
    frame.controlWord.payloadLen = gBlockHeaderLen + payloadLen
    frame.blockHeader.crc16 = item.crc16
    frame.blockHeader.total = len(item.data)
    frame.blockHeader.offset = offset
    frame.payload = append(frame.payload, item.data[offset:offset+payloadLen]...)
    gBlockSend(item.port, item.dstIA, &frame)
}

// gBlockTx 块传输发送
func gBlockTx(port int, dstIA uint64, code int, rid int, token int, data []uint8) {
    if len(data) < gSingleFrameSizeMax {
        return
    }
    if blockTxIsNodeExist(port, dstIA, code, rid, token) {
        return
    }

    item := blockTxCreateItem(port, dstIA, code, rid, token, data)
    blockTxSendFrame(item, 0)
    item.firstFrameRetryNum++
    item.firstFrameRetryTime = time.Now().Unix()
    blockTxItems.PushBack(&item)
}

func blockTxIsNodeExist(port int, dstIA uint64, code int, rid int, token int) bool {
    node := blockTxItems.Front()
    var item *tBlockTxItem
    for {
        if node == nil {
            break
        }
        item = node.Value.(*tBlockTxItem)

        if item.port == port && item.dstIA == dstIA && item.code == code && item.rid == rid && item.token == token {
            return true
        }
        node = node.Next()
    }
    return false
}

func blockTxCreateItem(port int, dstIA uint64, code int, rid int, token int, data []uint8) *tBlockTxItem {
    var item tBlockTxItem
    item.port = port
    item.dstIA = dstIA
    item.code = code
    item.rid = rid
    item.token = token
    item.data = append(item.data, data...)
    item.crc16 = crc16.CheckSum(data)

    item.isFirstFrame = true
    item.firstFrameRetryNum = 0
    now := time.Now().Unix()
    item.firstFrameRetryTime = now
    item.lastRxAckTime = now
    return &item
}

// gBlockRxBackFrame 接收到BACK帧时处理函数
func gBlockRxBackFrame(port int, srcIA uint64, frame *tFrame) {
    if frame.controlWord.code != gCodeBack {
        return
    }

    node := blockTxItems.Front()
    var nextNode *list.Element
    for {
        if node == nil {
            break
        }
        nextNode = node.Next()
        if checkNodeAndDealBackFrame(port, srcIA, frame, node) {
            break
        }
        node = nextNode
    }
}

// checkNodeAndDealBackFrame 检查节点是否符合条件,符合则处理BACK帧
// 返回true表示节点符合条件
func checkNodeAndDealBackFrame(port int, srcIA uint64, frame *tFrame, node *list.Element) bool {
    item := node.Value.(*tBlockTxItem)

    if item.port != port || item.dstIA != srcIA || item.rid != frame.controlWord.rid ||
        item.token != frame.controlWord.token {
        return false
    }
    if frame.controlWord.payloadLen != 2 {
        return false
    }
    startOffset := (int(frame.payload[0]) << 8) + int(frame.payload[1])
    if startOffset >= len(item.data) {
        blockTxItems.Remove(node)
        return true
    }

    if item.isFirstFrame {
        item.isFirstFrame = false
    }
    item.lastRxAckTime = time.Now().Unix()

    blockTxSendFrame(item, startOffset)
    return true
}

// gBlockTxDealRstFrame 块传输发送模块处理复位连接帧
func gBlockTxDealRstFrame(port int, srcIA uint64, frame *tFrame) {
    node := blockTxItems.Front()
    var item *tBlockTxItem
    for {
        if node == nil {
            break
        }

        item = node.Value.(*tBlockTxItem)
        if item.port == port && item.dstIA == srcIA && item.rid == frame.controlWord.rid &&
            item.token == frame.controlWord.token {
            blockTxItems.Remove(node)
            return
        }

        node = node.Next()
    }
}
