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
