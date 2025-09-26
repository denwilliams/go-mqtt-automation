package strategy

import (
	"strings"
	"testing"
	"time"
)

func TestNewJavaScriptExecutor(t *testing.T) {
	executor := NewJavaScriptExecutor()

	if executor == nil {
		t.Fatal("NewJavaScriptExecutor() returned nil")
	}

	if executor.maxExecutionTime <= 0 {
		t.Error("maxExecutionTime not set")
	}
}

func TestJavaScriptExecutor_Validate(t *testing.T) {
	executor := NewJavaScriptExecutor()

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "valid function",
			code:    "function process(context) { return 'test'; }",
			wantErr: false,
		},
		{
			name:    "valid with complex logic",
			code:    "function process(context) { var x = context.inputs['test']; return x * 2; }",
			wantErr: false,
		},
		{
			name:    "syntax error",
			code:    "function process(context) { return 'unclosed string; }",
			wantErr: true,
		},
		{
			name:    "missing process function",
			code:    "function otherFunction() { return 'test'; }",
			wantErr: true,
		},
		{
			name:    "process is not a function",
			code:    "var process = 'not a function';",
			wantErr: true,
		},
		{
			name:    "empty code",
			code:    "",
			wantErr: true,
		},
		{
			name:    "only comments",
			code:    "// This is just a comment",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJavaScriptExecutor_Execute_Basic(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		ID:   "test",
		Name: "Test Strategy",
		Code: `function process(context) {
			return {
				input_count: Object.keys(context.inputs).length,
				trigger: context.triggeringTopic,
				timestamp: context.getTime()
			};
		}`,
		Language: "javascript",
		Parameters: map[string]interface{}{
			"test_param": "test_value",
		},
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{
			"topic1": 25.5,
			"topic2": true,
		},
		TriggeringTopic: "topic1",
		LastOutputs:     nil,
		Parameters:      strategy.Parameters,
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	if result.ExecutionTime <= 0 {
		t.Error("ExecutionTime not set")
	}

	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	if resultMap["input_count"] != int64(2) {
		t.Errorf("input_count = %v, want 2", resultMap["input_count"])
	}

	if resultMap["trigger"] != "topic1" {
		t.Errorf("trigger = %v, want 'topic1'", resultMap["trigger"])
	}

	if _, ok := resultMap["timestamp"].(int64); !ok {
		t.Error("timestamp not returned as int64")
	}
}

func TestJavaScriptExecutor_Execute_WithLogging(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			context.log('This is a test message');
			context.log('Value is:', context.inputs['test']);
			return 'logged';
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{
			"test": 42,
		},
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	if len(result.LogMessages) != 2 {
		t.Errorf("expected 2 log messages, got %d", len(result.LogMessages))
	}

	if result.LogMessages[0] != "This is a test message" {
		t.Errorf("first log message = %q, want 'This is a test message'", result.LogMessages[0])
	}

	if result.LogMessages[1] != "Value is: 42" {
		t.Errorf("second log message = %q, want 'Value is: 42'", result.LogMessages[1])
	}
}

func TestJavaScriptExecutor_Execute_WithEmit(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			context.emit('output/topic1', 'hello');
			context.emit('output/topic2', { value: 42, active: true });
			return 'emitted';
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{},
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	if len(result.EmittedEvents) != 2 {
		t.Errorf("expected 2 emitted events, got %d", len(result.EmittedEvents))
	}

	event1 := result.EmittedEvents[0]
	if event1.Topic != "output/topic1" || event1.Value != "hello" {
		t.Errorf("first event = {%q, %v}, want {'output/topic1', 'hello'}", event1.Topic, event1.Value)
	}

	event2 := result.EmittedEvents[1]
	if event2.Topic != "output/topic2" {
		t.Errorf("second event topic = %q, want 'output/topic2'", event2.Topic)
	}
}

func TestJavaScriptExecutor_Execute_WithUtilityFunctions(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			return {
				current_time: context.getTime(),
				iso_time: context.getISO(),
				parsed_json: context.parseJSON('{"test": 123}'),
				stringified: context.stringify({foo: 'bar'})
			};
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{},
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Check getTime()
	if _, ok := resultMap["current_time"].(int64); !ok {
		t.Error("getTime() did not return int64")
	}

	// Check getISO()
	if isoTime, ok := resultMap["iso_time"].(string); !ok {
		t.Error("getISO() did not return string")
	} else if len(isoTime) == 0 {
		t.Error("getISO() returned empty string")
	}

	// Check parseJSON()
	parsedJSON := resultMap["parsed_json"]
	if parsedMap, ok := parsedJSON.(map[string]interface{}); !ok {
		t.Error("parseJSON() did not return object")
	} else {
		testValue := parsedMap["test"]
		// JavaScript numbers can be returned as either int64 or float64
		if testInt, ok := testValue.(int64); ok {
			if testInt != 123 {
				t.Errorf("parseJSON() result = %v, want 123", testInt)
			}
		} else if testFloat, ok := testValue.(float64); ok {
			if testFloat != 123.0 {
				t.Errorf("parseJSON() result = %v, want 123.0", testFloat)
			}
		} else {
			t.Errorf("parseJSON() test value = %v (type %T), want 123", testValue, testValue)
		}
	}

	// Check stringify()
	if stringified, ok := resultMap["stringified"].(string); !ok {
		t.Error("stringify() did not return string")
	} else if !strings.Contains(stringified, "foo") || !strings.Contains(stringified, "bar") {
		t.Errorf("stringify() result = %q, want JSON with foo and bar", stringified)
	}
}

func TestJavaScriptExecutor_Execute_WithMath(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			return {
				abs_value: Math.abs(-5),
				max_value: Math.max(1, 5, 3),
				min_value: Math.min(1, 5, 3),
				rounded: Math.round(3.7)
			};
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{},
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	if resultMap["abs_value"] != int64(5) {
		t.Errorf("Math.abs(-5) = %v, want 5", resultMap["abs_value"])
	}

	if resultMap["max_value"] != int64(5) {
		t.Errorf("Math.max(1,5,3) = %v, want 5", resultMap["max_value"])
	}

	if resultMap["min_value"] != int64(1) {
		t.Errorf("Math.min(1,5,3) = %v, want 1", resultMap["min_value"])
	}

	if resultMap["rounded"] != int64(4) {
		t.Errorf("Math.round(3.7) = %v, want 4", resultMap["rounded"])
	}
}

func TestJavaScriptExecutor_Execute_WithTrigonometry(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			var angle = Math.PI / 2; // 90 degrees in radians
			return {
				sine: Math.sin(angle),
				cosine: Math.cos(angle),
				tangent: Math.tan(Math.PI / 4), // 45 degrees = 1
				pi: Math.PI,
				sqrt: Math.sqrt(16),
				pow: Math.pow(2, 3),
				exp: Math.exp(0), // e^0 = 1
				log: Math.log(Math.E) // natural log of e = 1
			};
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{},
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Check Math.sin(PI/2) ≈ 1.0
	sineRaw := resultMap["sine"]
	if sine, ok := sineRaw.(float64); ok {
		if sine < 0.99 || sine > 1.01 {
			t.Errorf("Math.sin(PI/2) = %v, want ~1.0", sine)
		}
	} else if sineInt, ok := sineRaw.(int64); ok {
		if sineInt != 1 {
			t.Errorf("Math.sin(PI/2) = %v, want ~1", sineInt)
		}
	} else {
		t.Errorf("Math.sin() returned unexpected type %T: %v", sineRaw, sineRaw)
	}

	// Check Math.cos(PI/2) ≈ 0.0
	cosineRaw := resultMap["cosine"]
	if cosine, ok := cosineRaw.(float64); ok {
		if cosine < -0.01 || cosine > 0.01 {
			t.Errorf("Math.cos(PI/2) = %v, want ~0.0", cosine)
		}
	} else if cosineInt, ok := cosineRaw.(int64); ok {
		if cosineInt != 0 {
			t.Errorf("Math.cos(PI/2) = %v, want ~0", cosineInt)
		}
	} else {
		t.Errorf("Math.cos() returned unexpected type %T: %v", cosineRaw, cosineRaw)
	}

	// Check Math.tan(PI/4) ≈ 1.0
	tangentRaw := resultMap["tangent"]
	if tangent, ok := tangentRaw.(float64); ok {
		if tangent < 0.99 || tangent > 1.01 {
			t.Errorf("Math.tan(PI/4) = %v, want ~1.0", tangent)
		}
	} else if tangentInt, ok := tangentRaw.(int64); ok {
		if tangentInt != 1 {
			t.Errorf("Math.tan(PI/4) = %v, want ~1", tangentInt)
		}
	} else {
		t.Errorf("Math.tan() returned unexpected type %T: %v", tangentRaw, tangentRaw)
	}

	// Check Math.PI ≈ 3.14159...
	piRaw := resultMap["pi"]
	if pi, ok := piRaw.(float64); ok {
		if pi < 3.14 || pi > 3.15 {
			t.Errorf("Math.PI = %v, want ~3.14159", pi)
		}
	} else {
		t.Errorf("Math.PI returned unexpected type %T: %v", piRaw, piRaw)
	}

	// Check Math.sqrt(16) = 4.0
	sqrtRaw := resultMap["sqrt"]
	if sqrt, ok := sqrtRaw.(float64); ok {
		if sqrt != 4.0 {
			t.Errorf("Math.sqrt(16) = %v, want 4.0", sqrt)
		}
	} else if sqrtInt, ok := sqrtRaw.(int64); ok {
		if sqrtInt != 4 {
			t.Errorf("Math.sqrt(16) = %v, want 4", sqrtInt)
		}
	} else {
		t.Errorf("Math.sqrt() returned unexpected type %T: %v", sqrtRaw, sqrtRaw)
	}

	// Check Math.pow(2, 3) = 8.0
	powRaw := resultMap["pow"]
	if pow, ok := powRaw.(float64); ok {
		if pow != 8.0 {
			t.Errorf("Math.pow(2, 3) = %v, want 8.0", pow)
		}
	} else if powInt, ok := powRaw.(int64); ok {
		if powInt != 8 {
			t.Errorf("Math.pow(2, 3) = %v, want 8", powInt)
		}
	} else {
		t.Errorf("Math.pow() returned unexpected type %T: %v", powRaw, powRaw)
	}

	// Check Math.exp(0) = 1.0
	expRaw := resultMap["exp"]
	if exp, ok := expRaw.(float64); ok {
		if exp < 0.99 || exp > 1.01 {
			t.Errorf("Math.exp(0) = %v, want ~1.0", exp)
		}
	} else if expInt, ok := expRaw.(int64); ok {
		if expInt != 1 {
			t.Errorf("Math.exp(0) = %v, want ~1", expInt)
		}
	} else {
		t.Errorf("Math.exp() returned unexpected type %T: %v", expRaw, expRaw)
	}

	// Check Math.log(e) = 1.0
	logRaw := resultMap["log"]
	if log, ok := logRaw.(float64); ok {
		if log < 0.99 || log > 1.01 {
			t.Errorf("Math.log(e) = %v, want ~1.0", log)
		}
	} else if logInt, ok := logRaw.(int64); ok {
		if logInt != 1 {
			t.Errorf("Math.log(e) = %v, want ~1", logInt)
		}
	} else {
		t.Errorf("Math.log() returned unexpected type %T: %v", logRaw, logRaw)
	}
}

func TestJavaScriptExecutor_Execute_WithError(t *testing.T) {
	executor := NewJavaScriptExecutor()

	tests := []struct {
		name     string
		code     string
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "runtime error",
			code:     "function process(context) { throw new Error('test error'); }",
			wantErr:  true,
			errorMsg: "test error",
		},
		{
			name:     "reference error",
			code:     "function process(context) { return nonExistentVariable; }",
			wantErr:  true,
			errorMsg: "nonExistentVariable",
		},
		{
			name:     "syntax error in process function",
			code:     "function process(context) { return 'unclosed; }",
			wantErr:  true,
			errorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &Strategy{Code: tt.code}
			context := ExecutionContext{InputValues: map[string]interface{}{}}

			result := executor.Execute(strategy, context)

			if tt.wantErr && result.Error == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
			}

			if tt.wantErr && tt.errorMsg != "" && result.Error != nil {
				if !strings.Contains(result.Error.Error(), tt.errorMsg) {
					t.Errorf("error message %q does not contain %q", result.Error.Error(), tt.errorMsg)
				}
			}
		})
	}
}

func TestJavaScriptExecutor_Execute_Timeout(t *testing.T) {
	executor := NewJavaScriptExecutor()
	executor.maxExecutionTime = 10 * time.Millisecond // Very short timeout for testing

	strategy := &Strategy{
		Code: `function process(context) {
			// Infinite loop to test timeout
			while (true) {
				// This should timeout
			}
			return 'never reached';
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{},
	}

	result := executor.Execute(strategy, context)

	if result.Error == nil {
		t.Error("expected timeout error but got none")
	}

	if !strings.Contains(result.Error.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", result.Error)
	}

	if result.ExecutionTime < executor.maxExecutionTime {
		t.Errorf("execution time %v should be at least %v", result.ExecutionTime, executor.maxExecutionTime)
	}
}

func TestJavaScriptExecutor_Execute_ComplexStrategy(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			var temps = [];
			var total = 0;
			
			// Collect temperature values
			for (var topic in context.inputs) {
				if (topic.indexOf('temperature') !== -1) {
					var value = parseFloat(context.inputs[topic]);
					if (!isNaN(value)) {
						temps.push(value);
						total += value;
					}
				}
			}
			
			if (temps.length === 0) {
				context.log('No temperature sensors found');
				return null;
			}
			
			var average = total / temps.length;
			var threshold = context.parameters.threshold || 25;
			
			context.log('Average temperature: ' + average.toFixed(1) + '°C');
			
			if (average > threshold) {
				context.emit('alerts/high-temp', {
					average: average,
					threshold: threshold,
					sensor_count: temps.length
				});
				context.log('High temperature alert triggered!');
			}
			
			return {
				average_temp: Math.round(average * 10) / 10,
				sensor_count: temps.length,
				all_temps: temps,
				is_high: average > threshold,
				timestamp: context.getISO()
			};
		}`,
		Parameters: map[string]interface{}{
			"threshold": 22.0,
		},
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{
			"sensors/temperature/living-room": 24.5,
			"sensors/temperature/kitchen":     26.0,
			"sensors/temperature/bedroom":     21.5,
			"sensors/humidity/bathroom":       65.0, // Should be ignored
		},
		Parameters: strategy.Parameters,
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	// Check result
	resultMap, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// expectedAverage := (24.5 + 26.0 + 21.5) / 3 // 24.0
	avgTempRaw := resultMap["average_temp"]
	// JavaScript might return integers for whole numbers
	if avgTemp, ok := avgTempRaw.(float64); ok {
		if avgTemp != 24.0 {
			t.Errorf("average_temp = %v, want 24.0", avgTemp)
		}
	} else if avgTempInt, ok := avgTempRaw.(int64); ok {
		if avgTempInt != 24 {
			t.Errorf("average_temp = %v, want 24", avgTempInt)
		}
	} else {
		t.Errorf("average_temp not returned as float64 or int64, got %T: %v", avgTempRaw, avgTempRaw)
	}

	if resultMap["sensor_count"] != int64(3) {
		t.Errorf("sensor_count = %v, want 3", resultMap["sensor_count"])
	}

	if resultMap["is_high"] != true {
		t.Errorf("is_high = %v, want true (24.0 > 22.0)", resultMap["is_high"])
	}

	// Check log messages
	if len(result.LogMessages) < 2 {
		t.Errorf("expected at least 2 log messages, got %d", len(result.LogMessages))
	}

	// Check emitted events (high temp alert should be triggered)
	if len(result.EmittedEvents) != 1 {
		t.Errorf("expected 1 emitted event, got %d", len(result.EmittedEvents))
	} else {
		event := result.EmittedEvents[0]
		if event.Topic != "alerts/high-temp" {
			t.Errorf("emitted topic = %q, want 'alerts/high-temp'", event.Topic)
		}
	}
}

func TestJavaScriptExecutor_Execute_WithBooleanValues(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			// Test various boolean and falsy values
			context.emit('/true-value', true);
			context.emit('/false-value', false);
			context.emit('/null-value', null);
			context.emit('/undefined-value', undefined);
			context.emit('/zero-value', 0);
			context.emit('/empty-string', '');

			// Return false from main process
			return false;
		}`,
	}

	context := ExecutionContext{
		InputValues: map[string]interface{}{},
	}

	result := executor.Execute(strategy, context)

	if result.Error != nil {
		t.Fatalf("Execute() failed: %v", result.Error)
	}

	// Check main result is false (not null)
	if result.Result != false {
		t.Errorf("Main result = %v (type %T), want false", result.Result, result.Result)
	}

	// Check emitted events
	if len(result.EmittedEvents) != 6 {
		t.Fatalf("Expected 6 emitted events, got %d", len(result.EmittedEvents))
	}

	expectedEvents := map[string]interface{}{
		"/true-value":      true,
		"/false-value":     false,  // This should NOT be null
		"/null-value":      nil,
		"/undefined-value": nil,
		"/zero-value":      int64(0),
		"/empty-string":    "",
	}

	for _, event := range result.EmittedEvents {
		expectedValue, exists := expectedEvents[event.Topic]
		if !exists {
			t.Errorf("Unexpected event topic: %s", event.Topic)
			continue
		}

		if event.Topic == "/false-value" && event.Value != false {
			t.Errorf("Event %s = %v (type %T), want false (boolean)", event.Topic, event.Value, event.Value)
		} else if event.Topic == "/true-value" && event.Value != true {
			t.Errorf("Event %s = %v (type %T), want true (boolean)", event.Topic, event.Value, event.Value)
		} else if event.Value != expectedValue {
			t.Errorf("Event %s = %v (type %T), want %v (type %T)", event.Topic, event.Value, event.Value, expectedValue, expectedValue)
		}
	}
}

func TestJavaScriptExecutor_Execute_WithLastOutputs(t *testing.T) {
	executor := NewJavaScriptExecutor()

	strategy := &Strategy{
		Code: `function process(context) {
			var currentValue = context.inputs['sensor'];
			var lastValue = context.lastOutputs ? context.lastOutputs.value : null;
			var hasChanged = currentValue !== lastValue;
			
			if (hasChanged) {
				context.log('Value changed from ' + lastValue + ' to ' + currentValue);
			}
			
			return {
				value: currentValue,
				previous: lastValue,
				changed: hasChanged,
				change_count: (context.lastOutputs ? context.lastOutputs.change_count : 0) + (hasChanged ? 1 : 0)
			};
		}`,
	}

	// First execution (no previous outputs)
	context1 := ExecutionContext{
		InputValues: map[string]interface{}{
			"sensor": 10,
		},
		LastOutputs: nil,
	}

	result1 := executor.Execute(strategy, context1)
	if result1.Error != nil {
		t.Fatalf("First execute failed: %v", result1.Error)
	}

	result1Map := result1.Result.(map[string]interface{})
	if result1Map["change_count"] != int64(1) {
		t.Errorf("first execution change_count = %v, want 1", result1Map["change_count"])
	}

	// Second execution (with previous outputs, same value)
	context2 := ExecutionContext{
		InputValues: map[string]interface{}{
			"sensor": 10, // Same value
		},
		LastOutputs: result1Map,
	}

	result2 := executor.Execute(strategy, context2)
	if result2.Error != nil {
		t.Fatalf("Second execute failed: %v", result2.Error)
	}

	result2Map := result2.Result.(map[string]interface{})
	if result2Map["changed"] != false {
		t.Error("second execution should not show change for same value")
	}
	if result2Map["change_count"] != int64(1) {
		t.Errorf("second execution change_count = %v, want 1", result2Map["change_count"])
	}

	// Third execution (with previous outputs, different value)
	context3 := ExecutionContext{
		InputValues: map[string]interface{}{
			"sensor": 20, // Different value
		},
		LastOutputs: result2Map,
	}

	result3 := executor.Execute(strategy, context3)
	if result3.Error != nil {
		t.Fatalf("Third execute failed: %v", result3.Error)
	}

	result3Map := result3.Result.(map[string]interface{})
	if result3Map["changed"] != true {
		t.Error("third execution should show change for different value")
	}
	if result3Map["change_count"] != int64(2) {
		t.Errorf("third execution change_count = %v, want 2", result3Map["change_count"])
	}
}
