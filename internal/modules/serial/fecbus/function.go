package fecbus

import "fmt"

var namedFunctions = map[FunctionCode]FunctionInfo{
	FunctionSyncHeartbeat: {
		Code: FunctionSyncHeartbeat, Hex: "00H", Name: "同步系统节拍",
		Description: "Controller broadcasts sync heartbeat.", Direction: "controller_to_device", Answer: true,
	},
	FunctionReset: {
		Code: FunctionReset, Hex: "01H", Name: "系统复位",
		Description: "Controller resets selected system scope.", Direction: "controller_to_device", Answer: true,
	},
	FunctionSilence: {
		Code: FunctionSilence, Hex: "02H", Name: "系统消音",
		Description: "Controller silences selected system scope.", Direction: "controller_to_device", Answer: true,
	},
	FunctionSelfTest: {
		Code: FunctionSelfTest, Hex: "03H", Name: "系统自检",
		Description: "Controller starts self test.", Direction: "controller_to_device", Answer: true,
	},
	FunctionBroadcastClock: {
		Code: FunctionBroadcastClock, Hex: "04H", Name: "广播时钟",
		Description: "Controller broadcasts clock.", Direction: "controller_to_device", Answer: true,
	},
	FunctionNotifyUrgentEvent: {
		Code: FunctionNotifyUrgentEvent, Hex: "05H", Name: "通告紧急事件",
		Description: "Controller notifies device urgent event.", Direction: "controller_to_device", Answer: true,
	},
	FunctionNotifyGeneralEvent: {
		Code: FunctionNotifyGeneralEvent, Hex: "06H", Name: "通告一般事件",
		Description: "Controller notifies device general event.", Direction: "controller_to_device", Answer: true,
	},
	FunctionNotifyDebugEvent: {
		Code: FunctionNotifyDebugEvent, Hex: "07H", Name: "通告调试事件",
		Description: "Controller notifies device debug event.", Direction: "controller_to_device", Answer: true,
	},
	FunctionStatusAnswer: {
		Code: FunctionStatusAnswer, Hex: "0FH", Name: "状态应答",
		Description: "Status answer or end-frame flag.", Direction: "bidirectional", Answer: false,
	},
	FunctionDeviceUrgentEvent: {
		Code: FunctionDeviceUrgentEvent, Hex: "11H", Name: "设备紧急事件通告",
		Description: "Device notifies controller urgent event.", Direction: "device_to_controller", Answer: true,
	},
	FunctionDeviceGeneralEvent: {
		Code: FunctionDeviceGeneralEvent, Hex: "12H", Name: "设备一般事件通告",
		Description: "Device notifies controller general event.", Direction: "device_to_controller", Answer: true,
	},
	FunctionDeviceDebugEvent: {
		Code: FunctionDeviceDebugEvent, Hex: "13H", Name: "设备调试事件通告",
		Description: "Device notifies controller debug event.", Direction: "device_to_controller", Answer: true,
	},
	FunctionHeartbeatPollInfo: {
		Code: FunctionHeartbeatPollInfo, Hex: "14H", Name: "通告心跳",
		Description: "Controller communication heartbeat/poll information.", Direction: "device_to_controller", Answer: true,
	},
	FunctionPollConnectionStatus: {
		Code: FunctionPollConnectionStatus, Hex: "21H", Name: "设备巡检",
		Description: "Poll electrical control device connection status.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryDeviceStatus: {
		Code: FunctionQueryDeviceStatus, Hex: "22H", Name: "查设备状态",
		Description: "Query device status.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryConfig: {
		Code: FunctionQueryConfig, Hex: "23H", Name: "查设备配置",
		Description: "Query device configuration.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryIdentifier: {
		Code: FunctionQueryIdentifier, Hex: "24H", Name: "查设备标识",
		Description: "Query device identifier.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryParameter: {
		Code: FunctionQueryParameter, Hex: "25H", Name: "查设备参量",
		Description: "Query device parameter.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryComment: {
		Code: FunctionQueryComment, Hex: "26H", Name: "查设备注释",
		Description: "Query device comment.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryProgramming: {
		Code: FunctionQueryProgramming, Hex: "27H", Name: "查设备编程",
		Description: "Query device programming.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryRegistration: {
		Code: FunctionQueryRegistration, Hex: "28H", Name: "查注册登记信息",
		Description: "Query registration information.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryCurrentEvent: {
		Code: FunctionQueryCurrentEvent, Hex: "29H", Name: "查设备当前事件",
		Description: "Query current device event.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryHistoryEvent: {
		Code: FunctionQueryHistoryEvent, Hex: "2AH", Name: "查设备历史事件",
		Description: "Query historical device event.", Direction: "controller_to_device", Answer: true,
	},
	FunctionStopEventQuery: {
		Code: FunctionStopEventQuery, Hex: "2BH", Name: "停止查询设备事件",
		Description: "Stop querying device events.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryProtocolVersion: {
		Code: FunctionQueryProtocolVersion, Hex: "2CH", Name: "查 FECbus 协议版本号",
		Description: "Query FECbus protocol version.", Direction: "controller_to_device", Answer: true,
	},
	FunctionQueryDeviceList: {
		Code: FunctionQueryDeviceList, Hex: "2DH", Name: "查设备列表",
		Description: "Query device list.", Direction: "controller_to_device", Answer: true,
	},
}

// LookupFunction returns table C.2 metadata for all named, custom, and reserved codes.
func LookupFunction(code FunctionCode) (FunctionInfo, bool) {
	if info, ok := namedFunctions[code]; ok {
		return info, true
	}
	if isCustomFunction(code) {
		return FunctionInfo{
			Code:        code,
			Hex:         fmt.Sprintf("%02XH", byte(code)),
			Name:        "用户自定义",
			Description: "Custom FECbus function code range from table C.2.",
			Direction:   "custom",
			Answer:      true,
			Custom:      true,
		}, true
	}
	return FunctionInfo{
		Code:        code,
		Hex:         fmt.Sprintf("%02XH", byte(code)),
		Name:        "预留",
		Description: "Reserved FECbus function code.",
		Reserved:    true,
	}, true
}

// IsFunctionAllowed rejects only reserved function codes.
func IsFunctionAllowed(code FunctionCode) bool {
	info, ok := LookupFunction(code)
	return ok && !info.Reserved
}

// FunctionCatalog returns metadata for every 0-255 function code.
func FunctionCatalog() []FunctionInfo {
	out := make([]FunctionInfo, 0, 256)
	for i := 0; i <= 255; i++ {
		info, _ := LookupFunction(FunctionCode(i))
		out = append(out, info)
	}
	return out
}

func isCustomFunction(code FunctionCode) bool {
	value := byte(code)
	return (value >= 8 && value <= 14) ||
		(value >= 21 && value <= 31) ||
		(value >= 46 && value <= 127)
}
