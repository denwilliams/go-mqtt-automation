package strategy

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type JavaScriptExecutor struct {
	maxExecutionTime time.Duration
}

func NewJavaScriptExecutor() *JavaScriptExecutor {
	return &JavaScriptExecutor{
		maxExecutionTime: 30 * time.Second,
	}
}

func (jse *JavaScriptExecutor) Execute(strategy *Strategy, context ExecutionContext) ExecutionResult {
	start := time.Now()

	result := ExecutionResult{
		LogMessages:   []string{},
		EmittedEvents: []EmitEvent{},
		ExecutionTime: 0,
	}

	// Create new VM
	vm := goja.New()

	// Set up execution timeout
	done := make(chan bool)
	timeout := time.After(jse.maxExecutionTime)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				result.Error = fmt.Errorf("JavaScript execution panic: %v", r)
			}
			done <- true
		}()

		// Set up the JavaScript environment
		jse.setupEnvironment(vm, &context, &result)

		// Execute the strategy code
		_, err := vm.RunString(strategy.Code)
		if err != nil {
			result.Error = fmt.Errorf("JavaScript execution error: %w", err)
			return
		}

		// Call the process function if it exists
		if processFunc := vm.Get("process"); processFunc != nil {
			if fn, ok := goja.AssertFunction(processFunc); ok {
				contextObj := jse.createContextObject(vm, context)

				// Call the process function directly with the context object
				processResult, err := fn(goja.Undefined(), contextObj)
				if err != nil {
					result.Error = fmt.Errorf("process function execution error: %w", err)
					return
				}

				// Export the result
				result.Result = processResult.Export()
			} else {
				result.Error = fmt.Errorf("process function not found or not a function")
			}
		} else {
			result.Error = fmt.Errorf("process function not found in strategy code")
		}
	}()

	select {
	case <-done:
		result.ExecutionTime = time.Since(start)
	case <-timeout:
		result.Error = fmt.Errorf("execution timeout after %v", jse.maxExecutionTime)
		result.ExecutionTime = jse.maxExecutionTime
	}

	return result
}

func (jse *JavaScriptExecutor) Validate(code string) error {
	vm := goja.New()

	// Try to compile the code
	_, err := vm.RunString(code)
	if err != nil {
		return fmt.Errorf("JavaScript validation error: %w", err)
	}

	// Check if process function exists
	if processFunc := vm.Get("process"); processFunc == nil {
		return fmt.Errorf("process function not found")
	} else {
		if _, ok := goja.AssertFunction(processFunc); !ok {
			return fmt.Errorf("process is not a function")
		}
	}

	return nil
}

func (jse *JavaScriptExecutor) setupEnvironment(vm *goja.Runtime, context *ExecutionContext, result *ExecutionResult) {
	// Set up console.log functionality
	vm.Set("log", func(args ...interface{}) {
		message := make([]string, len(args))
		for i, arg := range args {
			message[i] = fmt.Sprintf("%v", arg)
		}
		result.LogMessages = append(result.LogMessages, strings.Join(message, " "))
	})

	// Set up emit functionality - supports both 1 and 2 argument patterns
	vm.Set("emit", func(args ...interface{}) {
		if len(args) == 1 {
			// Single argument = emit to main topic (empty path)
			result.EmittedEvents = append(result.EmittedEvents, EmitEvent{
				Topic: "", // Empty topic means main topic
				Value: args[0],
			})
		} else if len(args) == 2 {
			// Two arguments = topic path + value
			if topicStr, ok := args[0].(string); ok {
				result.EmittedEvents = append(result.EmittedEvents, EmitEvent{
					Topic: topicStr,
					Value: args[1],
				})
			}
		}
	})

	// Set up utility functions
	vm.Set("getTime", func() int64 {
		return time.Now().Unix()
	})

	vm.Set("getISO", func() string {
		return time.Now().Format(time.RFC3339)
	})

	vm.Set("parseJSON", func(jsonStr string) interface{} {
		var result interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil
		}
		return result
	})

	vm.Set("stringify", func(obj interface{}) string {
		data, err := json.Marshal(obj)
		if err != nil {
			return ""
		}
		return string(data)
	})

	// Note: Modern goja versions include a built-in Math object with all standard functions
	// including sin, cos, tan, sqrt, pow, PI, E, etc. We don't need to override it.

	// Set up context object that will be available to the script
	vm.Set("context", jse.createContextObject(vm, *context))
}

func (jse *JavaScriptExecutor) createContextObject(vm *goja.Runtime, context ExecutionContext) *goja.Object {
	obj := vm.NewObject()

	// Set inputs
	inputsObj := vm.NewObject()
	for key, value := range context.InputValues {
		inputsObj.Set(key, value)
	}
	obj.Set("inputs", inputsObj)

	// Set other context properties
	obj.Set("triggeringTopic", context.TriggeringTopic)
	obj.Set("lastOutputs", context.LastOutputs)
	obj.Set("topicName", context.TopicName)

	// Set parameters
	paramsObj := vm.NewObject()
	for key, value := range context.Parameters {
		paramsObj.Set(key, value)
	}
	obj.Set("parameters", paramsObj)

	// Add utility methods to context
	obj.Set("log", vm.Get("log"))
	obj.Set("emit", vm.Get("emit"))
	obj.Set("getTime", vm.Get("getTime"))
	obj.Set("getISO", vm.Get("getISO"))
	obj.Set("parseJSON", vm.Get("parseJSON"))
	obj.Set("stringify", vm.Get("stringify"))

	return obj
}
