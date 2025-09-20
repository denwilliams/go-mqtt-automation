#!/usr/bin/env python3
"""
MQTT Test Data Simulator for Home Automation System

This script publishes realistic test data to MQTT topics to demonstrate
the automation system's capabilities.

Requirements:
    pip install paho-mqtt

Usage:
    python scripts/test-mqtt-simulator.py
"""

import json
import random
import time
import threading
from datetime import datetime
import argparse
import paho.mqtt.client as mqtt

class MQTTSimulator:
    def __init__(self, broker_host="localhost", broker_port=1883):
        self.client = mqtt.Client()
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.running = False
        
        # Sensor state
        self.temperature_base = 22.0
        self.humidity_base = 45.0
        self.light_level = 50.0
        self.motion_sensors = {
            "sensors/motion/living-room": False,
            "sensors/motion/kitchen": False,
            "sensors/motion/bedroom": False,
        }
        
    def connect(self):
        """Connect to MQTT broker"""
        try:
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()
            print(f"✅ Connected to MQTT broker at {self.broker_host}:{self.broker_port}")
            return True
        except Exception as e:
            print(f"❌ Failed to connect to MQTT broker: {e}")
            return False
    
    def disconnect(self):
        """Disconnect from MQTT broker"""
        self.running = False
        self.client.loop_stop()
        self.client.disconnect()
        print("🔌 Disconnected from MQTT broker")
    
    def publish(self, topic, payload):
        """Publish message to topic"""
        if isinstance(payload, dict):
            payload = json.dumps(payload)
        result = self.client.publish(topic, payload)
        if result.rc == 0:
            print(f"📤 {topic}: {payload}")
        else:
            print(f"❌ Failed to publish to {topic}")
    
    def simulate_temperature_sensors(self):
        """Simulate temperature readings with gradual changes"""
        locations = ["living-room", "kitchen", "bedroom", "outdoor"]
        
        for location in locations:
            # Add some variation based on location
            variation = random.uniform(-2.0, 2.0)
            if location == "outdoor":
                variation *= 2  # Outdoor varies more
            elif location == "kitchen":
                variation += 1.0  # Kitchen tends to be warmer
            
            temp = self.temperature_base + variation
            # Add small random fluctuation
            temp += random.uniform(-0.5, 0.5)
            
            # Celsius
            self.publish(f"sensors/temperature/{location}", round(temp, 1))
            
            # Also publish Celsius for the converter test
            if location == "living-room":
                self.publish("sensors/temperature/celsius", round(temp, 1))
    
    def simulate_humidity_sensors(self):
        """Simulate humidity readings"""
        locations = ["living-room", "kitchen", "bedroom"]
        
        for location in locations:
            humidity = self.humidity_base + random.uniform(-10, 10)
            humidity = max(20, min(80, humidity))  # Keep in realistic range
            self.publish(f"sensors/humidity/{location}", round(humidity, 1))
    
    def simulate_light_sensors(self):
        """Simulate light level changes throughout the day"""
        current_hour = datetime.now().hour
        
        # Simulate day/night cycle
        if 6 <= current_hour <= 20:  # Daytime
            base_light = 70 + random.uniform(-20, 20)
        elif 21 <= current_hour <= 23 or 0 <= current_hour <= 5:  # Night
            base_light = 10 + random.uniform(-5, 15)
        else:  # Dawn/dusk
            base_light = 40 + random.uniform(-15, 15)
        
        self.light_level = max(0, min(100, base_light))
        self.publish("sensors/light", round(self.light_level, 1))
    
    def simulate_motion_sensors(self):
        """Simulate motion detection with realistic patterns"""
        current_hour = datetime.now().hour
        
        # Higher motion probability during day
        motion_probability = 0.3 if 6 <= current_hour <= 22 else 0.1
        
        for sensor_topic in self.motion_sensors:
            current_state = self.motion_sensors[sensor_topic]
            
            if current_state:
                # If motion is active, 70% chance to turn off
                if random.random() < 0.7:
                    self.motion_sensors[sensor_topic] = False
                    self.publish(sensor_topic, False)
            else:
                # If no motion, check probability to activate
                if random.random() < motion_probability:
                    self.motion_sensors[sensor_topic] = True
                    self.publish(sensor_topic, True)
    
    def simulate_device_status(self):
        """Simulate various device status messages"""
        devices = {
            "devices/thermostat/status": {
                "temperature": self.temperature_base,
                "target": 21,
                "mode": "auto",
                "heating": random.choice([True, False])
            },
            "devices/smart-plug/power": random.uniform(10, 150),
            "devices/door-sensor": random.choice(["open", "closed"]),
            "devices/window-sensor": random.choice(["open", "closed"]),
        }
        
        for topic, value in devices.items():
            self.publish(topic, value)
    
    def simulate_test_scenarios(self):
        """Publish specific test data for strategy testing"""
        # Test threshold detector with varying values
        test_value = 20 + (time.time() % 20)  # Oscillates between 20-40
        self.publish("test/threshold-sensor", round(test_value, 1))
        
        # Test toggle trigger
        if random.random() < 0.2:  # 20% chance to trigger toggle
            self.publish("test/toggle-button", True)
        
        # Test multiple sensors for averaging
        for i in range(3):
            temp = self.temperature_base + random.uniform(-3, 3)
            self.publish(f"test/temp-sensor-{i+1}", round(temp, 1))
    
    def run_simulation(self, duration=None):
        """Run the simulation"""
        if not self.connect():
            return
        
        self.running = True
        start_time = time.time()
        cycle_count = 0
        
        print("🚀 Starting MQTT simulation...")
        print("📊 Publishing test data every 5 seconds")
        print("Press Ctrl+C to stop\n")
        
        try:
            while self.running:
                if duration and (time.time() - start_time) > duration:
                    break
                
                cycle_count += 1
                print(f"\n--- Simulation Cycle {cycle_count} ---")
                
                # Simulate all sensor types
                self.simulate_temperature_sensors()
                self.simulate_humidity_sensors()
                self.simulate_light_sensors()
                self.simulate_motion_sensors()
                self.simulate_device_status()
                self.simulate_test_scenarios()
                
                # Gradually change base values for more realistic simulation
                self.temperature_base += random.uniform(-0.1, 0.1)
                self.temperature_base = max(18, min(30, self.temperature_base))
                
                self.humidity_base += random.uniform(-0.5, 0.5)
                self.humidity_base = max(30, min(70, self.humidity_base))
                
                time.sleep(5)
                
        except KeyboardInterrupt:
            print("\n⏹️ Simulation stopped by user")
        finally:
            self.disconnect()

def main():
    parser = argparse.ArgumentParser(description="MQTT Test Data Simulator")
    parser.add_argument("--host", default="localhost", help="MQTT broker host")
    parser.add_argument("--port", type=int, default=1883, help="MQTT broker port")
    parser.add_argument("--duration", type=int, help="Simulation duration in seconds")
    parser.add_argument("--scenario", choices=["basic", "motion", "temperature", "all"], 
                       default="all", help="Test scenario to run")
    
    args = parser.parse_args()
    
    simulator = MQTTSimulator(args.host, args.port)
    
    if args.scenario == "all":
        simulator.run_simulation(args.duration)
    else:
        # Could implement specific scenarios here
        print(f"Running {args.scenario} scenario...")
        simulator.run_simulation(args.duration)

if __name__ == "__main__":
    main()