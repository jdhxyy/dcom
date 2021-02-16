// Copyright 2021-2021 The TZIOT Authors. All rights reserved.
// 块传输接收模块
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
    "container/list"
    "gitee.com/jdhxyy/crc16"
    "gitee.com/jdhxyy/lagan"
    "sync"
    "time"
)

// tBlockRecvFunc 块传输接收函数类型
// 注意载荷实际长度不是frame载荷长度字段
type tBlockRecvFunc func(port int, srcIA uint64, frame *tFrame)

type tBlockRxItem struct {
    port        int
    srcIA       uint64
    frame       tFrame
    blockHeader tBlocHeader
    // 上次发送时间
    lastTxTime int64
    retryNums  int
}

var blockRxItems list.List
var blockRxItemsMutex sync.Mutex

var blockRecv tBlockRecvFunc

// gThreadBlockRxRun 块传输接收模块运行线程
func gThreadBlockRxRun() {
    for {
        blockRxItemsMutex.Lock()
        sendAllBackFrame()
        blockRxItemsMutex.Unlock()

        time.Sleep(gInterval)
    }
}

func sendAllBackFrame() {
    now := time.Now().Unix()
    interval := int64(gParam.BlockRetryInterval) * 1000

    node := blockRxItems.Front()
    var nodeNext *list.Element
    var item *tBlockRxItem
    for {
        if node == nil {
            break
        }
        nodeNext = node.Next()

        for {
            item = node.Value.(*tBlockRxItem)
            if now-item.lastTxTime < interval {
                break
            }
            if item.retryNums > gParam.BlockRetryMaxNum {
                lagan.Debug(Tag, "remove block node because retry num too many.0x%x rid:%d token:%d", item.srcIA,
                    item.frame.controlWord.rid, item.frame.controlWord.token)
                blockRxItems.Remove(node)
                break
            }
            // 超时重发
            if gParam.IsAllowSend(item.port) == false {
                break
            }
            sendBackFrame(item)
            break
        }

        node = nodeNext
    }
}

func sendBackFrame(item *tBlockRxItem) {
    var frame tFrame
    frame.controlWord.code = gCodeBack
    frame.controlWord.blockFlag = 0
    frame.controlWord.rid = item.frame.controlWord.rid
    frame.controlWord.token = item.frame.controlWord.token
    frame.controlWord.payloadLen = 2
    frame.payload[0] = uint8(item.blockHeader.offset >> 8)
    frame.payload[1] = uint8(item.blockHeader.offset)
    gSend(item.port, item.srcIA, &frame)

    item.retryNums++
    item.lastTxTime = time.Now().Unix()
}

// gBlockRxSetCallback 设置接收回调函数
func gBlockRxSetCallback(recvFunc tBlockRecvFunc) {
    blockRecv = recvFunc
}

// gBlockRxReceive 块传输接收数据
func gBlockRxReceive(port int, srcIA uint64, frame *tBlockFrame) {
    blockRxItemsMutex.Lock()
    defer blockRxItemsMutex.Unlock()

    node := getNodeBlockRxItems(port, srcIA, frame)
    if node == nil {
        createAndAppendNodeBlockRxItems(port, srcIA, frame)
    } else {
        editNodeBlockRxItems(port, node, frame)
    }
}

func getNodeBlockRxItems(port int, srcIA uint64, frame *tBlockFrame) *list.Element {
    node := blockRxItems.Front()
    var item *tBlockRxItem

    for {
        if node == nil {
            break
        }
        item = node.Value.(*tBlockRxItem)
        if item.port == port && item.srcIA == srcIA &&
            item.frame.controlWord.token == frame.controlWord.token &&
            item.frame.controlWord.rid == frame.controlWord.rid &&
            item.frame.controlWord.code == frame.controlWord.code {
            return node
        }
        node = node.Next()
    }
    return nil
}

func createAndAppendNodeBlockRxItems(port int, srcIA uint64, frame *tBlockFrame) {
    if frame.blockHeader.offset != 0 {
        gSendRstFrame(port, srcIA, SystemErrorWrongBlockOffset, frame.controlWord.rid, frame.controlWord.token)
        return
    }

    var item tBlockRxItem
    item.port = port
    item.srcIA = srcIA
    item.frame.controlWord = frame.controlWord
    item.blockHeader = frame.blockHeader
    item.frame.payload = append(item.frame.payload, frame.payload...)
    item.blockHeader.offset = len(frame.payload)
    blockRxItems.PushBack(&item)
    sendBackFrame(&item)
}

func editNodeBlockRxItems(port int, node *list.Element, frame *tBlockFrame) {
    item := node.Value.(*tBlockRxItem)
    if item.blockHeader.offset != frame.blockHeader.offset || item.port != port {
        return
    }

    item.frame.payload = append(item.frame.payload, frame.payload...)
    item.blockHeader.offset += len(frame.payload)

    item.retryNums = 0
    sendBackFrame(item)

    if item.blockHeader.offset >= item.blockHeader.total {
        crcCalc := crc16.CheckSum(item.frame.payload)
        if crcCalc != item.blockHeader.crc16 {
            blockRxItems.Remove(node)
            return
        }
        if blockRecv != nil {
            blockRecv(item.port, item.srcIA, &item.frame)
        }
        blockRxItems.Remove(node)
    }
}

// gBlockRxDealRstFrame 块传输接收模块处理复位连接帧
func gBlockRxDealRstFrame(port int, srcIA uint64, frame *tFrame) {
    node := blockRxItems.Front()
    var item *tBlockRxItem

    for {
        if node == nil {
            break
        }
        item = node.Value.(*tBlockRxItem)
        if item.port == port && item.srcIA == srcIA &&
            item.frame.controlWord.token == frame.controlWord.token &&
            item.frame.controlWord.rid == frame.controlWord.rid {
            blockRxItems.Remove(node)
            return
        }
        node = node.Next()
    }
}
