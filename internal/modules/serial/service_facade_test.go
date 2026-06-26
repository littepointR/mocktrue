package serial

import (
	"context"
	"testing"

	"github.com/littepointR/mocktrue/internal/core/eventbus"
	fb "github.com/littepointR/mocktrue/internal/modules/serial/fecbus"
	mb "github.com/littepointR/mocktrue/internal/modules/serial/modbus"
)

func TestServiceProtocolFacadeEmptyState(t *testing.T) {
	t.Parallel()
	svc := NewService(eventbus.New())

	if got := svc.ServiceName(); got != "serial" {
		t.Fatalf("ServiceName = %q, want serial", got)
	}
	if got := svc.ListModbusSessions(); len(got) != 0 {
		t.Fatalf("ListModbusSessions = %#v, want empty", got)
	}
	if got := svc.ListFecbusSessions(); len(got) != 0 {
		t.Fatalf("ListFecbusSessions = %#v, want empty", got)
	}
	if got := svc.ListMonitors(); len(got) != 0 {
		t.Fatalf("ListMonitors = %#v, want empty", got)
	}
	if err := svc.DeleteMonitor("missing"); err == nil {
		t.Fatalf("DeleteMonitor must reject missing monitor")
	}
	if err := svc.ClearMonitorFrames("missing"); err == nil {
		t.Fatalf("ClearMonitorFrames must reject missing monitor")
	}

	if _, err := svc.OpenModbusSession(context.Background(), mb.OpenSessionRequest{}); err == nil {
		t.Fatalf("OpenModbusSession must reject empty port")
	}
	if _, err := svc.OpenFecbusSession(context.Background(), fb.OpenSessionRequest{}); err == nil {
		t.Fatalf("OpenFecbusSession must reject empty port")
	}

	if err := svc.CloseModbusSession("missing"); err == nil {
		t.Fatalf("CloseModbusSession must reject missing session")
	}
	if _, err := svc.ModbusMasterRequest(mb.MasterRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("ModbusMasterRequest must reject missing session")
	}
	if _, err := svc.StartModbusSlave(mb.StartSlaveRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("StartModbusSlave must reject missing session")
	}
	if err := svc.StopModbusSlave("missing"); err == nil {
		t.Fatalf("StopModbusSlave must reject missing session")
	}
	if err := svc.UpdateModbusSlaveData("missing", mb.DataModelSnapshot{}); err == nil {
		t.Fatalf("UpdateModbusSlaveData must reject missing session")
	}
	if err := svc.AddModbusSlaveUnit("missing", mb.SlaveUnitSnapshot{UnitID: 1}); err == nil {
		t.Fatalf("AddModbusSlaveUnit must reject missing session")
	}
	if err := svc.RemoveModbusSlaveUnit("missing", 1); err == nil {
		t.Fatalf("RemoveModbusSlaveUnit must reject missing session")
	}
	if _, err := svc.ListModbusSlaveUnits("missing"); err == nil {
		t.Fatalf("ListModbusSlaveUnits must reject missing session")
	}
	if err := svc.UpdateModbusSlaveUnitData("missing", 1, mb.DataModelSnapshot{}); err == nil {
		t.Fatalf("UpdateModbusSlaveUnitData must reject missing session")
	}
	if _, err := svc.ModbusScanUnitIDs(mb.UnitScanRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("ModbusScanUnitIDs must reject missing session")
	}
	if _, err := svc.ModbusReadRegisters(mb.RegisterReadRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("ModbusReadRegisters must reject missing session")
	}
	if _, err := svc.ModbusScanRegisters(mb.RegisterScanRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("ModbusScanRegisters must reject missing session")
	}

	if err := svc.CloseFecbusSession("missing"); err == nil {
		t.Fatalf("CloseFecbusSession must reject missing session")
	}
	if _, err := svc.FecbusSendRequest(fb.SendRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("FecbusSendRequest must reject missing session")
	}
	if _, err := svc.StartFecbusSlave(fb.StartSlaveRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("StartFecbusSlave must reject missing session")
	}
	if err := svc.StopFecbusSlave("missing"); err == nil {
		t.Fatalf("StopFecbusSlave must reject missing session")
	}
	if err := svc.UpdateFecbusSlaveState("missing", fb.SlaveState{}); err == nil {
		t.Fatalf("UpdateFecbusSlaveState must reject missing session")
	}
	if err := svc.AddFecbusSlaveUnit("missing", fb.SlaveUnitState{Address: 1}); err == nil {
		t.Fatalf("AddFecbusSlaveUnit must reject missing session")
	}
	if err := svc.RemoveFecbusSlaveUnit("missing", 1); err == nil {
		t.Fatalf("RemoveFecbusSlaveUnit must reject missing session")
	}
	if _, err := svc.ListFecbusSlaveUnits("missing"); err == nil {
		t.Fatalf("ListFecbusSlaveUnits must reject missing session")
	}
	if _, err := svc.QueryFecbusFrames(fb.QueryRequest{SessionID: "missing"}); err == nil {
		t.Fatalf("QueryFecbusFrames must reject missing session")
	}
	if err := svc.ClearFecbusFrames("missing"); err == nil {
		t.Fatalf("ClearFecbusFrames must reject missing session")
	}
}

func TestServiceVirtualFacadeValidationAndEmptyLists(t *testing.T) {
	t.Parallel()
	svc := NewService(eventbus.New())

	if got := svc.ListVirtualPorts(); len(got) != 0 {
		t.Fatalf("ListVirtualPorts = %#v, want empty", got)
	}
	if got := svc.ListVirtualPairs(); len(got) != 0 {
		t.Fatalf("ListVirtualPairs = %#v, want empty", got)
	}
	if got := svc.ListBridges(); len(got) != 0 {
		t.Fatalf("ListBridges = %#v, want empty", got)
	}

	if err := svc.DeleteVirtualPair(""); err == nil {
		t.Fatalf("DeleteVirtualPair must reject empty id")
	}
	if err := svc.DeleteBridge(""); err == nil {
		t.Fatalf("DeleteBridge must reject empty id")
	}
	if _, err := svc.CreateBridge("", "a", "b", 115200); err == nil {
		t.Fatalf("CreateBridge must reject empty id")
	}
	if _, err := svc.CreateBridge("bridge", "", "b", 115200); err == nil {
		t.Fatalf("CreateBridge must reject empty ports")
	}
	if _, err := svc.CreateBridge("bridge", "a", "a", 115200); err == nil {
		t.Fatalf("CreateBridge must reject self bridge")
	}
}
