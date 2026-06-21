package serial

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	mb "github.com/littepointR/mocktrue/internal/modules/serial/modbus"
)

const (
	serialScriptDefaultTimeout       = 50 * time.Millisecond
	serialScriptDefaultMaxOutputSize = 64 * 1024
	serialScriptDefaultMaxStateSize  = 256 * 1024
)

type serialScriptRunInput struct {
	Data []byte
}

type serialScriptRunResult struct {
	Output []byte
	Drop   bool
	Fields []serialScriptField
	Errors []string
}

type serialScriptField struct {
	Name    string
	Value   any
	Display string
}

type serialScriptState struct {
	mu      sync.Mutex
	values  map[string]any
	maxSize int
}

func newSerialScriptState(maxSize int) *serialScriptState {
	if maxSize <= 0 {
		maxSize = serialScriptDefaultMaxStateSize
	}
	return &serialScriptState{
		values:  make(map[string]any),
		maxSize: maxSize,
	}
}

func (s *serialScriptState) get(key string) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned, err := jsonCompatibleClone(s.values[key])
	if err != nil {
		return nil
	}
	return cloned
}

func (s *serialScriptState) set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned, err := jsonCompatibleClone(value)
	if err != nil {
		return err
	}
	next := make(map[string]any, len(s.values)+1)
	for k, v := range s.values {
		next[k] = v
	}
	next[key] = cloned
	if size, err := jsonEncodedSize(next); err != nil {
		return err
	} else if size > s.maxSize {
		return fmt.Errorf("script state exceeds %d bytes", s.maxSize)
	}
	s.values = next
	return nil
}

func (s *serialScriptState) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
}

type serialScriptRuntime struct {
	script        string
	timeout       time.Duration
	maxOutputSize int
	encoding      string
	state         *serialScriptState
	nodeType      string
}

func newSerialScriptRuntime(config map[string]any) *serialScriptRuntime {
	return newSerialScriptRuntimeForNode("serial.script.transform", config)
}

func newSerialScriptRuntimeForNode(nodeType string, config map[string]any) *serialScriptRuntime {
	timeoutMs := graphIntConfig(config, "timeoutMs", int(serialScriptDefaultTimeout/time.Millisecond))
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = serialScriptDefaultTimeout
	}
	maxOutputSize := graphIntConfig(config, "maxOutputBytes", serialScriptDefaultMaxOutputSize)
	if maxOutputSize <= 0 {
		maxOutputSize = serialScriptDefaultMaxOutputSize
	}
	return &serialScriptRuntime{
		script:        graphStringConfigRaw(config, "script"),
		timeout:       timeout,
		maxOutputSize: maxOutputSize,
		encoding:      graphStringConfigWithDefault(config, "encoding", "utf-8"),
		state:         newSerialScriptState(graphIntConfig(config, "maxStateBytes", serialScriptDefaultMaxStateSize)),
		nodeType:      nodeType,
	}
}

func (r *serialScriptRuntime) run(input serialScriptRunInput) (serialScriptRunResult, error) {
	if strings.TrimSpace(r.script) == "" {
		return serialScriptRunResult{}, fmt.Errorf("script must not be empty")
	}

	vm := goja.New()
	result := serialScriptRunResult{}
	output := make([]byte, 0)

	appendOutput := func(data []byte) error {
		if len(data) == 0 {
			return nil
		}
		if len(output)+len(data) > r.maxOutputSize {
			return fmt.Errorf("script output exceeds %d bytes", r.maxOutputSize)
		}
		output = append(output, data...)
		return nil
	}
	throw := func(err error) {
		panic(vm.NewGoError(err))
	}

	if r.exposesInputAPI() {
		inputObj := vm.NewObject()
		if err := inputObj.Set("bytes", func() []int {
			return byteInts(input.Data)
		}); err != nil {
			return serialScriptRunResult{}, err
		}
		if err := inputObj.Set("hex", func() string {
			return formatHexBytes(input.Data)
		}); err != nil {
			return serialScriptRunResult{}, err
		}
		if err := inputObj.Set("text", func(call goja.FunctionCall) goja.Value {
			encoding := r.encoding
			if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
				encoding = call.Argument(0).String()
			}
			text, err := decodeSerialText(input.Data, encoding)
			if err != nil {
				throw(err)
			}
			return vm.ToValue(text)
		}); err != nil {
			return serialScriptRunResult{}, err
		}
		if err := vm.Set("input", inputObj); err != nil {
			return serialScriptRunResult{}, err
		}
	}

	if r.exposesOutputAPI() {
		outputObj := vm.NewObject()
		if err := outputObj.Set("bytes", func(call goja.FunctionCall) goja.Value {
			data, err := gojaValueToBytes(call.Argument(0))
			if err != nil {
				throw(err)
			}
			if err := appendOutput(data); err != nil {
				throw(err)
			}
			return goja.Undefined()
		}); err != nil {
			return serialScriptRunResult{}, err
		}
		if err := outputObj.Set("hex", func(call goja.FunctionCall) goja.Value {
			data, err := decodeHexContent(call.Argument(0).String())
			if err != nil {
				throw(err)
			}
			if err := appendOutput(data); err != nil {
				throw(err)
			}
			return goja.Undefined()
		}); err != nil {
			return serialScriptRunResult{}, err
		}
		if err := outputObj.Set("text", func(call goja.FunctionCall) goja.Value {
			encoding := r.encoding
			if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) && !goja.IsNull(call.Argument(1)) {
				encoding = call.Argument(1).String()
			}
			data, err := encodeSerialText(call.Argument(0).String(), encoding)
			if err != nil {
				throw(err)
			}
			if err := appendOutput(data); err != nil {
				throw(err)
			}
			return goja.Undefined()
		}); err != nil {
			return serialScriptRunResult{}, err
		}
		if err := vm.Set("output", outputObj); err != nil {
			return serialScriptRunResult{}, err
		}
	}

	if err := vm.Set("drop", func() {
		result.Drop = true
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := vm.Set("field", func(call goja.FunctionCall) goja.Value {
		field := serialScriptField{
			Name:  call.Argument(0).String(),
			Value: call.Argument(1).Export(),
		}
		if len(call.Arguments) > 2 {
			field.Display = call.Argument(2).String()
		}
		result.Fields = append(result.Fields, field)
		return goja.Undefined()
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := vm.Set("error", func(call goja.FunctionCall) goja.Value {
		result.Errors = append(result.Errors, call.Argument(0).String())
		return goja.Undefined()
	}); err != nil {
		return serialScriptRunResult{}, err
	}

	stateObj := vm.NewObject()
	if err := stateObj.Set("get", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(r.state.get(call.Argument(0).String()))
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := stateObj.Set("set", func(call goja.FunctionCall) goja.Value {
		if err := r.state.set(call.Argument(0).String(), call.Argument(1).Export()); err != nil {
			throw(err)
		}
		return goja.Undefined()
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := stateObj.Set("delete", func(call goja.FunctionCall) goja.Value {
		r.state.delete(call.Argument(0).String())
		return goja.Undefined()
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := vm.Set("state", stateObj); err != nil {
		return serialScriptRunResult{}, err
	}

	if err := vm.Set("crc16", func(call goja.FunctionCall) goja.Value {
		data, err := gojaValueToBytes(call.Argument(0))
		if err != nil {
			throw(err)
		}
		return vm.ToValue(mb.CRC16(data))
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := vm.Set("sum8", func(call goja.FunctionCall) goja.Value {
		data, err := gojaValueToBytes(call.Argument(0))
		if err != nil {
			throw(err)
		}
		var sum byte
		for _, b := range data {
			sum += b
		}
		return vm.ToValue(sum)
	}); err != nil {
		return serialScriptRunResult{}, err
	}
	if err := vm.Set("now", func() int64 {
		return time.Now().UnixNano()
	}); err != nil {
		return serialScriptRunResult{}, err
	}

	done := make(chan error, 1)
	go func() {
		_, err := vm.RunString(r.script)
		done <- err
	}()

	timer := time.NewTimer(r.timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			return serialScriptRunResult{}, fmt.Errorf("script error: %w", err)
		}
	case <-timer.C:
		vm.Interrupt("timeout")
		return serialScriptRunResult{}, fmt.Errorf("script execution timeout")
	}

	if result.Drop {
		output = nil
	}
	result.Output = append([]byte(nil), output...)
	return result, nil
}

func (r *serialScriptRuntime) exposesInputAPI() bool {
	return r.nodeType != "serial.script.generator"
}

func (r *serialScriptRuntime) exposesOutputAPI() bool {
	return r.nodeType != "serial.script.analyzer"
}

func byteInts(data []byte) []int {
	out := make([]int, len(data))
	for i, b := range data {
		out[i] = int(b)
	}
	return out
}

func gojaValueToBytes(value goja.Value) ([]byte, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	switch exported := value.Export().(type) {
	case []byte:
		return append([]byte(nil), exported...), nil
	case []int:
		out := make([]byte, len(exported))
		for i, n := range exported {
			if n < 0 || n > 255 {
				return nil, fmt.Errorf("byte value out of range: %d", n)
			}
			out[i] = byte(n)
		}
		return out, nil
	case []any:
		out := make([]byte, len(exported))
		for i, item := range exported {
			n, ok := numericByte(item)
			if !ok {
				return nil, fmt.Errorf("byte value at index %d is not numeric", i)
			}
			out[i] = byte(n)
		}
		return out, nil
	case string:
		return []byte(exported), nil
	default:
		return nil, fmt.Errorf("unsupported byte value: %T", exported)
	}
}

func numericByte(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, v >= 0 && v <= 255
	case int8:
		return int(v), v >= 0
	case int16:
		return int(v), v >= 0 && v <= 255
	case int32:
		return int(v), v >= 0 && v <= 255
	case int64:
		return int(v), v >= 0 && v <= 255
	case uint:
		return int(v), v <= 255
	case uint8:
		return int(v), true
	case uint16:
		return int(v), v <= 255
	case uint32:
		return int(v), v <= 255
	case uint64:
		return int(v), v <= 255
	case float32:
		n := int(v)
		return n, float32(n) == v && n >= 0 && n <= 255
	case float64:
		n := int(v)
		return n, float64(n) == v && n >= 0 && n <= 255
	default:
		return 0, false
	}
}

func jsonEncodedSize(value any) (int, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0, err
	}
	return len(encoded), nil
}

func jsonCompatibleClone(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var cloned any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}
