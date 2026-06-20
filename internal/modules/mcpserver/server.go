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

	"github.com/suyue/mocktrue/internal/core/config"
	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/modules/serial"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	mb "github.com/suyue/mocktrue/internal/modules/serial/modbus"
	"github.com/suyue/mocktrue/internal/modules/serial/monitor"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
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

func newMCPServer(serialService SerialRuntime) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "mocktrue", Version: "0.1.0"}, nil)
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
