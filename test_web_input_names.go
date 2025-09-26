package main

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"log"
	"os"
	"github.com/denwilliams/go-mqtt-automation/pkg/topics"
	"github.com/denwilliams/go-mqtt-automation/pkg/strategy"
	"github.com/denwilliams/go-mqtt-automation/pkg/web"
	"github.com/denwilliams/go-mqtt-automation/pkg/state"
)

func main() {
	fmt.Println("🧪 Testing web form input names processing...")
	
	// Setup components  
	manager := topics.NewManager(log.New(os.Stdout, "TOPIC: ", log.LstdFlags))
	engine := strategy.NewEngine(log.New(os.Stdout, "STRATEGY: ", log.LstdFlags))
	stateManager := &state.MockStateManager{}
	
	// Add a test strategy
	testStrategy := &strategy.Strategy{
		ID:   "test-strategy",
		Name: "Test Strategy",
		Code: `function process(context) { return "test"; }`,
		Language: "javascript",
	}
	
	err := engine.AddStrategy(testStrategy)
	if err != nil {
		fmt.Printf("❌ Failed to add strategy: %v\n", err)
		return
	}
	
	// Create web server
	server := web.NewServer(manager, engine, stateManager, log.New(os.Stdout, "WEB: ", log.LstdFlags))
	
	// Test 1: Create topic with input names
	fmt.Println("1. Creating topic through web form...")
	formData := url.Values{
		"name": {"test/topic"},
		"inputs": {"sensor/temp1\nsensor/temp2"},
		"input_names": {`{"sensor/temp1": "living_room", "sensor/temp2": "bedroom"}`},
		"strategy_id": {"test-strategy"},
		"emit_to_mqtt": {"on"},
	}
	
	req := httptest.NewRequest("POST", "/topics/new", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	server.ServeHTTP(w, req)
	
	if w.Code != 302 { // Expect redirect on success
		fmt.Printf("❌ FAIL: Topic creation returned %d, expected 302 redirect\n", w.Code)
		fmt.Printf("Response: %s\n", w.Body.String())
		return
	}
	
	// Check if topic was created with input names
	topic := manager.GetInternalTopic("test/topic")
	if topic == nil {
		fmt.Printf("❌ FAIL: Topic was not created\n")
		return
	}
	
	config := topic.GetConfig()
	fmt.Printf("📊 Created topic InputNames: %v\n", config.InputNames)
	
	if len(config.InputNames) != 2 {
		fmt.Printf("❌ FAIL: Expected 2 input names, got %d\n", len(config.InputNames))
		return
	}
	
	if config.InputNames["sensor/temp1"] != "living_room" {
		fmt.Printf("❌ FAIL: sensor/temp1 = '%s', expected 'living_room'\n", config.InputNames["sensor/temp1"])
		return
	}
	
	// Test 2: Update topic with different input names
	fmt.Println("2. Updating topic through web form...")
	updateFormData := url.Values{
		"inputs": {"sensor/temp1\nsensor/temp3"}, // Changed second input
		"input_names": {`{"sensor/temp1": "main_room", "sensor/temp3": "kitchen"}`}, // Updated names
		"strategy_id": {"test-strategy"},
		"emit_to_mqtt": {""},
		"noop_unchanged": {"on"},
	}
	
	updateReq := httptest.NewRequest("POST", "/topics/test%2Ftopic/edit", strings.NewReader(updateFormData.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateW := httptest.NewRecorder()
	
	server.ServeHTTP(updateW, updateReq)
	
	if updateW.Code != 302 { // Expect redirect on success
		fmt.Printf("❌ FAIL: Topic update returned %d, expected 302 redirect\n", updateW.Code)
		fmt.Printf("Response: %s\n", updateW.Body.String())
		return
	}
	
	// Check if input names were updated
	updatedConfig := topic.GetConfig()
	fmt.Printf("📊 Updated topic InputNames: %v\n", updatedConfig.InputNames)
	
	if len(updatedConfig.InputNames) != 2 {
		fmt.Printf("❌ FAIL: Expected 2 input names after update, got %d\n", len(updatedConfig.InputNames))
		return
	}
	
	if updatedConfig.InputNames["sensor/temp1"] != "main_room" {
		fmt.Printf("❌ FAIL: After update, sensor/temp1 = '%s', expected 'main_room'\n", updatedConfig.InputNames["sensor/temp1"])
		return
	}
	
	if updatedConfig.InputNames["sensor/temp3"] != "kitchen" {
		fmt.Printf("❌ FAIL: After update, sensor/temp3 = '%s', expected 'kitchen'\n", updatedConfig.InputNames["sensor/temp3"])
		return
	}
	
	// Verify other settings were preserved/updated correctly
	if updatedConfig.EmitToMQTT != false {
		fmt.Printf("❌ FAIL: EmitToMQTT should be false after update, got %v\n", updatedConfig.EmitToMQTT)
		return
	}
	
	if updatedConfig.NoOpUnchanged != true {
		fmt.Printf("❌ FAIL: NoOpUnchanged should be true after update, got %v\n", updatedConfig.NoOpUnchanged)
		return
	}
	
	fmt.Printf("✅ SUCCESS: Input names work correctly in web forms!\n")
	fmt.Printf("  - Topic creation with input names: ✅\n")
	fmt.Printf("  - Topic update with input names: ✅\n") 
	fmt.Printf("  - Other settings preserved: ✅\n")
}
