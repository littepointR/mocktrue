package mcpserver

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/littepointR/portweave/internal/core/config"
	"github.com/littepointR/portweave/internal/core/errors"
	"github.com/littepointR/portweave/internal/modules/serial"
	"github.com/littepointR/portweave/internal/modules/serial/buffer"
	fb "github.com/littepointR/portweave/internal/modules/serial/fecbus"
	"github.com/littepointR/portweave/internal/modules/serial/manager"
	mb "github.com/littepointR/portweave/internal/modules/serial/modbus"
	"github.com/littepointR/portweave/internal/modules/serial/monitor"
	"github.com/littepointR/portweave/internal/modules/serial/port"
)

// SerialRuntime is the serial surface exposed through MCP tools.
type SerialRuntime interface {
	EnumeratePorts(context.Context) ([]port.PortInfo, error)
	OpenPort(context.Context, manager.OpenRequest) (*manager.HandleStatus, error)
	ClosePort(string) error
	ListPorts() []manager.HandleStatus
	QueryPage(string, int64, int) (*buffer.Snapshot, error)
	Send(serial.SendRequest) (int, error)
	ResetCounters(string) error
	CreateVirtualPort(context.Context, string, string) (*serial.VirtualPortInfo, error)
	DeleteVirtualPort(string) error
	ListVirtualPorts() []serial.VirtualPortInfo
	CreateVirtualPair(context.Context, string, string, string) (*serial.VirtualPairInfo, error)
	DeleteVirtualPair(string) error
	ListVirtualPairs() []serial.VirtualPairInfo
	CreateBridge(string, string, string, int) (*serial.BridgeInfo, error)
	DeleteBridge(string) error
	ListBridges() []serial.BridgeInfo
	CleanupVirtual()
	StartAutoVirtualMonitor(context.Context, serial.AutoVirtualMonitorRequest) (*monitor.SessionInfo, error)
	StopMonitor(string) error
	DeleteMonitor(string) error
	ListMonitors() []monitor.SessionInfo
	QueryMonitorFrames(monitor.QueryRequest) (*monitor.FramePage, error)
	ClearMonitorFrames(string) error
	OpenModbusSession(context.Context, mb.OpenSessionRequest) (*mb.SessionInfo, error)
	CloseModbusSession(string) error
	ListModbusSessions() []mb.SessionInfo
	ModbusMasterRequest(mb.MasterRequest) (*mb.Transaction, error)
	StartModbusSlave(mb.StartSlaveRequest) (*mb.SessionInfo, error)
	StopModbusSlave(string) error
	UpdateModbusSlaveData(string, mb.DataModelSnapshot) error
	AddModbusSlaveUnit(string, mb.SlaveUnitSnapshot) error
	RemoveModbusSlaveUnit(string, byte) error
	ListModbusSlaveUnits(string) ([]mb.SlaveUnitInfo, error)
	UpdateModbusSlaveUnitData(string, byte, mb.DataModelSnapshot) error
	ModbusScanUnitIDs(mb.UnitScanRequest) (*mb.UnitScanResult, error)
	ModbusReadRegisters(mb.RegisterReadRequest) (*mb.RegisterReadResult, error)
	ModbusScanRegisters(mb.RegisterScanRequest) (*mb.RegisterScanResult, error)
	OpenFecbusSession(context.Context, fb.OpenSessionRequest) (*fb.SessionInfo, error)
	CloseFecbusSession(string) error
	ListFecbusSessions() []fb.SessionInfo
	FecbusSendRequest(fb.SendRequest) (*fb.Transaction, error)
	StartFecbusSlave(fb.StartSlaveRequest) (*fb.SessionInfo, error)
	StopFecbusSlave(string) error
	UpdateFecbusSlaveState(string, fb.SlaveState) error
	AddFecbusSlaveUnit(string, fb.SlaveUnitState) error
	RemoveFecbusSlaveUnit(string, byte) error
	ListFecbusSlaveUnits(string) ([]fb.SlaveUnitInfo, error)
	QueryFecbusFrames(fb.QueryRequest) (*fb.FramePage, error)
	ClearFecbusFrames(string) error
	StartSerialGraph(context.Context, serial.SerialGraphStartRequest) (*serial.SerialGraphRuntimeInfo, error)
	StopSerialGraph(string) error
	ListSerialGraphs() []serial.SerialGraphRuntimeInfo
	GetSerialGraphStatus(string) (*serial.SerialGraphRuntimeInfo, error)
	SendSerialGraphNode(serial.SerialGraphSendRequest) (int, error)
	QuerySerialGraphNodeBuffer(serial.SerialGraphBufferQuery) (*buffer.Snapshot, error)
	QuerySerialGraphNodeFrames(serial.SerialGraphFrameQuery) (*serial.SerialGraphFramePage, error)
	ClearSerialGraphNodeBuffer(string, string) error
	ResetSerialGraphNodeCounters(string, string) error
}

type noArgs struct{}

type openPortArgs struct {
	PortName  string `json:"port_name"`
	BaudRate  int    `json:"baud_rate,omitempty"`
	DataBits  int    `json:"data_bits,omitempty"`
	StopBits  string `json:"stop_bits,omitempty"`
	Parity    string `json:"parity,omitempty"`
	FlowMode  string `json:"flow_mode,omitempty"`
	ReadBufKB int    `json:"read_buf_kb,omitempty"`
}

type portIDArgs struct {
	PortID string `json:"port_id"`
}

type sendArgs struct {
	PortID   string `json:"port_id"`
	Content  string `json:"content"`
	Mode     string `json:"mode,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type queryBufferArgs struct {
	PortID string `json:"port_id"`
	Offset int64  `json:"offset,omitempty"`
	Length int    `json:"length,omitempty"`
}

type virtualPortArgs struct {
	ID       string `json:"id"`
	PortName string `json:"port_name"`
}

type virtualPairArgs struct {
	ID        string `json:"id"`
	Port1Name string `json:"port1_name"`
	Port2Name string `json:"port2_name"`
}

type idArgs struct {
	ID string `json:"id"`
}

type bridgeArgs struct {
	ID       string `json:"id"`
	Port1    string `json:"port1"`
	Port2    string `json:"port2"`
	BaudRate int    `json:"baud_rate,omitempty"`
}

type startMonitorArgs struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Port      string `json:"port"`
	BaudRate  int    `json:"baud_rate,omitempty"`
	DataBits  int    `json:"data_bits,omitempty"`
	StopBits  string `json:"stop_bits,omitempty"`
	Parity    string `json:"parity,omitempty"`
	FlowMode  string `json:"flow_mode,omitempty"`
	ReadBufKB int    `json:"read_buf_kb,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
}

type monitorIDArgs struct {
	MonitorID string `json:"monitor_id"`
}

type queryMonitorFramesArgs struct {
	MonitorID string `json:"monitor_id"`
	Offset    int64  `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Direction string `json:"direction,omitempty"`
	Search    string `json:"search,omitempty"`
}

type modbusOpenSessionArgs struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Port      string `json:"port"`
	Mode      string `json:"mode,omitempty"`
	Role      string `json:"role,omitempty"`
	BaudRate  int    `json:"baud_rate,omitempty"`
	DataBits  int    `json:"data_bits,omitempty"`
	StopBits  string `json:"stop_bits,omitempty"`
	Parity    string `json:"parity,omitempty"`
	FlowMode  string `json:"flow_mode,omitempty"`
	ReadBufKB int    `json:"read_buf_kb,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
	Retries   int    `json:"retries,omitempty"`
}

type modbusSessionIDArgs struct {
	SessionID string `json:"session_id"`
}

type modbusMasterRequestArgs struct {
	SessionID      string   `json:"session_id"`
	UnitID         int      `json:"unit_id"`
	Function       int      `json:"function"`
	AddressMode    string   `json:"address_mode,omitempty"`
	Address        int      `json:"address"`
	Quantity       int      `json:"quantity,omitempty"`
	Value          int      `json:"value,omitempty"`
	CoilValues     []bool   `json:"coil_values,omitempty"`
	RegisterValues []uint16 `json:"register_values,omitempty"`
	TimeoutMs      int      `json:"timeout_ms,omitempty"`
	Retries        int      `json:"retries,omitempty"`
}

type modbusBoolPointArg struct {
	Address int  `json:"address"`
	Value   bool `json:"value"`
}

type modbusRegisterPointArg struct {
	Address int `json:"address"`
	Value   int `json:"value"`
}

type modbusDataModelArgs struct {
	Coils            []modbusBoolPointArg     `json:"coils,omitempty"`
	DiscreteInputs   []modbusBoolPointArg     `json:"discrete_inputs,omitempty"`
	InputRegisters   []modbusRegisterPointArg `json:"input_registers,omitempty"`
	HoldingRegisters []modbusRegisterPointArg `json:"holding_registers,omitempty"`
}

type modbusSlaveUnitArg struct {
	UnitID    int                 `json:"unit_id"`
	DataModel modbusDataModelArgs `json:"data_model"`
}

type modbusStartSlaveArgs struct {
	SessionID string               `json:"session_id"`
	UnitID    int                  `json:"unit_id"`
	DataModel modbusDataModelArgs  `json:"data_model,omitempty"`
	Units     []modbusSlaveUnitArg `json:"units,omitempty"`
}

type modbusUnitIDArgs struct {
	SessionID string `json:"session_id"`
	UnitID    int    `json:"unit_id"`
}

type modbusUpdateSlaveDataArgs struct {
	SessionID string              `json:"session_id"`
	DataModel modbusDataModelArgs `json:"data_model"`
}

type modbusAddSlaveUnitArgs struct {
	SessionID string              `json:"session_id"`
	UnitID    int                 `json:"unit_id"`
	DataModel modbusDataModelArgs `json:"data_model"`
}

type modbusUpdateSlaveUnitDataArgs struct {
	SessionID string              `json:"session_id"`
	UnitID    int                 `json:"unit_id"`
	DataModel modbusDataModelArgs `json:"data_model"`
}

type modbusRegisterMappingArg struct {
	Address       int     `json:"address"`
	DataType      string  `json:"data_type,omitempty"`
	WordOrder     string  `json:"word_order,omitempty"`
	Length        int     `json:"length,omitempty"`
	ScalingFactor float64 `json:"scaling_factor,omitempty"`
	Comment       string  `json:"comment,omitempty"`
	GroupEnd      bool    `json:"group_end,omitempty"`
}

type modbusReadRegistersArgs struct {
	SessionID   string                     `json:"session_id"`
	UnitID      int                        `json:"unit_id"`
	Function    int                        `json:"function"`
	AddressMode string                     `json:"address_mode,omitempty"`
	Address     int                        `json:"address"`
	Quantity    int                        `json:"quantity"`
	Mappings    []modbusRegisterMappingArg `json:"mappings,omitempty"`
	TimeoutMs   int                        `json:"timeout_ms,omitempty"`
	Retries     int                        `json:"retries,omitempty"`
}

type modbusScanUnitIDsArgs struct {
	SessionID   string `json:"session_id"`
	UnitIDs     []int  `json:"unit_ids"`
	Function    int    `json:"function"`
	AddressMode string `json:"address_mode,omitempty"`
	Address     int    `json:"address"`
	Quantity    int    `json:"quantity"`
	TimeoutMs   int    `json:"timeout_ms,omitempty"`
	Retries     int    `json:"retries,omitempty"`
}

type modbusScanRegistersArgs struct {
	SessionID    string `json:"session_id"`
	UnitID       int    `json:"unit_id"`
	Function     int    `json:"function"`
	AddressMode  string `json:"address_mode,omitempty"`
	StartAddress int    `json:"start_address"`
	EndAddress   int    `json:"end_address"`
	ChunkSize    int    `json:"chunk_size,omitempty"`
	TimeoutMs    int    `json:"timeout_ms,omitempty"`
	Retries      int    `json:"retries,omitempty"`
}

type fecbusOpenSessionArgs struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Port      string `json:"port"`
	Role      string `json:"role,omitempty"`
	BaudRate  int    `json:"baud_rate,omitempty"`
	DataBits  int    `json:"data_bits,omitempty"`
	StopBits  string `json:"stop_bits,omitempty"`
	Parity    string `json:"parity,omitempty"`
	FlowMode  string `json:"flow_mode,omitempty"`
	ReadBufKB int    `json:"read_buf_kb,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
	Retries   int    `json:"retries,omitempty"`
}

type fecbusSessionIDArgs struct {
	SessionID string `json:"session_id"`
}

type fecbusSendRequestArgs struct {
	SessionID     string `json:"session_id"`
	FrameType     int    `json:"frame_type,omitempty"`
	TargetAddress int    `json:"target_address"`
	Priority      int    `json:"priority,omitempty"`
	SourceAddress int    `json:"source_address,omitempty"`
	MessageNumber int    `json:"message_number,omitempty"`
	GroupNumber   int    `json:"group_number,omitempty"`
	Function      int    `json:"function"`
	DataHex       string `json:"data_hex,omitempty"`
	ExpectAnswer  bool   `json:"expect_answer,omitempty"`
	TimeoutMs     int    `json:"timeout_ms,omitempty"`
	Retries       int    `json:"retries,omitempty"`
}

type fecbusSlaveStateArgs struct {
	SessionID        string `json:"session_id"`
	Address          int    `json:"address,omitempty"`
	StatusCode       int    `json:"status_code,omitempty"`
	AutoStatusAnswer bool   `json:"auto_status_answer,omitempty"`
	AcceptBroadcast  bool   `json:"accept_broadcast,omitempty"`
	Units            []struct {
		Address          int  `json:"address"`
		StatusCode       int  `json:"status_code,omitempty"`
		AutoStatusAnswer bool `json:"auto_status_answer,omitempty"`
		AcceptBroadcast  bool `json:"accept_broadcast,omitempty"`
	} `json:"units,omitempty"`
}

type fecbusSlaveUnitArgs struct {
	SessionID        string `json:"session_id"`
	Address          int    `json:"address"`
	StatusCode       int    `json:"status_code,omitempty"`
	AutoStatusAnswer bool   `json:"auto_status_answer,omitempty"`
	AcceptBroadcast  bool   `json:"accept_broadcast,omitempty"`
}

type fecbusQueryFramesArgs struct {
	SessionID string                    `json:"session_id"`
	Offset    int64                     `json:"offset,omitempty"`
	Limit     int                       `json:"limit,omitempty"`
	Direction string                    `json:"direction,omitempty"`
	Search    string                    `json:"search,omitempty"`
	Custom    []fecbusCustomFunctionArg `json:"custom,omitempty"`
}

type fecbusCustomFunctionArg struct {
	Code        int                   `json:"code"`
	Name        string                `json:"name,omitempty"`
	Description string                `json:"description,omitempty"`
	Direction   string                `json:"direction,omitempty"`
	Answer      bool                  `json:"answer,omitempty"`
	Fields      []fecbusCustomDataArg `json:"fields,omitempty"`
}

type fecbusCustomDataArg struct {
	Key     string            `json:"key"`
	Label   string            `json:"label,omitempty"`
	Offset  int               `json:"offset"`
	Length  int               `json:"length"`
	Type    string            `json:"type,omitempty"`
	Endian  string            `json:"endian,omitempty"`
	Enum    map[string]string `json:"enum,omitempty"`
	Meaning string            `json:"meaning,omitempty"`
}

type serialGraphPositionArg struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type serialGraphNodeArg struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position serialGraphPositionArg `json:"position,omitempty"`
	Config   map[string]any         `json:"config,omitempty"`
}

type serialGraphEdgeArg struct {
	ID                string `json:"id"`
	Source            string `json:"source"`
	SourceHandle      string `json:"source_handle,omitempty"`
	SourceHandleCamel string `json:"sourceHandle,omitempty"`
	Target            string `json:"target"`
	TargetHandle      string `json:"target_handle,omitempty"`
	TargetHandleCamel string `json:"targetHandle,omitempty"`
}

type serialGraphValidateArgs struct {
	Nodes []serialGraphNodeArg `json:"nodes"`
	Edges []serialGraphEdgeArg `json:"edges"`
}

type serialGraphStartArgs struct {
	ID    string               `json:"id,omitempty"`
	Nodes []serialGraphNodeArg `json:"nodes"`
	Edges []serialGraphEdgeArg `json:"edges"`
}

type serialGraphDemoTemplateArgs struct {
	ID                     string `json:"id"`
	GraphID                string `json:"graph_id,omitempty"`
	PortName               string `json:"port_name,omitempty"`
	RemoteHost             string `json:"remote_host,omitempty"`
	RemotePort             any    `json:"remote_port,omitempty"`
	AllowStartDisconnected bool   `json:"allow_start_disconnected,omitempty"`
}

type serialGraphIDArgs struct {
	GraphID string `json:"graph_id"`
}

type serialGraphSendArgs struct {
	GraphID  string `json:"graph_id"`
	NodeID   string `json:"node_id"`
	Content  string `json:"content"`
	Mode     string `json:"mode,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type serialGraphNodeBufferArgs struct {
	GraphID string `json:"graph_id"`
	NodeID  string `json:"node_id"`
	Offset  int64  `json:"offset,omitempty"`
	Length  int    `json:"length,omitempty"`
}

type serialGraphNodeFrameArgs struct {
	GraphID   string `json:"graph_id"`
	NodeID    string `json:"node_id"`
	Offset    int64  `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Direction string `json:"direction,omitempty"`
	Search    string `json:"search,omitempty"`
}

type serialGraphResetNodeArgs struct {
	GraphID string `json:"graph_id"`
	NodeID  string `json:"node_id"`
}

type serialGraphPortSpec struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Direction string `json:"direction"`
	Multiple  bool   `json:"multiple,omitempty"`
}

type serialGraphProvider struct {
	Type          string                `json:"type"`
	Title         string                `json:"title"`
	Category      string                `json:"category"`
	Description   string                `json:"description"`
	Inputs        []serialGraphPortSpec `json:"inputs"`
	Outputs       []serialGraphPortSpec `json:"outputs"`
	DefaultConfig map[string]any        `json:"default_config"`
	ResourceOwner bool                  `json:"resource_owner,omitempty"`
	ResourceKeys  []string              `json:"resource_keys,omitempty"`
}

func newMCPServer(serialService SerialRuntime) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "portweave", Version: "0.1.0"}, nil)
	registerTools(server, serialService)
	return server
}

func mcpHTTPHandler(server *mcp.Server, jsonResponse bool) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: jsonResponse, Stateless: true})
}

func mcpSessionHTTPHandler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
}

func registerTools(server *mcp.Server, serialService SerialRuntime) {
	addReadTool[noArgs, map[string]any](server, "serial_enumerate_ports", "List available serial ports.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		ports, err := serialService.EnumeratePorts(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"ports": ports}, nil
	})

	addWriteTool[openPortArgs, map[string]any](server, "serial_open_port", "Open a serial port and start receiving data.", false, func(ctx context.Context, req *mcp.CallToolRequest, args openPortArgs) (*mcp.CallToolResult, map[string]any, error) {
		status, err := serialService.OpenPort(ctx, manager.OpenRequest{Config: port.SerialConfig{
			PortName:  args.PortName,
			BaudRate:  args.BaudRate,
			DataBits:  args.DataBits,
			StopBits:  args.StopBits,
			Parity:    args.Parity,
			FlowMode:  args.FlowMode,
			ReadBufKB: args.ReadBufKB,
		}})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"handle": status}, nil
	})

	addWriteTool[portIDArgs, map[string]any](server, "serial_close_port", "Close an open serial port.", true, func(ctx context.Context, req *mcp.CallToolRequest, args portIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.ClosePort(args.PortID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"closed": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_list_open_ports", "List open serial port handles.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"handles": serialService.ListPorts()}, nil
	})

	addWriteTool[sendArgs, map[string]any](server, "serial_send", "Send ASCII or HEX content to an open serial port.", false, func(ctx context.Context, req *mcp.CallToolRequest, args sendArgs) (*mcp.CallToolResult, map[string]any, error) {
		mode := args.Mode
		if mode == "" {
			mode = "ascii"
		}
		written, err := serialService.Send(serial.SendRequest{
			PortID:   args.PortID,
			Content:  args.Content,
			Mode:     mode,
			Encoding: args.Encoding,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"bytes_written": written}, nil
	})

	addReadTool[queryBufferArgs, map[string]any](server, "serial_query_buffer", "Read a page from a port receive buffer.", func(ctx context.Context, req *mcp.CallToolRequest, args queryBufferArgs) (*mcp.CallToolResult, map[string]any, error) {
		length := args.Length
		if length <= 0 {
			length = 4096
		}
		page, err := serialService.QueryPage(args.PortID, args.Offset, length)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"offset": page.Offset,
			"data":   page.Data,
			"hex":    hex.EncodeToString(page.Data),
			"total":  page.Total,
			"eof":    page.EOF,
		}, nil
	})

	addWriteTool[portIDArgs, map[string]any](server, "serial_reset_counters", "Reset RX/TX byte counters for an open port.", true, func(ctx context.Context, req *mcp.CallToolRequest, args portIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.ResetCounters(args.PortID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"reset": true}, nil
	})

	addWriteTool[virtualPortArgs, map[string]any](server, "serial_create_virtual_port", "Create a user-facing virtual serial port.", false, func(ctx context.Context, req *mcp.CallToolRequest, args virtualPortArgs) (*mcp.CallToolResult, map[string]any, error) {
		vport, err := serialService.CreateVirtualPort(ctx, args.ID, args.PortName)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"virtual_port": vport}, nil
	})

	addWriteTool[idArgs, map[string]any](server, "serial_delete_virtual_port", "Delete a virtual serial port.", true, func(ctx context.Context, req *mcp.CallToolRequest, args idArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.DeleteVirtualPort(args.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_list_virtual_ports", "List virtual serial ports.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"virtual_ports": serialService.ListVirtualPorts()}, nil
	})

	addWriteTool[virtualPairArgs, map[string]any](server, "serial_create_virtual_pair", "Create a virtual serial pair.", false, func(ctx context.Context, req *mcp.CallToolRequest, args virtualPairArgs) (*mcp.CallToolResult, map[string]any, error) {
		pair, err := serialService.CreateVirtualPair(ctx, args.ID, args.Port1Name, args.Port2Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"virtual_pair": pair}, nil
	})

	addWriteTool[idArgs, map[string]any](server, "serial_delete_virtual_pair", "Delete a virtual serial pair.", true, func(ctx context.Context, req *mcp.CallToolRequest, args idArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.DeleteVirtualPair(args.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_list_virtual_pairs", "List virtual serial pairs.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"virtual_pairs": serialService.ListVirtualPairs()}, nil
	})

	addWriteTool[bridgeArgs, map[string]any](server, "serial_create_bridge", "Bridge two serial ports.", false, func(ctx context.Context, req *mcp.CallToolRequest, args bridgeArgs) (*mcp.CallToolResult, map[string]any, error) {
		bridge, err := serialService.CreateBridge(args.ID, args.Port1, args.Port2, args.BaudRate)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"bridge": bridge}, nil
	})

	addWriteTool[idArgs, map[string]any](server, "serial_delete_bridge", "Delete a serial bridge.", true, func(ctx context.Context, req *mcp.CallToolRequest, args idArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.DeleteBridge(args.ID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_list_bridges", "List active serial bridges.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"bridges": serialService.ListBridges()}, nil
	})

	addWriteTool[noArgs, map[string]any](server, "serial_cleanup_virtual", "Clean up virtual serial ports, pairs, bridges, and open handles.", true, func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		serialService.CleanupVirtual()
		return nil, map[string]any{"cleaned": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_graph_provider_catalog", "List serial topology node providers and connection rules.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{
			"providers": serialGraphProviders(),
			"rules": map[string]any{
				"compatible_kinds": []string{"bytes", "frame", "registers", "status", "control"},
				"fan_out":          "Only output ports marked multiple, serial.tap, or serial.tee may feed multiple inputs.",
				"cycles":           "Directed cycles are rejected.",
				"resource_owners":  "Nodes that own a local serial resource must not reuse the same configured port_name; serial.remote nodes must not reuse the same normalized raw TCP endpoint in one graph.",
				"remote_security":  "serial.remote raw TCP has no authentication, authorization, encryption, or serial parameter negotiation; use trusted LAN/VPN/SSH tunnel endpoints only.",
			},
		}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_graph_demo_catalog", "List read-only serial graph demo templates for MCP automation. Templates cover graph config, backend buffers, and frames; UI operation-log state is not stored in the backend.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"templates": serialGraphDemoCatalog()}, nil
	})

	addReadTool[serialGraphDemoTemplateArgs, map[string]any](server, "serial_graph_demo_template", "Return a read-only serial graph demo template usable with serial_graph_validate and serial_graph_start. This does not query or synthesize frontend UI operation-log state.", func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphDemoTemplateArgs) (*mcp.CallToolResult, map[string]any, error) {
		template, ok, err := serialGraphDemoTemplate(args)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, errors.New(errors.CodeInvalid, "serial graph demo template not found: "+args.ID)
		}
		return nil, template, nil
	})

	addReadTool[serialGraphValidateArgs, map[string]any](server, "serial_graph_validate", "Validate a serial topology graph without starting serial resources.", func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphValidateArgs) (*mcp.CallToolResult, map[string]any, error) {
		result := validateSerialGraph(args)
		return nil, map[string]any{
			"valid":  len(result) == 0,
			"errors": result,
		}, nil
	})

	addWriteTool[serialGraphStartArgs, map[string]any](server, "serial_graph_start", "Start a serial topology graph runtime.", false, func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphStartArgs) (*mcp.CallToolResult, map[string]any, error) {
		info, err := serialService.StartSerialGraph(ctx, serialGraphStartRequest(args))
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"graph": info}, nil
	})

	addWriteTool[serialGraphIDArgs, map[string]any](server, "serial_graph_stop", "Stop a serial topology graph runtime and release owned resources.", true, func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.StopSerialGraph(args.GraphID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"stopped": true}, nil
	})

	addReadTool[serialGraphIDArgs, map[string]any](server, "serial_graph_status", "Get status for a running serial topology graph.", func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		info, err := serialService.GetSerialGraphStatus(args.GraphID)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"graph": info}, nil
	})

	addWriteTool[serialGraphSendArgs, map[string]any](server, "serial_graph_send", "Send data through a writable graph endpoint or protocol node.", false, func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphSendArgs) (*mcp.CallToolResult, map[string]any, error) {
		written, err := serialService.SendSerialGraphNode(serial.SerialGraphSendRequest{
			GraphID:  args.GraphID,
			NodeID:   args.NodeID,
			Content:  args.Content,
			Mode:     args.Mode,
			Encoding: args.Encoding,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"bytes_written": written}, nil
	})

	addReadTool[serialGraphNodeBufferArgs, map[string]any](server, "serial_graph_query_node_buffer", "Read buffered bytes from a graph node.", func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphNodeBufferArgs) (*mcp.CallToolResult, map[string]any, error) {
		page, err := serialService.QuerySerialGraphNodeBuffer(serial.SerialGraphBufferQuery{
			GraphID: args.GraphID,
			NodeID:  args.NodeID,
			Offset:  args.Offset,
			Length:  args.Length,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"offset": page.Offset,
			"total":  page.Total,
			"eof":    page.EOF,
			"hex":    hex.EncodeToString(page.Data),
			"data":   string(page.Data),
		}, nil
	})

	addReadTool[serialGraphNodeFrameArgs, map[string]any](server, "serial_graph_query_node_frames", "Read monitor/protocol frames captured by a graph node.", func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphNodeFrameArgs) (*mcp.CallToolResult, map[string]any, error) {
		page, err := serialService.QuerySerialGraphNodeFrames(serial.SerialGraphFrameQuery{
			GraphID:   args.GraphID,
			NodeID:    args.NodeID,
			Offset:    args.Offset,
			Limit:     args.Limit,
			Direction: args.Direction,
			Search:    args.Search,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"frames": page.Frames, "total": page.Total, "next_offset": page.NextOffset}, nil
	})

	addWriteTool[serialGraphResetNodeArgs, map[string]any](server, "serial_graph_clear_node_buffer", "Clear buffered data and captured frames for a graph node.", true, func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphResetNodeArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.ClearSerialGraphNodeBuffer(args.GraphID, args.NodeID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"cleared": true}, nil
	})

	addWriteTool[serialGraphResetNodeArgs, map[string]any](server, "serial_graph_reset_node_counters", "Reset RX/TX counters for a graph node.", true, func(ctx context.Context, req *mcp.CallToolRequest, args serialGraphResetNodeArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.ResetSerialGraphNodeCounters(args.GraphID, args.NodeID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"reset": true}, nil
	})

	addWriteTool[startMonitorArgs, map[string]any](server, "serial_start_monitor", "Start monitoring one serial port and expose an auto-created virtual port for external tools.", false, func(ctx context.Context, req *mcp.CallToolRequest, args startMonitorArgs) (*mcp.CallToolResult, map[string]any, error) {
		session, err := serialService.StartAutoVirtualMonitor(ctx, serial.AutoVirtualMonitorRequest{
			ID:   args.ID,
			Name: args.Name,
			Port: args.Port,
			Config: port.SerialConfig{
				BaudRate:  args.BaudRate,
				DataBits:  args.DataBits,
				StopBits:  args.StopBits,
				Parity:    args.Parity,
				FlowMode:  args.FlowMode,
				ReadBufKB: args.ReadBufKB,
			},
			Encoding: args.Encoding,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"monitor": session}, nil
	})

	addWriteTool[monitorIDArgs, map[string]any](server, "serial_stop_monitor", "Stop a serial monitor and release its auto-created virtual port.", true, func(ctx context.Context, req *mcp.CallToolRequest, args monitorIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.StopMonitor(args.MonitorID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"stopped": true}, nil
	})

	addWriteTool[monitorIDArgs, map[string]any](server, "serial_delete_monitor", "Delete a serial monitor and its captured frames.", true, func(ctx context.Context, req *mcp.CallToolRequest, args monitorIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.DeleteMonitor(args.MonitorID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"deleted": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "serial_list_monitors", "List serial monitor sessions.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"monitors": serialService.ListMonitors()}, nil
	})

	addReadTool[queryMonitorFramesArgs, map[string]any](server, "serial_query_monitor_frames", "Read a filtered page of captured serial monitor frames.", func(ctx context.Context, req *mcp.CallToolRequest, args queryMonitorFramesArgs) (*mcp.CallToolResult, map[string]any, error) {
		page, err := serialService.QueryMonitorFrames(monitor.QueryRequest{
			MonitorID: args.MonitorID,
			Offset:    args.Offset,
			Limit:     args.Limit,
			Direction: args.Direction,
			Search:    args.Search,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"page": page}, nil
	})

	addWriteTool[monitorIDArgs, map[string]any](server, "serial_clear_monitor_frames", "Clear captured frames and counters for a serial monitor.", true, func(ctx context.Context, req *mcp.CallToolRequest, args monitorIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.ClearMonitorFrames(args.MonitorID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"cleared": true}, nil
	})

	addWriteTool[modbusOpenSessionArgs, map[string]any](server, "modbus_open_session", "Open a dedicated Modbus RTU/ASCII serial session.", false, func(ctx context.Context, req *mcp.CallToolRequest, args modbusOpenSessionArgs) (*mcp.CallToolResult, map[string]any, error) {
		session, err := serialService.OpenModbusSession(ctx, mb.OpenSessionRequest{
			ID:   args.ID,
			Name: args.Name,
			Mode: mb.FrameMode(args.Mode),
			Role: mb.SessionRole(args.Role),
			Config: port.SerialConfig{
				PortName:  args.Port,
				BaudRate:  args.BaudRate,
				DataBits:  args.DataBits,
				StopBits:  args.StopBits,
				Parity:    args.Parity,
				FlowMode:  args.FlowMode,
				ReadBufKB: args.ReadBufKB,
			},
			TimeoutMs: args.TimeoutMs,
			Retries:   args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"session": session}, nil
	})

	addWriteTool[modbusSessionIDArgs, map[string]any](server, "modbus_close_session", "Close a Modbus serial session.", true, func(ctx context.Context, req *mcp.CallToolRequest, args modbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.CloseModbusSession(args.SessionID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"closed": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "modbus_list_sessions", "List open Modbus sessions.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"sessions": serialService.ListModbusSessions()}, nil
	})

	addWriteTool[modbusMasterRequestArgs, map[string]any](server, "modbus_master_request", "Send one Modbus master read or write request.", false, func(ctx context.Context, req *mcp.CallToolRequest, args modbusMasterRequestArgs) (*mcp.CallToolResult, map[string]any, error) {
		tx, err := serialService.ModbusMasterRequest(mb.MasterRequest{
			SessionID:      args.SessionID,
			UnitID:         byte(clampInt(args.UnitID, 1, 247)),
			Function:       mb.FunctionCode(args.Function),
			AddressMode:    mb.AddressMode(args.AddressMode),
			Address:        uint16(clampInt(args.Address, 0, 65535)),
			Quantity:       uint16(clampInt(args.Quantity, 0, 65535)),
			Value:          uint16(clampInt(args.Value, 0, 65535)),
			CoilValues:     args.CoilValues,
			RegisterValues: args.RegisterValues,
			TimeoutMs:      args.TimeoutMs,
			Retries:        args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"transaction": tx}, nil
	})

	addWriteTool[modbusStartSlaveArgs, map[string]any](server, "modbus_start_slave", "Start Modbus slave simulation for the selected Unit ID or Unit list.", false, func(ctx context.Context, req *mcp.CallToolRequest, args modbusStartSlaveArgs) (*mcp.CallToolResult, map[string]any, error) {
		units := make([]mb.SlaveUnitSnapshot, 0, len(args.Units))
		for _, unit := range args.Units {
			units = append(units, mb.SlaveUnitSnapshot{
				UnitID:    byte(clampInt(unit.UnitID, 1, 247)),
				DataModel: dataModelSnapshot(unit.DataModel),
			})
		}
		session, err := serialService.StartModbusSlave(mb.StartSlaveRequest{
			SessionID: args.SessionID,
			UnitID:    byte(clampInt(args.UnitID, 1, 247)),
			DataModel: dataModelSnapshot(args.DataModel),
			Units:     units,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"session": session}, nil
	})

	addWriteTool[modbusSessionIDArgs, map[string]any](server, "modbus_stop_slave", "Stop Modbus slave simulation.", true, func(ctx context.Context, req *mcp.CallToolRequest, args modbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.StopModbusSlave(args.SessionID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"stopped": true}, nil
	})

	addWriteTool[modbusUpdateSlaveDataArgs, map[string]any](server, "modbus_update_slave_data", "Replace the default Modbus slave data model.", false, func(ctx context.Context, req *mcp.CallToolRequest, args modbusUpdateSlaveDataArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.UpdateModbusSlaveData(args.SessionID, dataModelSnapshot(args.DataModel)); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"updated": true}, nil
	})

	addWriteTool[modbusAddSlaveUnitArgs, map[string]any](server, "modbus_add_slave_unit", "Add one Unit ID to a Modbus slave simulation.", false, func(ctx context.Context, req *mcp.CallToolRequest, args modbusAddSlaveUnitArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.AddModbusSlaveUnit(args.SessionID, mb.SlaveUnitSnapshot{
			UnitID:    byte(clampInt(args.UnitID, 1, 247)),
			DataModel: dataModelSnapshot(args.DataModel),
		}); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"added": true}, nil
	})

	addWriteTool[modbusUnitIDArgs, map[string]any](server, "modbus_remove_slave_unit", "Remove one Unit ID from a Modbus slave simulation.", true, func(ctx context.Context, req *mcp.CallToolRequest, args modbusUnitIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.RemoveModbusSlaveUnit(args.SessionID, byte(clampInt(args.UnitID, 1, 247))); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"removed": true}, nil
	})

	addReadTool[modbusSessionIDArgs, map[string]any](server, "modbus_list_slave_units", "List configured Unit IDs for a Modbus slave session.", func(ctx context.Context, req *mcp.CallToolRequest, args modbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		units, err := serialService.ListModbusSlaveUnits(args.SessionID)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"units": units}, nil
	})

	addWriteTool[modbusUpdateSlaveUnitDataArgs, map[string]any](server, "modbus_update_slave_unit_data", "Replace one Unit ID data model in a Modbus slave simulation.", false, func(ctx context.Context, req *mcp.CallToolRequest, args modbusUpdateSlaveUnitDataArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.UpdateModbusSlaveUnitData(args.SessionID, byte(clampInt(args.UnitID, 1, 247)), dataModelSnapshot(args.DataModel)); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"updated": true}, nil
	})

	addReadTool[modbusReadRegistersArgs, map[string]any](server, "modbus_read_registers", "Read Modbus coils/registers and decode optional mappings.", func(ctx context.Context, req *mcp.CallToolRequest, args modbusReadRegistersArgs) (*mcp.CallToolResult, map[string]any, error) {
		result, err := serialService.ModbusReadRegisters(mb.RegisterReadRequest{
			SessionID:   args.SessionID,
			UnitID:      byte(clampInt(args.UnitID, 1, 247)),
			Function:    mb.FunctionCode(args.Function),
			AddressMode: mb.AddressMode(args.AddressMode),
			Address:     uint16(clampInt(args.Address, 0, 65535)),
			Quantity:    uint16(clampInt(args.Quantity, 1, 65535)),
			Mappings:    registerMappings(args.Mappings),
			TimeoutMs:   args.TimeoutMs,
			Retries:     args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"result": result}, nil
	})

	addReadTool[modbusScanUnitIDsArgs, map[string]any](server, "modbus_scan_unit_ids", "Scan Modbus Unit IDs with a small read request.", func(ctx context.Context, req *mcp.CallToolRequest, args modbusScanUnitIDsArgs) (*mcp.CallToolResult, map[string]any, error) {
		result, err := serialService.ModbusScanUnitIDs(mb.UnitScanRequest{
			SessionID:   args.SessionID,
			UnitIDs:     args.UnitIDs,
			Function:    mb.FunctionCode(args.Function),
			AddressMode: mb.AddressMode(args.AddressMode),
			Address:     uint16(clampInt(args.Address, 0, 65535)),
			Quantity:    uint16(clampInt(args.Quantity, 1, 65535)),
			TimeoutMs:   args.TimeoutMs,
			Retries:     args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"result": result}, nil
	})

	addReadTool[modbusScanRegistersArgs, map[string]any](server, "modbus_scan_registers", "Scan a Modbus register range and return non-zero values.", func(ctx context.Context, req *mcp.CallToolRequest, args modbusScanRegistersArgs) (*mcp.CallToolResult, map[string]any, error) {
		result, err := serialService.ModbusScanRegisters(mb.RegisterScanRequest{
			SessionID:    args.SessionID,
			UnitID:       byte(clampInt(args.UnitID, 1, 247)),
			Function:     mb.FunctionCode(args.Function),
			AddressMode:  mb.AddressMode(args.AddressMode),
			StartAddress: uint16(clampInt(args.StartAddress, 0, 65535)),
			EndAddress:   uint16(clampInt(args.EndAddress, 0, 65535)),
			ChunkSize:    uint16(clampInt(args.ChunkSize, 1, 65535)),
			TimeoutMs:    args.TimeoutMs,
			Retries:      args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"result": result}, nil
	})

	addReadTool[noArgs, map[string]any](server, "fecbus_function_catalog", "List FECbus function codes from GB 4717-2024 Appendix C table C.2.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"functions": fb.FunctionCatalog()}, nil
	})

	addWriteTool[fecbusOpenSessionArgs, map[string]any](server, "fecbus_open_session", "Open a dedicated FECbus serial session.", false, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusOpenSessionArgs) (*mcp.CallToolResult, map[string]any, error) {
		session, err := serialService.OpenFecbusSession(ctx, fb.OpenSessionRequest{
			ID:   args.ID,
			Name: args.Name,
			Role: fb.SessionRole(args.Role),
			Config: port.SerialConfig{
				PortName:  args.Port,
				BaudRate:  args.BaudRate,
				DataBits:  args.DataBits,
				StopBits:  args.StopBits,
				Parity:    args.Parity,
				FlowMode:  args.FlowMode,
				ReadBufKB: args.ReadBufKB,
			},
			TimeoutMs: args.TimeoutMs,
			Retries:   args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"session": session}, nil
	})

	addWriteTool[fecbusSessionIDArgs, map[string]any](server, "fecbus_close_session", "Close a FECbus serial session.", true, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.CloseFecbusSession(args.SessionID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"closed": true}, nil
	})

	addReadTool[noArgs, map[string]any](server, "fecbus_list_sessions", "List open FECbus sessions.", func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"sessions": serialService.ListFecbusSessions()}, nil
	})

	addWriteTool[fecbusSendRequestArgs, map[string]any](server, "fecbus_send_request", "Send one FECbus frame and optionally wait for an answer.", false, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSendRequestArgs) (*mcp.CallToolResult, map[string]any, error) {
		tx, err := serialService.FecbusSendRequest(fb.SendRequest{
			SessionID:     args.SessionID,
			FrameType:     fb.FrameType(clampInt(args.FrameType, 0, 1)),
			TargetAddress: byte(clampInt(args.TargetAddress, 0, 63)),
			Priority:      byte(clampInt(args.Priority, 0, 3)),
			SourceAddress: byte(clampInt(args.SourceAddress, 0, 63)),
			MessageNumber: byte(clampInt(args.MessageNumber, 0, 63)),
			GroupNumber:   byte(clampInt(args.GroupNumber, 0, 127)),
			Function:      fb.FunctionCode(clampInt(args.Function, 0, 255)),
			DataHex:       args.DataHex,
			ExpectAnswer:  args.ExpectAnswer,
			TimeoutMs:     args.TimeoutMs,
			Retries:       args.Retries,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"transaction": tx}, nil
	})

	addWriteTool[fecbusSlaveStateArgs, map[string]any](server, "fecbus_start_slave", "Start FECbus electrical-control-device simulation.", false, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSlaveStateArgs) (*mcp.CallToolResult, map[string]any, error) {
		session, err := serialService.StartFecbusSlave(fb.StartSlaveRequest{
			SessionID: args.SessionID,
			State:     fecbusSlaveState(args),
			Units:     fecbusSlaveUnits(args),
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"session": session}, nil
	})

	addWriteTool[fecbusSessionIDArgs, map[string]any](server, "fecbus_stop_slave", "Stop FECbus device simulation.", true, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.StopFecbusSlave(args.SessionID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"stopped": true}, nil
	})

	addWriteTool[fecbusSlaveStateArgs, map[string]any](server, "fecbus_update_slave_state", "Replace FECbus device simulation status-answer state.", false, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSlaveStateArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.UpdateFecbusSlaveState(args.SessionID, fecbusSlaveState(args)); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"updated": true}, nil
	})

	addReadTool[fecbusSessionIDArgs, map[string]any](server, "fecbus_list_slave_units", "List configured FECbus simulated slave addresses.", func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		units, err := serialService.ListFecbusSlaveUnits(args.SessionID)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"units": units}, nil
	})

	addWriteTool[fecbusSlaveUnitArgs, map[string]any](server, "fecbus_add_slave_unit", "Add or replace one FECbus simulated slave address.", false, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSlaveUnitArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.AddFecbusSlaveUnit(args.SessionID, fecbusSlaveUnit(args)); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"updated": true}, nil
	})

	addWriteTool[fecbusSlaveUnitArgs, map[string]any](server, "fecbus_remove_slave_unit", "Remove one FECbus simulated slave address.", true, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSlaveUnitArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.RemoveFecbusSlaveUnit(args.SessionID, byte(clampInt(args.Address, 1, 63))); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"removed": true}, nil
	})

	addReadTool[fecbusQueryFramesArgs, map[string]any](server, "fecbus_query_frames", "Read a filtered page of FECbus TX/RX frames.", func(ctx context.Context, req *mcp.CallToolRequest, args fecbusQueryFramesArgs) (*mcp.CallToolResult, map[string]any, error) {
		page, err := serialService.QueryFecbusFrames(fb.QueryRequest{
			SessionID: args.SessionID,
			Offset:    args.Offset,
			Limit:     args.Limit,
			Direction: args.Direction,
			Search:    args.Search,
			Custom:    fecbusCustomFunctions(args.Custom),
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"page": page}, nil
	})

	addWriteTool[fecbusSessionIDArgs, map[string]any](server, "fecbus_clear_frames", "Clear captured FECbus frames and counters.", true, func(ctx context.Context, req *mcp.CallToolRequest, args fecbusSessionIDArgs) (*mcp.CallToolResult, map[string]any, error) {
		if err := serialService.ClearFecbusFrames(args.SessionID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"cleared": true}, nil
	})
}

func addReadTool[In, Out any](server *mcp.Server, name, description string, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler)
}

func addWriteTool[In, Out any](server *mcp.Server, name, description string, destructive bool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(destructive),
			OpenWorldHint:   boolPtr(false),
			ReadOnlyHint:    false,
		},
	}, handler)
}

func boolPtr(value bool) *bool {
	return &value
}

func serialGraphProviders() []serialGraphProvider {
	serialDefaults := map[string]any{
		"portName":  "",
		"baudRate":  115200,
		"dataBits":  8,
		"stopBits":  "1",
		"parity":    "none",
		"flowMode":  "none",
		"readBufKB": 32,
	}
	remoteDefaults := map[string]any{
		"protocol":               "raw-tcp",
		"role":                   "client",
		"host":                   "",
		"port":                   3001,
		"connectTimeoutMs":       3000,
		"writeTimeoutMs":         3000,
		"reconnect":              true,
		"reconnectIntervalMs":    1000,
		"allowStartDisconnected": false,
		"readBufKB":              32,
		"baudRate":               115200,
		"dataBits":               8,
		"stopBits":               "1",
		"parity":                 "none",
		"flowMode":               "none",
		"viewMode":               "ascii",
		"autoScroll":             true,
		"showTimestamp":          false,
	}
	bytesIn := serialGraphPortSpec{ID: "in", Label: "接收", Kind: "bytes", Direction: "input"}
	bytesOut := serialGraphPortSpec{ID: "out", Label: "发送", Kind: "bytes", Direction: "output"}
	scriptDefaults := scriptDefaultsForTemplate()

	return []serialGraphProvider{
		{
			Type:          "serial.physical",
			Title:         "物理串口",
			Category:      "串口",
			Description:   "系统中已有的真实串口资源。",
			Inputs:        []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "output"}},
			DefaultConfig: serialDefaults,
			ResourceOwner: true,
			ResourceKeys:  []string{"portName"},
		},
		{
			Type:          "serial.virtual",
			Title:         "虚拟串口",
			Category:      "串口",
			Description:   "由 PortWeave 创建和管理的单端虚拟串口。",
			Inputs:        []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "output"}},
			DefaultConfig: withConfig(serialDefaults, map[string]any{"portName": "portweave-vport"}),
			ResourceOwner: true,
			ResourceKeys:  []string{"portName"},
		},
		{
			Type:          "serial.remote",
			Title:         "远端串口",
			Category:      "串口",
			Description:   "通过 raw TCP client/server 连接远端串口端点并在拓扑中收发字节流。",
			Inputs:        []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "output"}},
			DefaultConfig: remoteDefaults,
			ResourceOwner: true,
		},
		{
			Type:        "serial.bridge",
			Title:       "串口桥接",
			Category:    "串口",
			Description: "把两路字节流双向桥接。",
			Inputs: []serialGraphPortSpec{
				{ID: "a-in", Label: "接收 A", Kind: "bytes", Direction: "input"},
				{ID: "b-in", Label: "接收 B", Kind: "bytes", Direction: "input"},
			},
			Outputs: []serialGraphPortSpec{
				{ID: "a-out", Label: "发送 A", Kind: "bytes", Direction: "output"},
				{ID: "b-out", Label: "发送 B", Kind: "bytes", Direction: "output"},
			},
			DefaultConfig: map[string]any{},
		},
		{
			Type:          "serial.monitor",
			Title:         "串口监控",
			Category:      "工具",
			Description:   "监听一路串口字节流并生成监控帧。",
			Inputs:        []serialGraphPortSpec{bytesIn},
			Outputs:       []serialGraphPortSpec{},
			DefaultConfig: map[string]any{"displayMode": "hex"},
		},
		{
			Type:          "serial.filter",
			Title:         "过滤器",
			Category:      "工具",
			Description:   "按关键字、正则或 Wireshark-like 表达式过滤字节流。",
			Inputs:        []serialGraphPortSpec{bytesIn},
			Outputs:       []serialGraphPortSpec{bytesOut},
			DefaultConfig: map[string]any{"mode": "plain", "expression": "", "caseSensitive": false, "wholeWord": false},
		},
		{
			Type:          "serial.tap",
			Title:         "分流器",
			Category:      "工具",
			Description:   "允许一路字节流分发到多个下游节点。",
			Inputs:        []serialGraphPortSpec{bytesIn},
			Outputs:       []serialGraphPortSpec{{ID: bytesOut.ID, Label: bytesOut.Label, Kind: bytesOut.Kind, Direction: bytesOut.Direction, Multiple: true}},
			DefaultConfig: map[string]any{},
		},
		{
			Type:          "serial.tee",
			Title:         "T 型分支",
			Category:      "工具",
			Description:   "分流器的别名，用于表达串联链路中的并联分支。",
			Inputs:        []serialGraphPortSpec{bytesIn},
			Outputs:       []serialGraphPortSpec{{ID: bytesOut.ID, Label: bytesOut.Label, Kind: bytesOut.Kind, Direction: bytesOut.Direction, Multiple: true}},
			DefaultConfig: map[string]any{},
		},
		{
			Type:          "serial.script.transform",
			Title:         "脚本转换",
			Category:      "脚本",
			Description:   "通过安全脚本处理输入字节流并输出处理后的字节流。",
			Inputs:        []serialGraphPortSpec{bytesIn},
			Outputs:       []serialGraphPortSpec{bytesOut},
			DefaultConfig: scriptDefaults,
		},
		{
			Type:        "serial.script.generator",
			Title:       "脚本生成",
			Category:    "脚本",
			Description: "通过安全脚本按启动或定时器生成字节流。",
			Inputs:      []serialGraphPortSpec{},
			Outputs:     []serialGraphPortSpec{bytesOut},
			DefaultConfig: withConfig(scriptDefaults, map[string]any{
				"script":     "output.text(\"tick\", \"utf-8\")",
				"autoRun":    true,
				"intervalMs": 1000,
			}),
		},
		{
			Type:        "serial.script.analyzer",
			Title:       "脚本分析",
			Category:    "脚本",
			Description: "通过安全脚本解析输入字节流并生成字段和错误记录。",
			Inputs:      []serialGraphPortSpec{bytesIn},
			Outputs:     []serialGraphPortSpec{},
			DefaultConfig: withConfig(scriptDefaults, map[string]any{
				"script":      "field(\"length\", input.bytes().length)",
				"displayMode": "hex",
			}),
		},
		{
			Type:          "serial.modbus.master",
			Title:         "Modbus 主站",
			Category:      "协议",
			Description:   "Modbus RTU/ASCII 主站请求和寄存器视图。",
			Inputs:        []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "output"}},
			DefaultConfig: map[string]any{"mode": "rtu", "unitIds": "1", "addressMode": "zero-based", "functionCode": 3, "address": 0, "quantity": 1, "value": 0},
		},
		{
			Type:          "serial.modbus.slave",
			Title:         "Modbus 从站",
			Category:      "协议",
			Description:   "Modbus RTU/ASCII 多 Unit ID 从站数据区。",
			Inputs:        []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "output"}},
			DefaultConfig: map[string]any{"mode": "rtu", "unitIds": "1"},
		},
		{
			Type:          "serial.fecbus.master",
			Title:         "FECbus 主控",
			Category:      "协议",
			Description:   "FECbus 主控请求和帧解析。",
			Inputs:        []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "output"}},
			DefaultConfig: map[string]any{"sourceAddress": 1, "targetAddress": 2, "priority": 3, "messageNumber": 1, "groupNumber": 0, "functionCode": 44, "dataHex": ""},
		},
		{
			Type:          "serial.fecbus.slave",
			Title:         "FECbus 从机",
			Category:      "协议",
			Description:   "FECbus 从机应答和设备状态。",
			Inputs:        []serialGraphPortSpec{{ID: "rx", Label: "接收", Kind: "bytes", Direction: "input"}},
			Outputs:       []serialGraphPortSpec{{ID: "tx", Label: "发送", Kind: "bytes", Direction: "output"}},
			DefaultConfig: map[string]any{"address": 2, "defaultStatus": 10, "autoStatusAnswer": true},
		},
	}
}

const (
	serialGraphObservabilityTemplateID = "serial-observability-filter-logging"
	serialGraphRemoteTemplateID        = "serial-remote-raw-tcp"
)

func serialGraphDemoCatalog() []map[string]any {
	return []map[string]any{
		{
			"id":                  serialGraphObservabilityTemplateID,
			"title":               "Serial observability filter/logging",
			"description":         "script generator -> virtual -> tap -> plain/regex/expression filters -> monitors; usable with serial_graph_validate and serial_graph_start.",
			"tags":                []string{"serial", "filter", "logging", "observability"},
			"node_types":          []string{"serial.script.generator", "serial.virtual", "serial.tap", "serial.filter", "serial.monitor"},
			"operation_log_state": "not_in_backend",
		},
		{
			"id":                  serialGraphRemoteTemplateID,
			"title":               "Remote serial raw TCP",
			"description":         "raw TCP server/client loopback with endpoint buffers and monitor branches; usable with serial_graph_validate and serial_graph_start.",
			"tags":                []string{"serial", "remote", "raw-tcp", "mcp"},
			"node_types":          []string{"serial.script.generator", "serial.remote", "serial.tap", "serial.monitor"},
			"operation_log_state": "not_in_backend",
			"security":            "raw TCP has no built-in authentication or encryption; bind gateways to trusted interfaces or wrap with SSH/VPN/TLS outside PortWeave.",
		},
	}
}

func serialGraphDemoTemplate(args serialGraphDemoTemplateArgs) (map[string]any, bool, error) {
	switch args.ID {
	case serialGraphObservabilityTemplateID:
		return serialGraphObservabilityDemoTemplate(args), true, nil
	case serialGraphRemoteTemplateID:
		template, err := serialGraphRemoteDemoTemplate(args)
		return template, true, err
	default:
		return nil, false, nil
	}
}

func scriptDefaultsForTemplate() map[string]any {
	return map[string]any{
		"script":         "output.bytes(input.bytes())",
		"timeoutMs":      50,
		"maxOutputBytes": 65536,
		"maxStateBytes":  262144,
		"onError":        "mark-error-and-drop",
		"encoding":       "utf-8",
		"autoRun":        false,
		"intervalMs":     1000,
		"displayMode":    "hex",
	}
}

func serialGraphObservabilityDemoTemplate(args serialGraphDemoTemplateArgs) map[string]any {
	graphID := strings.TrimSpace(args.GraphID)
	if graphID == "" {
		graphID = "serial-observability-mcp-demo"
	}
	portName := strings.TrimSpace(args.PortName)
	if portName == "" {
		portName = "portweave-mcp-demo"
	}

	generatorConfig := withConfig(scriptDefaultsForTemplate(), map[string]any{"script": "output.text(\"OK TEMP=42\\r\\n\", \"utf-8\")", "autoRun": true, "intervalMs": 1000})
	nodes := []serialGraphNodeArg{
		{ID: "generator", Type: "serial.script.generator", Position: serialGraphPositionArg{X: 20, Y: 80}, Config: generatorConfig},
		{ID: "virtual", Type: "serial.virtual", Position: serialGraphPositionArg{X: 240, Y: 80}, Config: map[string]any{"portName": portName, "baudRate": 115200, "dataBits": 8, "stopBits": "1", "parity": "none", "flowMode": "none", "readBufKB": 32}},
		{ID: "tap", Type: "serial.tap", Position: serialGraphPositionArg{X: 460, Y: 80}, Config: map[string]any{}},
		{ID: "filter-plain", Type: "serial.filter", Position: serialGraphPositionArg{X: 680, Y: 0}, Config: map[string]any{"mode": "plain", "expression": "OK", "caseSensitive": false, "wholeWord": false}},
		{ID: "filter-regex", Type: "serial.filter", Position: serialGraphPositionArg{X: 680, Y: 120}, Config: map[string]any{"mode": "regex", "expression": `TEMP=[0-9]+`, "caseSensitive": false, "wholeWord": false}},
		{ID: "filter-expression", Type: "serial.filter", Position: serialGraphPositionArg{X: 680, Y: 240}, Config: map[string]any{"mode": "expression", "expression": "len >= 4 and hex contains \"0d0a\"", "caseSensitive": false, "wholeWord": false}},
		{ID: "monitor-plain", Type: "serial.monitor", Position: serialGraphPositionArg{X: 940, Y: 0}, Config: map[string]any{"displayMode": "ascii"}},
		{ID: "monitor-regex", Type: "serial.monitor", Position: serialGraphPositionArg{X: 940, Y: 120}, Config: map[string]any{"displayMode": "hex"}},
		{ID: "monitor-expression", Type: "serial.monitor", Position: serialGraphPositionArg{X: 940, Y: 240}, Config: map[string]any{"displayMode": "ascii"}},
	}
	edges := []serialGraphEdgeArg{
		{ID: "edge-generator-virtual", Source: "generator", SourceHandle: "out", Target: "virtual", TargetHandle: "tx"},
		{ID: "edge-virtual-tap", Source: "virtual", SourceHandle: "rx", Target: "tap", TargetHandle: "in"},
		{ID: "edge-tap-filter-plain", Source: "tap", SourceHandle: "out", Target: "filter-plain", TargetHandle: "in"},
		{ID: "edge-tap-filter-regex", Source: "tap", SourceHandle: "out", Target: "filter-regex", TargetHandle: "in"},
		{ID: "edge-tap-filter-expression", Source: "tap", SourceHandle: "out", Target: "filter-expression", TargetHandle: "in"},
		{ID: "edge-filter-plain-monitor", Source: "filter-plain", SourceHandle: "out", Target: "monitor-plain", TargetHandle: "in"},
		{ID: "edge-filter-regex-monitor", Source: "filter-regex", SourceHandle: "out", Target: "monitor-regex", TargetHandle: "in"},
		{ID: "edge-filter-expression-monitor", Source: "filter-expression", SourceHandle: "out", Target: "monitor-expression", TargetHandle: "in"},
	}

	return map[string]any{
		"id":          serialGraphObservabilityTemplateID,
		"graph_id":    graphID,
		"title":       "Serial observability filter/logging",
		"description": "Read-only MCP template demonstrating script-generated traffic, virtual loopback, plain/regex/expression filters, and monitor frames.",
		"nodes":       nodes,
		"edges":       edges,
		"usage": map[string]any{
			"validate":             "Call serial_graph_validate with the nodes and edges from this response.",
			"start":                map[string]any{"tool": "serial_graph_start", "arguments": serialGraphStartArgs{ID: graphID, Nodes: nodes, Edges: edges}},
			"send_pass_example":    map[string]any{"tool": "serial_graph_send", "arguments": map[string]any{"graph_id": graphID, "node_id": "virtual", "content": "OK TEMP=42\r\n", "mode": "ascii"}},
			"send_drop_example":    map[string]any{"tool": "serial_graph_send", "arguments": map[string]any{"graph_id": graphID, "node_id": "virtual", "content": "DROP", "mode": "ascii"}},
			"query_buffer_example": map[string]any{"tool": "serial_graph_query_node_buffer", "arguments": map[string]any{"graph_id": graphID, "node_id": "virtual", "offset": 0, "length": 256}},
			"query_frames_example": map[string]any{"tool": "serial_graph_query_node_frames", "arguments": map[string]any{"graph_id": graphID, "node_id": "monitor-plain", "offset": 0, "limit": 20}},
			"operation_log_state":  "MCP supports graph config/templates plus backend buffers and frames; frontend UI operation-log state is not stored in the backend.",
		},
	}
}

func serialGraphRemoteDemoTemplate(args serialGraphDemoTemplateArgs) (map[string]any, error) {
	graphID := strings.TrimSpace(args.GraphID)
	if graphID == "" {
		graphID = "serial-remote-raw-tcp-mcp-demo"
	}
	host := strings.TrimSpace(args.RemoteHost)
	if host == "" {
		host = "127.0.0.1"
	}
	if strings.Contains(host, "://") {
		return nil, errors.New(errors.CodeInvalid, "remote_host must not include URL scheme")
	}
	portValue := args.RemotePort
	if portValue == nil {
		portValue = 3001
	}
	port, portOK := serialGraphStrictIntConfig(map[string]any{"remote_port": portValue}, "remote_port", 3001)
	if !portOK {
		return nil, errors.New(errors.CodeInvalid, "remote_port must be an integer")
	}
	if port < 1 || port > 65535 {
		return nil, errors.New(errors.CodeInvalid, fmt.Sprintf("remote_port out of range: %d", port))
	}

	remoteBaseConfig := map[string]any{
		"protocol":               "raw-tcp",
		"host":                   host,
		"port":                   port,
		"connectTimeoutMs":       3000,
		"writeTimeoutMs":         3000,
		"reconnect":              true,
		"reconnectIntervalMs":    1000,
		"allowStartDisconnected": args.AllowStartDisconnected,
		"readBufKB":              32,
		"baudRate":               115200,
		"dataBits":               8,
		"stopBits":               "1",
		"parity":                 "none",
		"flowMode":               "none",
		"viewMode":               "ascii",
		"autoScroll":             true,
		"showTimestamp":          true,
	}
	serverConfig := withConfig(remoteBaseConfig, map[string]any{"role": "server", "reconnect": false, "allowStartDisconnected": false})
	clientConfig := withConfig(remoteBaseConfig, map[string]any{"role": "client", "reconnect": true, "allowStartDisconnected": args.AllowStartDisconnected})
	if errs := validateSerialGraphRemoteConfig("a-remote-server", serverConfig); len(errs) > 0 {
		return nil, errors.New(errors.CodeInvalid, strings.Join(errs, "; "))
	}
	if errs := validateSerialGraphRemoteConfig("b-remote-client", clientConfig); len(errs) > 0 {
		return nil, errors.New(errors.CodeInvalid, strings.Join(errs, "; "))
	}
	nodes := []serialGraphNodeArg{
		{ID: "client-generator", Type: "serial.script.generator", Position: serialGraphPositionArg{X: 20, Y: 240}, Config: withConfig(scriptDefaultsForTemplate(), map[string]any{"script": "output.text(\"client -> server\\r\\n\", \"utf-8\")", "autoRun": true, "intervalMs": 1000})},
		{ID: "a-remote-server", Type: "serial.remote", Position: serialGraphPositionArg{X: 260, Y: 60}, Config: serverConfig},
		{ID: "b-remote-client", Type: "serial.remote", Position: serialGraphPositionArg{X: 260, Y: 240}, Config: clientConfig},
		{ID: "server-tap", Type: "serial.tap", Position: serialGraphPositionArg{X: 500, Y: 60}, Config: map[string]any{}},
		{ID: "client-tap", Type: "serial.tap", Position: serialGraphPositionArg{X: 500, Y: 240}, Config: map[string]any{}},
		{ID: "server-monitor", Type: "serial.monitor", Position: serialGraphPositionArg{X: 740, Y: 60}, Config: map[string]any{"displayMode": "hex"}},
		{ID: "client-monitor", Type: "serial.monitor", Position: serialGraphPositionArg{X: 740, Y: 240}, Config: map[string]any{"displayMode": "hex"}},
	}
	edges := []serialGraphEdgeArg{
		{ID: "edge-client-generator-remote", Source: "client-generator", SourceHandle: "out", Target: "b-remote-client", TargetHandle: "tx"},
		{ID: "edge-server-remote-tap", Source: "a-remote-server", SourceHandle: "rx", Target: "server-tap", TargetHandle: "in"},
		{ID: "edge-server-tap-monitor", Source: "server-tap", SourceHandle: "out", Target: "server-monitor", TargetHandle: "in"},
		{ID: "edge-client-remote-tap", Source: "b-remote-client", SourceHandle: "rx", Target: "client-tap", TargetHandle: "in"},
		{ID: "edge-client-tap-monitor", Source: "client-tap", SourceHandle: "out", Target: "client-monitor", TargetHandle: "in"},
	}

	return map[string]any{
		"id":          serialGraphRemoteTemplateID,
		"graph_id":    graphID,
		"title":       "Remote serial raw TCP",
		"description": "Read-only MCP template demonstrating an in-graph raw TCP server/client remote serial loopback with endpoint buffers and monitor branches.",
		"nodes":       nodes,
		"edges":       edges,
		"usage": map[string]any{
			"validate":             "Call serial_graph_validate with the nodes and edges from this response before starting resources.",
			"start":                map[string]any{"tool": "serial_graph_start", "arguments": serialGraphStartArgs{ID: graphID, Nodes: nodes, Edges: edges}},
			"status_example":       map[string]any{"tool": "serial_graph_status", "arguments": map[string]any{"graph_id": graphID}},
			"send_example":         map[string]any{"tool": "serial_graph_send", "arguments": map[string]any{"graph_id": graphID, "node_id": "a-remote-server", "content": "server -> client\r\n", "mode": "ascii"}},
			"query_buffer_example": map[string]any{"tool": "serial_graph_query_node_buffer", "arguments": map[string]any{"graph_id": graphID, "node_id": "b-remote-client", "offset": 0, "length": 256}},
			"query_frames_example": map[string]any{"tool": "serial_graph_query_node_frames", "arguments": map[string]any{"graph_id": graphID, "node_id": "client-monitor", "offset": 0, "limit": 20}},
			"security":             "raw TCP has no built-in authentication or encryption; expose only on trusted networks or wrap with SSH/VPN/TLS outside PortWeave.",
			"operation_log_state":  "MCP supports graph config/templates plus backend buffers and frames; frontend UI operation-log state is not stored in the backend.",
		},
	}, nil
}

func validateSerialGraph(graph serialGraphValidateArgs) []string {
	var errs []string
	providers := serialGraphProviderMap()
	nodes := make(map[string]serialGraphNodeArg, len(graph.Nodes))

	for _, node := range graph.Nodes {
		if node.ID == "" {
			errs = append(errs, "node id must not be empty")
			continue
		}
		if _, exists := nodes[node.ID]; exists {
			errs = append(errs, "duplicate node id: "+node.ID)
			continue
		}
		if _, ok := providers[node.Type]; !ok {
			errs = append(errs, "provider not found: "+node.Type)
		}
		if node.Type == "serial.remote" {
			errs = append(errs, validateSerialGraphRemoteConfig(node.ID, node.Config)...)
		}
		nodes[node.ID] = node
	}

	errs = append(errs, validateSerialGraphResourceOwners(graph.Nodes, providers)...)
	seenEdges := make(map[string]bool, len(graph.Edges))
	for _, edge := range graph.Edges {
		edge = normalizeSerialGraphEdge(edge)
		if edge.ID == "" {
			errs = append(errs, "edge id must not be empty")
			continue
		}
		if seenEdges[edge.ID] {
			errs = append(errs, "duplicate edge id: "+edge.ID)
			continue
		}
		seenEdges[edge.ID] = true
		result := validateSerialGraphConnection(graph, edge, nodes, providers)
		for _, err := range result {
			errs = append(errs, edge.ID+": "+err)
		}
	}
	return errs
}

func serialGraphStartRequest(args serialGraphStartArgs) serial.SerialGraphStartRequest {
	nodes := make([]serial.SerialGraphNodeSpec, 0, len(args.Nodes))
	for _, node := range args.Nodes {
		nodes = append(nodes, serial.SerialGraphNodeSpec{
			ID:     node.ID,
			Type:   node.Type,
			Config: node.Config,
			Position: serial.SerialGraphPosition{
				X: node.Position.X,
				Y: node.Position.Y,
			},
		})
	}
	edges := make([]serial.SerialGraphEdgeSpec, 0, len(args.Edges))
	for _, edge := range args.Edges {
		edge = normalizeSerialGraphEdge(edge)
		edges = append(edges, serial.SerialGraphEdgeSpec{
			ID:           edge.ID,
			Source:       edge.Source,
			SourceHandle: edge.SourceHandle,
			Target:       edge.Target,
			TargetHandle: edge.TargetHandle,
		})
	}
	return serial.SerialGraphStartRequest{
		ID:    args.ID,
		Nodes: nodes,
		Edges: edges,
	}
}

func validateSerialGraphConnection(
	graph serialGraphValidateArgs,
	draft serialGraphEdgeArg,
	nodes map[string]serialGraphNodeArg,
	providers map[string]serialGraphProvider,
) []string {
	var errs []string
	sourceNode, sourceOK := nodes[draft.Source]
	targetNode, targetOK := nodes[draft.Target]
	if !sourceOK {
		errs = append(errs, "source node not found: "+draft.Source)
	}
	if !targetOK {
		errs = append(errs, "target node not found: "+draft.Target)
	}
	if !sourceOK || !targetOK {
		return errs
	}
	if sourceNode.ID == targetNode.ID {
		errs = append(errs, "node cannot connect to itself")
	}

	sourceProvider, sourceProviderOK := providers[sourceNode.Type]
	targetProvider, targetProviderOK := providers[targetNode.Type]
	if !sourceProviderOK {
		errs = append(errs, "provider not found: "+sourceNode.Type)
	}
	if !targetProviderOK {
		errs = append(errs, "provider not found: "+targetNode.Type)
	}
	if !sourceProviderOK || !targetProviderOK {
		return errs
	}

	sourcePort, sourcePortOK := outputPort(sourceProvider, draft.SourceHandle)
	targetPort, targetPortOK := inputPort(targetProvider, draft.TargetHandle)
	if !sourcePortOK {
		errs = append(errs, "output port not found: "+sourceNode.Type+"."+draft.SourceHandle)
	}
	if !targetPortOK {
		errs = append(errs, "input port not found: "+targetNode.Type+"."+draft.TargetHandle)
	}
	if !sourcePortOK || !targetPortOK {
		return errs
	}

	if sourcePort.Kind != targetPort.Kind {
		errs = append(errs, "incompatible port kinds: "+sourcePort.Kind+" -> "+targetPort.Kind)
	}

	otherEdges := serialGraphOtherEdges(graph.Edges, draft.ID)
	if !targetPort.Multiple {
		for _, edge := range otherEdges {
			if edge.Target == draft.Target && edge.TargetHandle == draft.TargetHandle {
				errs = append(errs, "input already connected: "+targetNode.ID+"."+targetPort.ID)
				break
			}
		}
	}

	fanOutAllowed := sourcePort.Multiple || sourceProvider.Type == "serial.tap" || sourceProvider.Type == "serial.tee"
	if !fanOutAllowed {
		for _, edge := range otherEdges {
			if edge.Source == draft.Source && edge.SourceHandle == draft.SourceHandle {
				errs = append(errs, "fan-out requires a tap node: "+sourceNode.ID+"."+sourcePort.ID)
				break
			}
		}
	}

	if !serialGraphEdgeTargetsTerminalProtocolInput(nodes, draft) && serialGraphCreatesCycle(draft, otherEdges, nodes) {
		errs = append(errs, "directed cycle not allowed")
	}
	return errs
}

func validateSerialGraphResourceOwners(nodes []serialGraphNodeArg, providers map[string]serialGraphProvider) []string {
	var errs []string
	used := map[string]string{}
	for _, node := range nodes {
		provider, ok := providers[node.Type]
		if !ok || !provider.ResourceOwner {
			continue
		}
		if node.Type == "serial.remote" {
			endpoint, ok := serialGraphRemoteEndpoint(node.Config)
			if !ok {
				continue
			}
			role := serialGraphStringConfig(node.Config, "role", "client")
			resourceKey := "remote:" + endpoint + ":" + role
			if previous, exists := used[resourceKey]; exists {
				errs = append(errs, fmt.Sprintf("resource remote endpoint duplicated: %s (%s, %s)", endpoint, previous, node.ID))
			} else {
				used[resourceKey] = node.ID
			}
			continue
		}
		for _, key := range provider.ResourceKeys {
			value := strings.TrimSpace(fmt.Sprint(node.Config[key]))
			if value == "" || value == "<nil>" {
				continue
			}
			resourceKey := key + ":" + value
			if previous, exists := used[resourceKey]; exists {
				errs = append(errs, fmt.Sprintf("resource port duplicated: %s (%s, %s)", value, previous, node.ID))
			} else {
				used[resourceKey] = node.ID
			}
		}
	}
	return errs
}

func validateSerialGraphRemoteConfig(nodeID string, config map[string]any) []string {
	var errs []string
	protocol := serialGraphStringConfig(config, "protocol", "raw-tcp")
	role := serialGraphStringConfig(config, "role", "client")
	host := serialGraphStringConfig(config, "host", "")
	port, portOK := serialGraphStrictIntConfig(config, "port", 3001)
	connectTimeoutMs, connectTimeoutOK := serialGraphStrictIntConfig(config, "connectTimeoutMs", 3000)
	writeTimeoutMs, writeTimeoutOK := serialGraphStrictIntConfig(config, "writeTimeoutMs", 3000)
	reconnectIntervalMs, reconnectIntervalOK := serialGraphStrictIntConfig(config, "reconnectIntervalMs", 1000)
	readBufKB, readBufOK := serialGraphStrictIntConfig(config, "readBufKB", 32)
	prefix := nodeID + ": "
	if protocol != "raw-tcp" {
		errs = append(errs, prefix+"unsupported protocol: "+protocol)
	}
	if role != "client" && role != "server" {
		errs = append(errs, prefix+"unsupported role: "+role)
	}
	if host == "" {
		errs = append(errs, prefix+"remote host required")
	}
	if strings.Contains(host, "://") {
		errs = append(errs, prefix+"remote host must not include URL scheme")
	}
	if !portOK {
		errs = append(errs, prefix+"remote port must be an integer")
	} else if port < 1 || port > 65535 {
		errs = append(errs, fmt.Sprintf("%sremote port out of range: %d", prefix, port))
	}
	if !connectTimeoutOK {
		errs = append(errs, prefix+"connectTimeoutMs must be an integer")
	} else if connectTimeoutMs <= 0 {
		errs = append(errs, prefix+"connectTimeoutMs must be positive")
	} else if connectTimeoutMs < 100 || connectTimeoutMs > 60000 {
		errs = append(errs, prefix+"connectTimeoutMs out of range")
	}
	if !writeTimeoutOK {
		errs = append(errs, prefix+"writeTimeoutMs must be an integer")
	} else if writeTimeoutMs <= 0 {
		errs = append(errs, prefix+"writeTimeoutMs must be positive")
	} else if writeTimeoutMs < 100 || writeTimeoutMs > 60000 {
		errs = append(errs, prefix+"writeTimeoutMs out of range")
	}
	if !reconnectIntervalOK {
		errs = append(errs, prefix+"reconnectIntervalMs must be an integer")
	} else if reconnectIntervalMs <= 0 {
		errs = append(errs, prefix+"reconnectIntervalMs must be positive")
	} else if reconnectIntervalMs < 100 || reconnectIntervalMs > 60000 {
		errs = append(errs, prefix+"reconnectIntervalMs out of range")
	}
	if !readBufOK {
		errs = append(errs, prefix+"readBufKB must be an integer")
	} else if readBufKB <= 0 {
		errs = append(errs, prefix+"readBufKB must be positive")
	} else if readBufKB > 1024 {
		errs = append(errs, prefix+"readBufKB out of range")
	}
	return errs
}

func serialGraphRemoteEndpoint(config map[string]any) (string, bool) {
	host := serialGraphStringConfig(config, "host", "")
	port, portOK := serialGraphStrictIntConfig(config, "port", 3001)
	if host == "" || strings.Contains(host, "://") || !portOK || port < 1 || port > 65535 {
		return "", false
	}
	return net.JoinHostPort(strings.ToLower(host), strconv.Itoa(port)), true
}

func serialGraphStringConfig(config map[string]any, key string, fallback string) string {
	if config == nil {
		return fallback
	}
	value, ok := config[key]
	if !ok || value == nil {
		return fallback
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func serialGraphStrictIntConfig(config map[string]any, key string, fallback int) (int, bool) {
	if config == nil {
		return fallback, true
	}
	value, ok := config[key]
	if !ok {
		return fallback, true
	}
	if value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		parsed := int(typed)
		return parsed, float64(parsed) == typed
	case float32:
		parsed := int(typed)
		return parsed, float32(parsed) == typed
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func serialGraphProviderMap() map[string]serialGraphProvider {
	providers := serialGraphProviders()
	out := make(map[string]serialGraphProvider, len(providers))
	for _, provider := range providers {
		out[provider.Type] = provider
	}
	return out
}

func inputPort(provider serialGraphProvider, id string) (serialGraphPortSpec, bool) {
	for _, port := range provider.Inputs {
		if port.ID == id {
			return port, true
		}
	}
	return serialGraphPortSpec{}, false
}

func outputPort(provider serialGraphProvider, id string) (serialGraphPortSpec, bool) {
	for _, port := range provider.Outputs {
		if port.ID == id {
			return port, true
		}
	}
	return serialGraphPortSpec{}, false
}

func serialGraphOtherEdges(edges []serialGraphEdgeArg, id string) []serialGraphEdgeArg {
	out := make([]serialGraphEdgeArg, 0, len(edges))
	for _, edge := range edges {
		edge = normalizeSerialGraphEdge(edge)
		if edge.ID != id {
			out = append(out, edge)
		}
	}
	return out
}

func normalizeSerialGraphEdge(edge serialGraphEdgeArg) serialGraphEdgeArg {
	if edge.SourceHandle == "" {
		edge.SourceHandle = edge.SourceHandleCamel
	}
	if edge.TargetHandle == "" {
		edge.TargetHandle = edge.TargetHandleCamel
	}
	return edge
}

func serialGraphCreatesCycle(draft serialGraphEdgeArg, edges []serialGraphEdgeArg, nodes map[string]serialGraphNodeArg) bool {
	visited := map[string]bool{}
	stack := []string{draft.Target}
	for len(stack) > 0 {
		nodeID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[nodeID] {
			continue
		}
		if nodeID == draft.Source {
			return true
		}
		visited[nodeID] = true
		for _, edge := range edges {
			if serialGraphEdgeTargetsTerminalProtocolInput(nodes, edge) {
				continue
			}
			if edge.Source == nodeID {
				stack = append(stack, edge.Target)
			}
		}
	}
	return false
}

func serialGraphEdgeTargetsTerminalProtocolInput(nodes map[string]serialGraphNodeArg, edge serialGraphEdgeArg) bool {
	if edge.TargetHandle != "rx" {
		return false
	}
	target, ok := nodes[edge.Target]
	if !ok {
		return false
	}
	return target.Type == "serial.modbus.master" || target.Type == "serial.fecbus.master"
}

func withConfig(base map[string]any, patch map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(patch))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range patch {
		out[key] = value
	}
	return out
}

func dataModelSnapshot(args modbusDataModelArgs) mb.DataModelSnapshot {
	return mb.DataModelSnapshot{
		Coils:            boolPoints(args.Coils),
		DiscreteInputs:   boolPoints(args.DiscreteInputs),
		InputRegisters:   registerPoints(args.InputRegisters),
		HoldingRegisters: registerPoints(args.HoldingRegisters),
	}
}

func boolPoints(args []modbusBoolPointArg) []mb.BoolPoint {
	out := make([]mb.BoolPoint, 0, len(args))
	for _, item := range args {
		out = append(out, mb.BoolPoint{
			Address: uint16(clampInt(item.Address, 0, 65535)),
			Value:   item.Value,
		})
	}
	return out
}

func registerPoints(args []modbusRegisterPointArg) []mb.RegisterPoint {
	out := make([]mb.RegisterPoint, 0, len(args))
	for _, item := range args {
		out = append(out, mb.RegisterPoint{
			Address: uint16(clampInt(item.Address, 0, 65535)),
			Value:   uint16(clampInt(item.Value, 0, 65535)),
		})
	}
	return out
}

func registerMappings(args []modbusRegisterMappingArg) []mb.RegisterMapping {
	out := make([]mb.RegisterMapping, 0, len(args))
	for _, item := range args {
		out = append(out, mb.RegisterMapping{
			Address:       uint16(clampInt(item.Address, 0, 65535)),
			DataType:      mb.DataType(item.DataType),
			WordOrder:     mb.WordOrder(item.WordOrder),
			Length:        uint16(clampInt(item.Length, 0, 65535)),
			ScalingFactor: item.ScalingFactor,
			Comment:       item.Comment,
			GroupEnd:      item.GroupEnd,
		})
	}
	return out
}

func fecbusSlaveState(args fecbusSlaveStateArgs) fb.SlaveState {
	status := args.StatusCode
	if status == 0 {
		status = int(fb.StatusReceivedOK)
	}
	return fb.SlaveState{
		Address:          byte(clampInt(args.Address, 1, 63)),
		DefaultStatus:    fb.StatusCode(clampInt(status, 0, 255)),
		AutoStatusAnswer: args.AutoStatusAnswer,
		AcceptBroadcast:  args.AcceptBroadcast,
	}
}

func fecbusSlaveUnits(args fecbusSlaveStateArgs) []fb.SlaveUnitState {
	if len(args.Units) == 0 {
		return nil
	}
	out := make([]fb.SlaveUnitState, 0, len(args.Units))
	for _, unit := range args.Units {
		status := unit.StatusCode
		if status == 0 {
			status = int(fb.StatusReceivedOK)
		}
		out = append(out, fb.SlaveUnitState{
			Address:          byte(clampInt(unit.Address, 1, 63)),
			DefaultStatus:    fb.StatusCode(clampInt(status, 0, 255)),
			AutoStatusAnswer: unit.AutoStatusAnswer,
			AcceptBroadcast:  unit.AcceptBroadcast,
		})
	}
	return out
}

func fecbusSlaveUnit(args fecbusSlaveUnitArgs) fb.SlaveUnitState {
	status := args.StatusCode
	if status == 0 {
		status = int(fb.StatusReceivedOK)
	}
	return fb.SlaveUnitState{
		Address:          byte(clampInt(args.Address, 1, 63)),
		DefaultStatus:    fb.StatusCode(clampInt(status, 0, 255)),
		AutoStatusAnswer: args.AutoStatusAnswer,
		AcceptBroadcast:  args.AcceptBroadcast,
	}
}

func fecbusCustomFunctions(args []fecbusCustomFunctionArg) []fb.CustomFunctionDefinition {
	out := make([]fb.CustomFunctionDefinition, 0, len(args))
	for _, item := range args {
		fields := make([]fb.CustomDataFieldDefinition, 0, len(item.Fields))
		for _, field := range item.Fields {
			enum := make(map[byte]string, len(field.Enum))
			for raw, label := range field.Enum {
				key := clampInt(parseFlexibleInt(raw), 0, 255)
				enum[byte(key)] = label
			}
			fields = append(fields, fb.CustomDataFieldDefinition{
				Key:     field.Key,
				Label:   field.Label,
				Offset:  field.Offset,
				Length:  field.Length,
				Type:    field.Type,
				Endian:  field.Endian,
				Enum:    enum,
				Meaning: field.Meaning,
			})
		}
		out = append(out, fb.CustomFunctionDefinition{
			Code:        fb.FunctionCode(clampInt(item.Code, 0, 255)),
			Name:        item.Name,
			Description: item.Description,
			Direction:   item.Direction,
			Answer:      item.Answer,
			Fields:      fields,
		})
	}
	return out
}

func parseFlexibleInt(value string) int {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		parsed, err := strconv.ParseInt(value[2:], 16, 32)
		if err == nil {
			return int(parsed)
		}
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func originGuard(next http.Handler, cfg config.MCPConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && !isAllowedOrigin(origin, cfg) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string, cfg config.MCPConfig) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		host = parsed.Host
		portText = ""
	}
	if portText == "" {
		return false
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port != cfg.Port {
		return false
	}
	if strings.EqualFold(host, cfg.Host) {
		return true
	}
	if !cfg.AllowLocalOrigins {
		return false
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func listenAddress(cfg config.MCPConfig) (string, error) {
	if cfg.Host == "" {
		return "", errors.New(errors.CodeInvalid, "mcp host must not be empty")
	}
	if !isLocalHost(cfg.Host) {
		return "", errors.New(errors.CodeInvalid, "mcp host must be localhost or a loopback address")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return "", errors.New(errors.CodeInvalid, "mcp port must be between 1 and 65535")
	}
	return net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), nil
}

func isLocalHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizedPath(path string) string {
	if path == "" {
		return "/mcp"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func shutdownHTTPServer(ctx context.Context, server *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("shutdown mcp http server: %w", err)
	}
	return nil
}
