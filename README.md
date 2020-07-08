# Aquarea2mqtt
Panasonic Aquarea Service Cloud to MQTT gateway. Intended for Home Assistant integration, though not quite there yet.


Configuration 
Create config.json file based on config.example.json.

values: 

```
AquareaServiceCloudURL="https://aquarea-service.panasonic.com/" < base URL for aquarea Service Cloud 
AquareaSmartCloudURL="https://aquarea-smart.panasonic.com/" < base URL for aquarea Smart Cloud
AquareaServiceCloudLogin="" < Aquarea Service Cloud login !!! it's not the same like for a smart cloud!!
AquareaServiceCloudPassword="" < Aquarea Service Cloud password !!! it's not the same like for a smart cloud!!
AquateaTimeout="4s" < time to wait for Auarea response
MqttServer="" 
MqttPort=1883
MqttLogin="test"
MqttPass="testpass"
MqttClientID="aquarea-test-pub"
MqttKeepalive="60s"  < MQTT keepalive timeour
PoolInterval="20s" < Update interval(from Aquarea service)
LogSecOffset=500 <number of seconds for searching last statistic information from Aquarea Service Cloud
```


published topics :
- pretty much everything from Device informatio, Statistics and User settings  
   
 
  home assistant config examples (outdated):
  
  ```

  climate:
  - platform: mqtt
    name: HeatPumpSetpoint
    initial: 0
    min_temp: -5
    max_temp: 5
    modes:
      - "auto"
    current_temperature_topic: "aquarea/state/B76<REST OF DEVICE ID>/Zone1SetpointTemperature"
    temperature_command_topic: "aquarea/B76<REST OF DEVICE ID>/Zone1SetpointTemperature/set"
    precision: 1.0
	
binary_sensor:
   - platform: mqtt
    name: "HeatPump DefrostStatus"
    state_topic: "aquarea/state/B76<REST OF DEVICE ID>/DefrostStatus"
	
sensor:
  - platform: mqtt
    name: "HeatPump Zone1WaterTemperature"
    unit_of_measurement: '°C'
    state_topic: "aquarea/state/B76<REST OF DEVICE ID>/Zone1WaterTemperature"
```


TODO:
	- Test on ServiceCloud  with more than one heatpump
	- test with heatpump equiped with option board etc
