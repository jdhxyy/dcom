# 海萤物联网教程：物联网RPC框架Go DCOM

欢迎前往社区交流：[海萤物联网社区](http://www.ztziot.com)

## 简介
RPC：Remote Procedure Call，远程过程调用。使用RPC可以让一台计算机的程序程调用另一台计算机的上的程序。

RPC通过把网络通讯抽象为远程的过程调用，调用远程的过程就像调用本地的子程序一样方便，从而屏蔽了通讯复杂性，使开发人员可以无需关注网络编程的细节，将更多的时间和精力放在业务逻辑本身的实现上，提高工作效率。

DCOM：Device Communication Protocol(DCOM)，设备间通信协议。DCOM是针对物联网使用场景开发的RPC框架，主要有如下特点：

- 协议开销极小仅4字节。物联网很多场景都是几十字节的短帧，RPC协议本身的开销如果过大会导致在这些场景无法应用
- 可以跨语言通信。DCOM协议在设计上与语言无关，无论C，Golang，Python等都可以使用DCOM
- 可以跨通信介质通信。DCOM协议可以工作在以太网，串口，wifi，小无线等一切通信介质之上

在海萤物联网中，节点间使用DCOM来通信。本文介绍Go语言开发的DCOM包的使用方法。

## 开源
- [github上的项目地址](https://github.com/jdhxyy/dcom)
- [gitee上的项目地址](https://gitee.com/jdhxyy/dcom)

## 安装
推荐使用go mod：github.com/jdhxyy/dcom

安装好后在项目中即可导入使用：
```go
import "https://github.com/jdhxyy/dcom"
```

## 基础概念
### 资源ID
资源ID又称服务号：Resource ID，简称RID。节点使用服务号开放自己的能力或者资源。服务号取值1-1023共1023个数。

比如某个节点是智能插座，开放两个能力：
- 插座开关
- 插座当前状态

则可对外提供两个服务：

服务号|说明
---|---
1|插座开关
2|插座当前状态

### 协议号
协议号protocol，每个协议可以绑定一组资源ID，不同协议下的资源ID可以重复。

### 管道
管道：pipe。DCOM通信时数据流可以绑定到某个管道上，DCOM支持多个管道同时通信。管道可以是UDP端口、TCP端口、串口、4G、无线等，每个管道都有独立的发送和接收回调。管道号不能为0，是64位数。

## 通信模型
DCOM主要有两种通信模型，有应答通信，无应答通信。

常用的是有应答的通信，如果没有收到应答，DCOM在通信时会尝试重传。每次使用Call函数与目的节点通信，都是一个新的会话，每个会话都有唯一的token值来保证数据不会串扰。


## API
```go
// Load 模块载入
func Load(param *LoadParam)

// Receive 接收数据
// 应用模块接收到数据后需调用本函数
// 本函数接收帧的格式为DCOM协议数据
func Receive(protocol int, pipe uint64, srcIA uint64, bytes []uint8)

// Register 注册服务回调函数
func Register(protocol int, rid int, callback CallbackFunc)

// Call RPC同步调用
// timeout是超时时间,单位:ms.为0表示不需要应答
// 返回值是应答字节流和错误码.错误码非SystemOK表示调用失败
func Call(protocol int, pipe uint64, dstIA uint64, rid int, timeout int, req []uint8) ([]uint8, int)
```

- 辅助函数
```go
// StructToBytes 结构体转字节流
func StructToBytes(s interface{}) ([]uint8, error)

// BytesToStruct 字节流转结构体 结构体中的元素首字母要求大写
// s是结构体指针,保存转换后的结构体
func BytesToStruct(data []uint8, s interface{}) error
```

- 数据结构
```go
// LoadParam 载入参数
type LoadParam struct {
	// 块传输帧重试间隔.单位:ms
	BlockRetryInterval int
	// 块传输帧重试最大次数
	BlockRetryMaxNum int

	// API接口
	// 是否允许发送
	IsAllowSend IsAllowSendFuncByPipeFunc
	// 发送的是DCOM协议数据
	Send SendByPipeFunc
}

// IsAllowSendFuncByPipeFunc 某管道是否允许发送函数类型
type IsAllowSendFuncByPipeFunc func(pipe uint64) bool

// SendByPipeFunc 向指定端口发送函数类型
type SendByPipeFunc func(protocol int, pipe uint64, dstIA uint64, bytes []uint8)

// CallbackFunc 注册DCOM服务回调函数
// 返回值是应答和错误码.错误码为0表示回调成功,否则是错误码
type CallbackFunc func(pipe uint64, srcIA uint64, req []uint8) ([]uint8, int)
```

- 系统错误码
```go
const (
	// 正确值
	SystemOK = 0
	// 接收超时
	SystemErrorRxTimeout = 0x10
	// 发送超时
	SystemErrorTxTimeout = 0x11
	// 内存不足
	SystemErrorNotEnoughMemory = 0x12
	// 没有对应的资源ID
	SystemErrorInvalidRid = 0x13
	// 块传输校验错误
	SystemErrorWrongBlockCheck = 0x14
	// 块传输偏移地址错误
	SystemErrorWrongBlockOffset = 0x15
	// 参数错误
	SystemErrorParamInvalid = 0x16
)
```

### Load：模块载入
在使用DCOM前必须要初始化。DCOM支持重传，所以在初始化时需输入重传间隔以及重传最大次数。

DCOM与通信介质无关，不同介质可定义不同的管道号。应用程序需要在是否允许发送函数（IsAllowSend ），以及发送函数（Send）中编写不同管道的操作。

- 示例：某节点有两个管道
```go
func main() {
	var param dcom.LoadParam
	param.BlockRetryMaxNum = 5
	param.BlockRetryInterval = 1000
	param.IsAllowSend = isAllowSend
	param.Send = send
	dcom.Load(&param)
}

func isAllowSend(pipe uin64) bool {
	if pipe == 1 {
		return isPipe1AllowSend()
	} else {
		return isPipe2AllowSend()
	}
}

func send(protocol int, pipe uint64, dstIA uint64, data []uint8) {
	if pipe == 1 {
		pipe1Send(data)
	} else {
		pipe2Send(data)
	}
}
```

protocol，dstIA等字段根据需求处理。

### Receive 接收数据
应用程序接收到数据需要调用Receive函数，将数据发送给DCOM。

- 示例：某节点有两个管道都可接收
```go
func pipe1Receive(data []uint8) {
	dcom.Receive(0, 1, 0x2140000000000101, data)
}

func pipe2Receive(data []uint8) {
	dcom.Receive(0, 2, 0x2140000000000101, data)
}
```

协议号protocol示例中填写的是0，应用时根据实际场景填写。

### Register：服务注册
节点可以通过注册服务开放自身的能力。

- 示例：假设节点2140::101是智能插座，提供控制和读取开关状态两个服务：

```go
dcom.Register(0, 1, controlService)
dcom.Register(0, 2, getStateService)

// controlService 控制开关服务
// 返回值是应答和错误码.错误码为0表示回调成功,否则是错误码
func controlService(pipe uint64, srcIA uint64, req []uint8) ([]uint8, int) {
	if req[0] == 0 {
		off()
	} else {
		on()
	}
	return nil, dcom.SystemOK
}

// getStateService 读取开关状态服务
// 返回值是应答和错误码.错误码为0表示回调成功,否则是错误码
func getStateService(pipe uint64, srcIA uint64, req []uint8) ([]uint8, int) {
	return []uint8{state()}, dcom.SystemOK
}
```

### Call：同步调用
```go
// Call RPC同步调用
// timeout是超时时间,单位:ms.为0表示不需要应答
// 返回值是应答字节流和错误码.错误码非SystemOK表示调用失败
func Call(protocol int, pipe uint64, dstIA uint64, rid int, timeout int, req []uint8) ([]uint8, int)
```

同步调用会在获取到结果之前阻塞。节点可以通过同步调用，调用目标节点的函数或者服务。timeout字段是超时时间，单位是毫秒。如果目标节点超时未回复，则会调用失败。如果超时时间填0，则表示不需要目标节点回复。

- 示例：2141::102节点控制智能插座2141::101开关状态为开

```go
resp, errCode := dcom.Call(0, 1, 0x2140000000000101, 3000, []uint8{1})
```

- 示例：2141::102节点读取智能插座2141::101开关状态

```go
resp, errCode := dcom.Call(0, 2, 0x2140000000000101, 3000, nil)
if errCode == dcom.SystemOK {
	fmt.println("开关状态:", resp[0])
}
```

## 请求和应答数据格式
DCOM通信双方发送的数据流都是二进制，请求（req）和应答（resp）的数据类型都是[]uint8。

二进制不利于应用处理，所以会将二进制转换为其他数据类型来处理。常用的有以下三种：
- 结构体
- json
- 字符串

在物联网中，硬件节点的资源有限，且大部分都是使用C语言编写代码。所以建议使用C语言结构体来通信。结构体约定使用1字节对齐，小端模式。

比如海萤物联网的ntp服务提供的数据结构：
```c
struct {
    // 时区
    uint8 TimeZone
    uint16 Year
    uint8 Month
    uint8 Day
    uint8 Hour
    uint8 Minute
    uint8 Second
    // 星期
    uint8 Weekday
}
```

DCOM中提供结构体和二进制转换的函数：
```go
// StructToBytes 结构体转字节流
func StructToBytes(s interface{}) ([]uint8, error)

// BytesToStruct 字节流转结构体 结构体中的元素首字母要求大写
// s是结构体指针,保存转换后的结构体
func BytesToStruct(data []uint8, s interface{}) error
```

注意：go中定义这些结构体要求属性必须要大写，否则转换成二进制会失败。

- 示例：将时间结构体转换为二进制
```go
// ACK格式
type Time struct {
	// 时区
	TimeZone uint8
	Year     uint16
	Month    uint8
	Day      uint8
	Hour     uint8
	Minute   uint8
	Second   uint8
	// 星期
	Weekday uint8
}

t := Time{8, 2021, 4, 1, 7, 32, 1, 4}
data := StructToBytes(t)
```

- 示例：将二进制转换为结构体
```go
var t Time
err := BytesToStruct(data, &t)
```

约定：DCOM通信中如果直接使用二进制通信，建议使用大端模式。如果使用结构体通信，结构体编码是小端模式。如果混合传输，既有二进制也有结构体，二进制部分使用大端，结构体部分使用小端。