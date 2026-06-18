package script

import (
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/suyue/mocktrue/internal/modules/serial/protocol"
)

// Engine manages a goja JavaScript runtime for protocol parsing scripts.
// Each engine instance is NOT safe for concurrent use; create one per
// concurrent parsing context.
type Engine struct {
	runtime *goja.Runtime
	fields  []protocol.Field
	errs    []string
	data    []byte
	mu      sync.Mutex
}

// NewEngine creates a new script engine with the API functions injected.
func NewEngine() *Engine {
	vm := goja.New()
	e := &Engine{
		runtime: vm,
		fields:  make([]protocol.Field, 0),
		errs:    make([]string, 0),
	}
	e.injectAPI()
	return e
}

// injectAPI registers the global functions available to scripts.
func (e *Engine) injectAPI() {
	vm := e.runtime

	vm.Set("byte", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		offset := int(call.Argument(0).ToInteger())
		return vm.ToValue(e.getByte(offset))
	})

	vm.Set("bytes", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		offset := int(call.Argument(0).ToInteger())
		length := int(call.Argument(1).ToInteger())
		return vm.ToValue(e.getBytes(offset, length))
	})

	vm.Set("u16", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		offset := int(call.Argument(0).ToInteger())
		le := true
		if len(call.Arguments) > 1 {
			le = call.Argument(1).ToBoolean()
		}
		return vm.ToValue(e.readU16(offset, le))
	})

	vm.Set("u32", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		offset := int(call.Argument(0).ToInteger())
		le := true
		if len(call.Arguments) > 1 {
			le = call.Argument(1).ToBoolean()
		}
		return vm.ToValue(e.readU32(offset, le))
	})

	vm.Set("field", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		name := call.Argument(0).String()
		value := call.Argument(1).Export()
		display := ""
		if len(call.Arguments) > 2 {
			display = call.Argument(2).String()
		}
		e.fields = append(e.fields, protocol.Field{
			Name:    name,
			Value:   value,
			Display: display,
		})
		return goja.Undefined()
	})

	vm.Set("error", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		msg := call.Argument(0).String()
		e.errs = append(e.errs, msg)
		return goja.Undefined()
	})

	vm.Set("len", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		return vm.ToValue(len(e.data))
	})

	vm.Set("now", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(time.Now().UnixNano())
	})
}

func (e *Engine) getByte(offset int) int {
	if offset < 0 || offset >= len(e.data) {
		return 0
	}
	return int(e.data[offset])
}

func (e *Engine) getBytes(offset, length int) []int {
	result := make([]int, 0, length)
	for i := 0; i < length; i++ {
		idx := offset + i
		if idx >= 0 && idx < len(e.data) {
			result = append(result, int(e.data[idx]))
		} else {
			result = append(result, 0)
		}
	}
	return result
}

func (e *Engine) readU16(offset int, le bool) int {
	if offset+1 >= len(e.data) {
		return 0
	}
	if le {
		return int(e.data[offset]) | int(e.data[offset+1])<<8
	}
	return int(e.data[offset])<<8 | int(e.data[offset+1])
}

func (e *Engine) readU32(offset int, le bool) int {
	if offset+3 >= len(e.data) {
		return 0
	}
	if le {
		return int(e.data[offset]) | int(e.data[offset+1])<<8 |
			int(e.data[offset+2])<<16 | int(e.data[offset+3])<<24
	}
	return int(e.data[offset])<<24 | int(e.data[offset+1])<<16 |
		int(e.data[offset+2])<<8 | int(e.data[offset+3])
}

// RunScript executes a JavaScript parsing script with the given data.
func (e *Engine) RunScript(script string, data []byte, timeout time.Duration) (protocol.ParseResult, error) {
	e.mu.Lock()
	e.fields = e.fields[:0]
	e.errs = e.errs[:0]
	e.data = data
	e.mu.Unlock()

	done := make(chan struct{})
	var result protocol.ParseResult
	var execErr error

	go func() {
		defer close(done)
		_, err := e.runtime.RunString(script)
		if err != nil {
			execErr = fmt.Errorf("script error: %w", err)
			return
		}
		e.mu.Lock()
		result = protocol.ParseResult{
			OK:       len(e.errs) == 0,
			Fields:   e.fields,
			Errors:   e.errs,
			Consumed: len(data),
		}
		e.mu.Unlock()
	}()

	select {
	case <-done:
		return result, execErr
	case <-time.After(timeout):
		e.runtime.Interrupt("timeout")
		return protocol.ParseResult{}, fmt.Errorf("script execution timed out")
	}
}
