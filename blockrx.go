// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 块传输接收模块
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
	"container/list"
	"github.com/jdhxyy/crc16"
	"sync"
	"time"
)

// tBlockRecvFunc 块传输接收函数类型
// 注意载荷实际长度不是frame载荷长度字段
type tBlockRecvFunc func(protocol int, pipe uint64, srcIA uint64, frame *tFrame)

type tBlockRxItem struct {
	protocol    int
	pipe        uint64
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
	now := gGetTime()
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
				logWarn("block rx send back retry num too many!token:%d", item.frame.controlWord.token)
				blockRxItems.Remove(node)
				break
			}
			// 超时重发
			if gParam.IsAllowSend(item.pipe) == false {
				break
			}
			logWarn("block rx send back retry num:%d token:%d", item.retryNums, item.frame.controlWord.token)
			sendBackFrame(item)
			break
		}

		node = nodeNext
	}
}

func sendBackFrame(item *tBlockRxItem) {
	logInfo("block rx send back frame.token:%d offset:%d", item.frame.controlWord.token, item.blockHeader.offset)
	var frame tFrame
	frame.controlWord.code = gCodeBack
	frame.controlWord.blockFlag = 0
	frame.controlWord.rid = item.frame.controlWord.rid
	frame.controlWord.token = item.frame.controlWord.token
	frame.controlWord.payloadLen = 2
	frame.payload = make([]uint8, 2)
	frame.payload[0] = uint8(item.blockHeader.offset >> 8)
	frame.payload[1] = uint8(item.blockHeader.offset)
	gSend(item.protocol, item.pipe, item.srcIA, &frame)

	item.retryNums++
	item.lastTxTime = gGetTime()
}

// gBlockRxSetCallback 设置接收回调函数
func gBlockRxSetCallback(recvFunc tBlockRecvFunc) {
	blockRecv = recvFunc
}

// gBlockRxReceive 块传输接收数据
func gBlockRxReceive(protocol int, pipe uint64, srcIA uint64, frame *tBlockFrame) {
	blockRxItemsMutex.Lock()
	defer blockRxItemsMutex.Unlock()

	logInfo("block rx receive.token:%d src_ia:0x%x", frame.controlWord.token, srcIA)
	node := getNodeBlockRxItems(protocol, pipe, srcIA, frame)
	if node == nil {
		createAndAppendNodeBlockRxItems(protocol, pipe, srcIA, frame)
	} else {
		editNodeBlockRxItems(protocol, pipe, node, frame)
	}
}

func getNodeBlockRxItems(protocol int, pipe uint64, srcIA uint64, frame *tBlockFrame) *list.Element {
	node := blockRxItems.Front()
	var item *tBlockRxItem

	for {
		if node == nil {
			break
		}
		item = node.Value.(*tBlockRxItem)
		if item.protocol == protocol && item.pipe == pipe && item.srcIA == srcIA &&
			item.frame.controlWord.token == frame.controlWord.token &&
			item.frame.controlWord.rid == frame.controlWord.rid &&
			item.frame.controlWord.code == frame.controlWord.code {
			return node
		}
		node = node.Next()
	}
	return nil
}

func createAndAppendNodeBlockRxItems(protocol int, pipe uint64, srcIA uint64, frame *tBlockFrame) {
	if frame.blockHeader.offset != 0 {
		logWarn("block rx create and append item failed!offset is not 0:%d.token:%d send rst",
			frame.blockHeader.offset, frame.controlWord.token)
		gSendRstFrame(protocol, pipe, srcIA, SystemErrorWrongBlockOffset, frame.controlWord.rid,
			frame.controlWord.token)
		return
	}

	var item tBlockRxItem
	item.pipe = pipe
	item.srcIA = srcIA
	item.frame.controlWord = frame.controlWord
	item.blockHeader = frame.blockHeader
	item.frame.payload = append(item.frame.payload, frame.payload...)
	item.blockHeader.offset = len(frame.payload)
	blockRxItems.PushBack(&item)
	sendBackFrame(&item)
}

func editNodeBlockRxItems(protocol int, pipe uint64, node *list.Element, frame *tBlockFrame) {
	item := node.Value.(*tBlockRxItem)
	if item.blockHeader.offset != frame.blockHeader.offset || item.protocol != protocol || item.pipe != pipe {
		logWarn("block rx edit item failed!token:%d.item<->frame:offset:%d %d,protocol:%d %d,pipe:%d %d",
			frame.controlWord.token, item.blockHeader.offset, frame.blockHeader.offset, item.protocol, protocol,
			item.pipe, pipe)
		return
	}

	item.frame.payload = append(item.frame.payload, frame.payload...)
	item.blockHeader.offset += len(frame.payload)

	item.retryNums = 0
	sendBackFrame(item)

	if item.blockHeader.offset >= item.blockHeader.total {
		logInfo("block rx receive end.token:%d", item.frame.controlWord.token)
		crcCalc := crc16.Checksum(item.frame.payload)
		if crcCalc != item.blockHeader.crc16 {
			logWarn("block rx crc is wrong.token:%d crc calc:0x%x get:0x%x", item.frame.controlWord.token, crcCalc,
				item.blockHeader.crc16)
			blockRxItems.Remove(node)
			return
		}
		if blockRecv != nil {
			blockRecv(item.protocol, item.pipe, item.srcIA, &item.frame)
		}
		blockRxItems.Remove(node)
	}
}

// gBlockRxDealRstFrame 块传输接收模块处理复位连接帧
func gBlockRxDealRstFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	node := blockRxItems.Front()
	var item *tBlockRxItem

	for {
		if node == nil {
			break
		}
		item = node.Value.(*tBlockRxItem)
		if item.protocol == protocol && item.pipe == pipe && item.srcIA == srcIA &&
			item.frame.controlWord.token == frame.controlWord.token &&
			item.frame.controlWord.rid == frame.controlWord.rid {
			logWarn("block rx rst.token:%d", item.frame.controlWord.token)
			blockRxItems.Remove(node)
			return
		}
		node = node.Next()
	}
}
