package topics

import (
	"fmt"
	"log"
	"strings"
	"testing"
)

// Test data flow through a chain of dependent topics
func TestChainedTopicDataFlow(t *testing.T) {
	manager := NewManager(log.New(log.Writer(), "TEST: ", log.LstdFlags))

	// Track strategy executions and their outputs
	executionLog := make([]string, 0)
	resultValues := make(map[string]interface{})

	// Mock strategy executor that logs executions and returns predictable results
	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			executionLog = append(executionLog, fmt.Sprintf("%s:%s", strategyID, triggerTopic))

			switch strategyID {
			case "multiply-by-2":
				// Simple doubling strategy
				for _, value := range inputs {
					if num, ok := value.(float64); ok {
						result := num * 2
						resultValues[strategyID] = result
						return result, nil
					}
				}
				return 0, fmt.Errorf("no numeric input found")

			case "add-10":
				// Add 10 to input
				for _, value := range inputs {
					if num, ok := value.(float64); ok {
						result := num + 10
						resultValues[strategyID] = result
						return result, nil
					}
				}
				return 0, fmt.Errorf("no numeric input found")

			case "format-string":
				// Format as string
				for _, value := range inputs {
					if num, ok := value.(float64); ok {
						result := fmt.Sprintf("Final: %.1f", num)
						resultValues[strategyID] = result
						return result, nil
					}
				}
				return "", fmt.Errorf("no numeric input found")

			default:
				return "unknown", nil
			}
		},
	}

	manager.SetStrategyExecutor(mockExec)

	// Set up chain: sensor -> double -> add10 -> format
	// 1. External topic (sensor input)
	sensorTopic := manager.AddExternalTopic("sensor/temperature")

	// 2. First internal topic (doubles the sensor value)
	doubledTopic, err := manager.AddInternalTopic("processed/doubled", []string{"sensor/temperature"}, nil, "multiply-by-2", false, false)
	if err != nil {
		t.Fatalf("Failed to add doubled topic: %v", err)
	}

	// 3. Second internal topic (adds 10 to doubled value)
	adjustedTopic, err := manager.AddInternalTopic("processed/adjusted", []string{"processed/doubled"}, nil, "add-10", false, false)
	if err != nil {
		t.Fatalf("Failed to add adjusted topic: %v", err)
	}

	// 4. Final internal topic (formats as string)
	finalTopic, err := manager.AddInternalTopic("output/formatted", []string{"processed/adjusted"}, nil, "format-string", false, false)
	if err != nil {
		t.Fatalf("Failed to add final topic: %v", err)
	}

	// Inject initial value into sensor topic
	initialValue := 5.0
	err = sensorTopic.Emit(initialValue)
	if err != nil {
		t.Fatalf("Failed to emit sensor value: %v", err)
	}

	// Verify the chain was triggered in the correct order
	expectedExecutions := []string{
		"multiply-by-2:sensor/temperature",
		"add-10:processed/doubled",
		"format-string:processed/adjusted",
	}

	if len(executionLog) != len(expectedExecutions) {
		t.Errorf("Expected %d executions, got %d: %v", len(expectedExecutions), len(executionLog), executionLog)
	}

	for i, expected := range expectedExecutions {
		if i >= len(executionLog) || executionLog[i] != expected {
			t.Errorf("Execution %d: expected %q, got %q", i, expected, safeGet(executionLog, i))
		}
	}

	// Verify intermediate values
	if resultValues["multiply-by-2"] != 10.0 {
		t.Errorf("Doubled value: expected 10.0, got %v", resultValues["multiply-by-2"])
	}

	if resultValues["add-10"] != 20.0 {
		t.Errorf("Adjusted value: expected 20.0, got %v", resultValues["add-10"])
	}

	if resultValues["format-string"] != "Final: 20.0" {
		t.Errorf("Final value: expected 'Final: 20.0', got %v", resultValues["format-string"])
	}

	// Verify final topic values
	if doubledTopic.LastValue() != 10.0 {
		t.Errorf("Doubled topic value: expected 10.0, got %v", doubledTopic.LastValue())
	}

	if adjustedTopic.LastValue() != 20.0 {
		t.Errorf("Adjusted topic value: expected 20.0, got %v", adjustedTopic.LastValue())
	}

	if finalTopic.LastValue() != "Final: 20.0" {
		t.Errorf("Final topic value: expected 'Final: 20.0', got %v", finalTopic.LastValue())
	}
}

// Test branching topic chains (one input feeding multiple chains)
func TestBranchingTopicChains(t *testing.T) {
	manager := NewManager(log.New(log.Writer(), "TEST-BRANCH: ", log.LstdFlags))

	executionLog := make([]string, 0)
	resultValues := make(map[string]interface{})

	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			executionLog = append(executionLog, fmt.Sprintf("%s:%s", strategyID, triggerTopic))

			switch strategyID {
			case "celsius-to-fahrenheit":
				for _, value := range inputs {
					if temp, ok := value.(float64); ok {
						result := (temp * 9 / 5) + 32
						resultValues[strategyID] = result
						return result, nil
					}
				}
				return 0, fmt.Errorf("no temperature input")

			case "temperature-status":
				for _, value := range inputs {
					if temp, ok := value.(float64); ok {
						var status string
						if temp < 0 {
							status = "freezing"
						} else if temp < 20 {
							status = "cold"
						} else if temp < 30 {
							status = "comfortable"
						} else {
							status = "hot"
						}
						resultValues[strategyID] = status
						return status, nil
					}
				}
				return "unknown", fmt.Errorf("no temperature input")

			case "alert-check":
				for _, value := range inputs {
					if temp, ok := value.(float64); ok {
						alert := temp > 25 || temp < 5
						resultValues[strategyID] = alert
						return alert, nil
					}
				}
				return false, fmt.Errorf("no temperature input")

			default:
				return "unknown", nil
			}
		},
	}

	manager.SetStrategyExecutor(mockExec)

	// Set up branching chains from single sensor
	sensorTopic := manager.AddExternalTopic("sensor/room-temp")

	// Branch 1: Temperature conversion
	fahrenheitTopic, err := manager.AddInternalTopic("converted/fahrenheit", []string{"sensor/room-temp"}, nil, "celsius-to-fahrenheit", false, false)
	if err != nil {
		t.Fatalf("Failed to add fahrenheit topic: %v", err)
	}

	// Branch 2: Status determination
	statusTopic, err := manager.AddInternalTopic("status/temperature", []string{"sensor/room-temp"}, nil, "temperature-status", false, false)
	if err != nil {
		t.Fatalf("Failed to add status topic: %v", err)
	}

	// Branch 3: Alert checking
	alertTopic, err := manager.AddInternalTopic("alerts/temperature", []string{"sensor/room-temp"}, nil, "alert-check", false, false)
	if err != nil {
		t.Fatalf("Failed to add alert topic: %v", err)
	}

	// Inject temperature value
	tempValue := 22.0 // 22°C should be comfortable and not trigger alert
	err = sensorTopic.Emit(tempValue)
	if err != nil {
		t.Fatalf("Failed to emit temperature: %v", err)
	}

	// All three strategies should execute
	if len(executionLog) != 3 {
		t.Errorf("Expected 3 executions, got %d: %v", len(executionLog), executionLog)
	}

	// Verify each branch got triggered
	expectedStrategies := map[string]bool{
		"celsius-to-fahrenheit": false,
		"temperature-status":    false,
		"alert-check":           false,
	}

	for _, execution := range executionLog {
		parts := strings.Split(execution, ":")
		if len(parts) >= 1 {
			expectedStrategies[parts[0]] = true
		}
	}

	for strategy, executed := range expectedStrategies {
		if !executed {
			t.Errorf("Strategy %s was not executed", strategy)
		}
	}

	// Verify results
	expectedFahrenheit := (22.0 * 9 / 5) + 32 // 71.6°F
	if resultValues["celsius-to-fahrenheit"] != expectedFahrenheit {
		t.Errorf("Fahrenheit conversion: expected %v, got %v", expectedFahrenheit, resultValues["celsius-to-fahrenheit"])
	}

	if resultValues["temperature-status"] != "comfortable" {
		t.Errorf("Temperature status: expected 'comfortable', got %v", resultValues["temperature-status"])
	}

	if resultValues["alert-check"] != false {
		t.Errorf("Alert check: expected false, got %v", resultValues["alert-check"])
	}

	// Verify topic values
	if fahrenheitTopic.LastValue() != expectedFahrenheit {
		t.Errorf("Fahrenheit topic value: expected %v, got %v", expectedFahrenheit, fahrenheitTopic.LastValue())
	}

	if statusTopic.LastValue() != "comfortable" {
		t.Errorf("Status topic value: expected 'comfortable', got %v", statusTopic.LastValue())
	}

	if alertTopic.LastValue() != false {
		t.Errorf("Alert topic value: expected false, got %v", alertTopic.LastValue())
	}
}

// Test convergent topic chains (multiple inputs feeding single output)
func TestConvergentTopicChains(t *testing.T) {
	manager := NewManager(log.New(log.Writer(), "TEST-CONVERGE: ", log.LstdFlags))

	executionLog := make([]string, 0)
	resultValues := make(map[string]interface{})

	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			executionLog = append(executionLog, fmt.Sprintf("%s:%s", strategyID, triggerTopic))

			switch strategyID {
			case "average-temperature":
				total := 0.0
				count := 0
				for _, value := range inputs {
					if temp, ok := value.(float64); ok {
						total += temp
						count++
					}
				}
				if count == 0 {
					return 0, fmt.Errorf("no temperature inputs")
				}
				result := total / float64(count)
				resultValues[strategyID] = result
				return result, nil

			case "hvac-control":
				// Get average temperature and determine HVAC action
				for _, value := range inputs {
					if avgTemp, ok := value.(float64); ok {
						var action string
						if avgTemp < 18 {
							action = "heat"
						} else if avgTemp > 24 {
							action = "cool"
						} else {
							action = "off"
						}
						resultValues[strategyID] = action
						return action, nil
					}
				}
				return "off", fmt.Errorf("no average temperature input")

			default:
				return "unknown", nil
			}
		},
	}

	manager.SetStrategyExecutor(mockExec)

	// Create multiple sensor inputs
	sensor1 := manager.AddExternalTopic("sensors/living-room/temp")
	sensor2 := manager.AddExternalTopic("sensors/kitchen/temp")
	sensor3 := manager.AddExternalTopic("sensors/bedroom/temp")

	// Convergent topic that averages all temperatures
	avgTopic, err := manager.AddInternalTopic("calculated/average-temp",
		[]string{"sensors/living-room/temp", "sensors/kitchen/temp", "sensors/bedroom/temp"},
		nil, "average-temperature", false, false)
	if err != nil {
		t.Fatalf("Failed to add average topic: %v", err)
	}

	// Final topic that controls HVAC based on average
	hvacTopic, err := manager.AddInternalTopic("control/hvac", []string{"calculated/average-temp"}, nil, "hvac-control", false, false)
	if err != nil {
		t.Fatalf("Failed to add HVAC topic: %v", err)
	}

	// Emit values to all sensors
	err = sensor1.Emit(20.0)
	if err != nil {
		t.Fatalf("Failed to emit sensor1 value: %v", err)
	}

	// After first sensor, average should be calculated and HVAC triggered
	if len(executionLog) < 1 || !strings.Contains(executionLog[0], "average-temperature") {
		t.Errorf("Expected average calculation after first sensor, got: %v", executionLog)
	}

	err = sensor2.Emit(22.0)
	if err != nil {
		t.Fatalf("Failed to emit sensor2 value: %v", err)
	}

	// After second sensor, average should be recalculated
	avgExecutions := 0
	for _, execution := range executionLog {
		if strings.Contains(execution, "average-temperature") {
			avgExecutions++
		}
	}
	if avgExecutions < 2 {
		t.Errorf("Expected at least 2 average calculations, got %d: %v", avgExecutions, executionLog)
	}

	err = sensor3.Emit(19.0)
	if err != nil {
		t.Fatalf("Failed to emit sensor3 value: %v", err)
	}

	// After third sensor, we should have at least 3 average calculations
	avgExecutions = 0
	for _, execution := range executionLog {
		if strings.Contains(execution, "average-temperature") {
			avgExecutions++
		}
	}
	if avgExecutions < 3 {
		t.Errorf("Expected at least 3 average calculations, got %d: %v", avgExecutions, executionLog)
	}

	// The last average calculation should trigger HVAC control
	hvacExecuted := false
	for _, execution := range executionLog {
		if strings.Contains(execution, "hvac-control") {
			hvacExecuted = true
		}
	}

	if !hvacExecuted {
		t.Error("HVAC control was not executed")
	}

	// Verify final average: (20 + 22 + 19) / 3 = 20.33
	expectedAvg := 61.0 / 3
	actualAvg := avgTopic.LastValue()
	if actualAvg != expectedAvg {
		t.Errorf("Average temperature: expected %v, got %v", expectedAvg, actualAvg)
	}

	// Since average is ~20.33, HVAC should be "off" (between 18 and 24)
	expectedHvac := "off"
	actualHvac := hvacTopic.LastValue()
	if actualHvac != expectedHvac {
		t.Errorf("HVAC control: expected %s, got %v", expectedHvac, actualHvac)
	}
}

// Test complex multi-level chain with error handling
func TestComplexChainWithErrors(t *testing.T) {
	manager := NewManager(log.New(log.Writer(), "TEST-ERROR: ", log.LstdFlags))

	executionLog := make([]string, 0)
	errorLog := make([]string, 0)

	mockExec := &mockStrategyExecutor{
		executeFunc: func(strategyID string, inputs map[string]interface{}, triggerTopic string, lastOutput interface{}) (interface{}, error) {
			executionLog = append(executionLog, fmt.Sprintf("%s:%s", strategyID, triggerTopic))

			switch strategyID {
			case "validate-sensor":
				for _, value := range inputs {
					if temp, ok := value.(float64); ok {
						// Reject obviously invalid temperatures
						if temp < -50 || temp > 100 {
							err := fmt.Errorf("invalid temperature: %v", temp)
							errorLog = append(errorLog, err.Error())
							return nil, err
						}
						return temp, nil
					}
				}
				err := fmt.Errorf("no numeric input")
				errorLog = append(errorLog, err.Error())
				return nil, err

			case "safety-check":
				for _, value := range inputs {
					if temp, ok := value.(float64); ok {
						safe := temp >= 0 && temp <= 40
						return safe, nil
					}
				}
				return false, fmt.Errorf("no temperature input")

			default:
				return "unknown", nil
			}
		},
	}

	manager.SetStrategyExecutor(mockExec)

	// Set up validation chain
	sensorTopic := manager.AddExternalTopic("sensor/raw-temp")

	validatedTopic, err := manager.AddInternalTopic("validated/temp", []string{"sensor/raw-temp"}, nil, "validate-sensor", false, false)
	if err != nil {
		t.Fatalf("Failed to add validated topic: %v", err)
	}

	safetyTopic, err := manager.AddInternalTopic("safety/temp-check", []string{"validated/temp"}, nil, "safety-check", false, false)
	if err != nil {
		t.Fatalf("Failed to add safety topic: %v", err)
	}

	// Test 1: Valid temperature should flow through chain
	err = sensorTopic.Emit(25.0)
	if err != nil {
		t.Fatalf("Failed to emit valid temperature: %v", err)
	}

	if len(executionLog) != 2 {
		t.Errorf("Expected 2 executions for valid temp, got %d: %v", len(executionLog), executionLog)
	}

	if validatedTopic.LastValue() != 25.0 {
		t.Errorf("Validated temp: expected 25.0, got %v", validatedTopic.LastValue())
	}

	if safetyTopic.LastValue() != true {
		t.Errorf("Safety check: expected true, got %v", safetyTopic.LastValue())
	}

	// Test 2: Invalid temperature should cause error and stop chain
	executionLog = nil // Reset log
	errorLog = nil

	err = sensorTopic.Emit(150.0) // Invalid temperature
	if err != nil {
		t.Fatalf("Failed to emit invalid temperature: %v", err)
	}

	// Validation should execute but fail, safety should not execute
	if len(executionLog) != 1 {
		t.Errorf("Expected 1 execution for invalid temp, got %d: %v", len(executionLog), executionLog)
	}

	if len(errorLog) == 0 {
		t.Error("Expected error to be logged for invalid temperature")
	}

	// Validated topic should still have previous valid value since validation failed
	if validatedTopic.LastValue() != 25.0 {
		t.Errorf("Validated temp after error: expected 25.0, got %v", validatedTopic.LastValue())
	}
}

// Helper function to safely get element from slice
func safeGet(slice []string, index int) string {
	if index >= len(slice) {
		return "<missing>"
	}
	return slice[index]
}
