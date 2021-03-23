// Copyright 2021-2021 The jdh99 Authors. All rights reserved.
// 等待队列
// Authors: jdh99 <jdh821@163.com>

package dcom

import (
	"container/list"
	"sync"
	"time"
)

// Resp 异步调用应答
type Resp struct {
	Error int
	Bytes []uint8
	Done  chan *Resp
}

// done 结果返回.框架内调用
func (resp *Resp) done() {
	select {
	case resp.Done <- resp:
	default:
	}
}

type tWaitItem struct {
	resp *Resp
	end  chan bool

	protocol  int
	pipe      uint64
	timeoutUs int64
	req       []uint8

	dstIA uint64
	rid   int
	token int

	// 启动时间.单位:us.用于判断是否超过总超时
	startTime int64
	// 上次发送时间戳.单位:us.用于重传
	lastRetryTimestamp int64
	retryNum           int
	code               int
}

var waitItems list.List
var waitItemsMutex sync.Mutex

func init() {
	go checkWaitItemsThread()
}

// checkWaitItems 检查等待列表线程
// 检查项有重发,超时等
func checkWaitItemsThread() {
	for {
		checkWaitItems()
		time.Sleep(time.Millisecond)
	}
}

func checkWaitItems() {
	waitItemsMutex.Lock()
	defer waitItemsMutex.Unlock()

	node := waitItems.Front()
	var nodeNext *list.Element
	for {
		if node == nil {
			break
		}
		nodeNext = node.Next()
		retrySend(node)
		node = nodeNext
	}
}

func retrySend(node *list.Element) {
	item := node.Value.(*tWaitItem)
	t := gGetTime()
	if t-item.startTime > item.timeoutUs {
		logWarn("wait ack timeout!task failed!token:%d", item.token)
		waitItems.Remove(node)
		if len(item.req) >= gSingleFrameSizeMax {
			gBlockRemove(item.protocol, item.pipe, item.dstIA, item.code, item.rid, item.token)
		}
		item.resp.Error = SystemErrorRxTimeout
		item.end <- true
		return
	}

	// 块传输不用此处重传.块传输模块自己负责
	if len(item.req) >= gSingleFrameSizeMax {
		return
	}

	if t-item.lastRetryTimestamp < int64(gParam.BlockRetryInterval*1000) {
		return
	}

	// 重传
	item.retryNum++
	if item.retryNum >= gParam.BlockRetryMaxNum {
		logWarn("retry too many!task failed!token:%d", item.token)
		waitItems.Remove(node)
		item.resp.Error = SystemErrorRxTimeout
		item.end <- true
		return
	}
	item.lastRetryTimestamp = t
	logWarn("retry send.token:%d retry num:%d", item.token, item.retryNum)
	waitlistSendFrame(item.protocol, item.pipe, item.dstIA, item.code, item.rid, item.token, item.req)
}

// Call RPC同步调用
// timeout是超时时间,单位:ms.为0表示不需要应答
// 返回值是应答字节流和错误码.错误码非SystemOK表示调用失败
func Call(protocol int, pipe uint64, dstIA uint64, rid int, timeout int, req []uint8) ([]uint8, int) {
	logInfo("call.protocol:%d pipe:0x%x dst ia:0x%x rid:%d timeout:%d", protocol, pipe, dstIA, rid, timeout)
	resp := CallAsync(protocol, pipe, dstIA, rid, timeout, req)
	<-resp.Done
	logInfo("call resp.result:%d len:%d", resp.Error, len(resp.Bytes))
	return resp.Bytes, resp.Error
}

// CallAsync RPC异步调用
// timeout是超时时间,单位:ms.为0表示不需要应答
// 返回值中错误码非SystemOK表示调用失败
func CallAsync(protocol int, pipe uint64, dstIA uint64, rid int, timeout int, req []uint8) *Resp {
	var resp Resp
	resp.Done = make(chan *Resp, 10)

	code := gCodeCon
	if timeout == 0 {
		code = gCodeNon
	}

	token := gGetToken()
	logInfo("call async.token:%d protocol:%d pipe:0x%x dst ia:0x%x rid:%d timeout:%d", token, protocol, pipe,
		dstIA, rid, timeout)
	waitlistSendFrame(protocol, pipe, dstIA, code, rid, token, req)

	if code == gCodeNon {
		resp.Error = SystemOK
		go func() {
			select {
			case <-time.After(time.Millisecond):
				resp.done()
			}
		}()
		return &resp
	}

	var item tWaitItem
	item.resp = &resp
	item.end = make(chan bool)
	item.pipe = pipe
	item.timeoutUs = int64(timeout) * 1000
	item.req = req

	item.dstIA = dstIA
	item.rid = rid
	item.token = token
	item.code = code

	item.retryNum = 0
	item.startTime = gGetTime()
	item.lastRetryTimestamp = gGetTime()
	waitItems.PushBack(&item)

	// 等待数据
	go func() {
		select {
		case <-item.end:
			item.resp.done()
		}
	}()
	return &resp
}

func waitlistSendFrame(protocol int, pipe uint64, dstIA uint64, code int, rid int, token int, data []uint8) {
	if len(data) >= gSingleFrameSizeMax {
		gBlockTx(protocol, pipe, dstIA, code, rid, token, data)
		return
	}

	var frame tFrame
	frame.controlWord.code = code
	frame.controlWord.blockFlag = 0
	frame.controlWord.rid = rid
	frame.controlWord.token = token
	frame.controlWord.payloadLen = len(data)
	frame.payload = append(frame.payload, data...)
	logInfo("send frame.token:%d", token)
	gSend(protocol, pipe, dstIA, &frame)
}

// gRxAckFrame 接收到ACK帧时处理函数
func gRxAckFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	waitItemsMutex.Lock()
	defer waitItemsMutex.Unlock()

	logInfo("rx ack frame.src ia:0x%x", srcIA)
	node := waitItems.Front()
	var nodeNext *list.Element
	for {
		if node == nil {
			break
		}
		nodeNext = node.Next()
		if checkNodeAndDealAckFrame(protocol, pipe, srcIA, frame, node) {
			break
		}
		node = nodeNext
	}
}

func checkNodeAndDealAckFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame, node *list.Element) bool {
	item := node.Value.(*tWaitItem)
	if item.protocol != protocol || item.pipe != pipe || item.dstIA != srcIA || item.rid != frame.controlWord.rid ||
		item.token != frame.controlWord.token {
		return false
	}

	logInfo("deal ack frame.token:%d", item.token)
	waitItems.Remove(node)
	item.resp.Bytes = append(item.resp.Bytes, frame.payload...)
	item.resp.Error = SystemOK
	item.end <- true
	return true
}

// gRxRstFrame 接收到RST帧时处理函数
func gRxRstFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame) {
	waitItemsMutex.Lock()
	defer waitItemsMutex.Unlock()

	logWarn("rx rst frame.src ia:0x%x", srcIA)
	node := waitItems.Front()
	var nodeNext *list.Element
	for {
		if node == nil {
			break
		}
		nodeNext = node.Next()
		if dealRstFrame(protocol, pipe, srcIA, frame, node) {
			break
		}
		node = nodeNext
	}
}

// dealRstFrame 处理复位连接帧
// 返回true表示节点符合条件
func dealRstFrame(protocol int, pipe uint64, srcIA uint64, frame *tFrame, node *list.Element) bool {
	item := node.Value.(*tWaitItem)
	if item.protocol != protocol || item.pipe != pipe || item.dstIA != srcIA || item.rid != frame.controlWord.rid ||
		item.token != frame.controlWord.token {
		return false
	}
	err := int(frame.payload[0])
	logWarn("deal rst frame.token:%d result:0x%x", item.token, err)
	waitItems.Remove(node)
	item.resp.Error = err
	item.end <- true
	return true
}
