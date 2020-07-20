package radioBridge

import (
	"encoding/hex"
	"fmt"
	"strconv"
)

type radioBridge struct {
	version     int
	sequence    int
	messageType messageType
	payload     string
	data        map[string]string
	alert       bool
}

type eventPayload int

const (
	periodic           eventPayload = 0
	tempOverThreshold  eventPayload = 1
	tempUnderThreshold eventPayload = 2
	tempIncrease       eventPayload = 3
	tempDecrease       eventPayload = 4
	humOverThreshold   eventPayload = 5
	humUnderThreshold  eventPayload = 6
	humIncrease        eventPayload = 7
	humDecrease        eventPayload = 8
)

type messageType int

const (
	reset             messageType = 0
	supervisory       messageType = 1
	tamper            messageType = 2
	tempuratureEvent  messageType = 0x0D
	linkQuality       messageType = 0xFB
	rateLimitExceeded messageType = 0xFC
	sensorState       messageType = 0xFD
	downlinkAck       messageType = 0xFF
)

func (event eventPayload) String() string {
	names := [...]string{
		"Periodic Report",
		"Temperature Over Threshold",
		"Temperature Under Threshold",
		"Temperature Increase",
		"Temperature Decrease",
		"Humidity Over Threshold",
		"Humidity Under Threshold",
		"Humidity Increase",
		"Humidity Decrease",
	}

	if event < periodic || event > humDecrease {
		return "Unknown Event"
	}

	return names[event]
}

func (event messageType) String() string {
	names := [...]string{
		"Reset",
		"Supervisory",
		"Tamper",
		"Temperature",
		"Link Quality",
		"Rate Limit Exceeded",
		"Sensor State",
		"Downlink Ack",
	}

	if event < reset || event > sensorState {
		return "Unknown Message"
	}

	return names[event]
}

func New() radioBridge {
	r := radioBridge{}
	r.alert = false
	return r
}

/*
New function creates a new radio bridge sensor given the type and payload
*/
func (s *radioBridge) Decode(sensorType string, pdu string) {
	//Decode pdu to byte array
	data, err := hex.DecodeString(pdu)
	if err != nil {
		panic(err)
	}

	fmt.Println(data)
	//Get common message structure
	s.version = int(data[0] & 0xF0 >> 4)
	s.sequence = int(data[0] & 0x0F)
	s.messageType = (messageType)(data[1])
	s.payload = pdu[2:len(pdu)]

	//Decode the message payload
	// TODO: check for common messages then decode the sensor specific stuff if needed
	payloadData, err := hex.DecodeString(s.payload)
	if err != nil {
		panic(err)
	}
	if s.messageType == reset {
		s.decodeResetEvent(payloadData)
	} else if s.messageType == supervisory {
		s.decodeSupervisoryEvent(payloadData)
	} else if s.messageType == tamper {
		s.decodeTamperEvent(payloadData)
	} else if s.messageType == linkQuality {
		s.decodeLinkQuality(payloadData)
	} else if s.messageType == rateLimitExceeded {
		s.decodeRateLimitEvent(payloadData)
	} else if s.messageType == sensorState {
		s.decodeTestEvent(payloadData)
	} else if s.messageType == tempuratureEvent {
		s.decodeTemperatureEvent(s.payload)
	}
}

func (s radioBridge) HasAlert() bool {
	return s.alert
}

func (s radioBridge) GetData() map[string]string {
	return s.data
}

func (s *radioBridge) decodeTemperatureEvent(payload string) {
	dataMap := make(map[string]string)

	//Convert string to int for bitbashing
	pdu, err := strconv.ParseInt(payload, 16, 0)
	if err != nil {
		fmt.Println(err)
	}

	//Temp
	temp := float32((pdu & 0x007F000000) >> 24)
	tempDecimal := float32((pdu & 0x0000F00000) >> 20)
	tempSign := (pdu & 0x0080000000) >> 31

	temp += (tempDecimal / 10)
	if tempSign == 1 {
		temp *= -1
	}

	//Humidity
	hum := float32(pdu & 0x000000FF00 >> 8)
	humDecimal := float32(pdu & 0x00000000F0 >> 4)
	hum += (humDecimal / 10)

	//Save all data to map
	eventType := (string)((eventPayload)(pdu & 0xFF00000000 >> 32))
	dataMap["Event Payload"] = eventType
	dataMap["Temperature"] = fmt.Sprintf("%f", temp)
	dataMap["Relative Humidity"] = fmt.Sprintf("%f", hum)

	//If an alert event also save to alert map
	if eventType != "Periodic Report" {
		s.alert = true
	}
	s.data = dataMap
}

func (s *radioBridge) decodeResetEvent(data []byte) {
	dataMap := make(map[string]string)

	typeCode := data[0]
	hardwareVersion := data[1]
	firmwareBytes := (int16(data[2]) << 8) | int16(data[3])
	resetCode := (int16(data[4]) << 8) | int16(data[5])

	var firmwareVersion string
	if (firmwareBytes&0x80)>>7 == 0 {
		//Before version 2.0
		major := (firmwareBytes & 0x7F00) >> 8
		minor := firmwareBytes & 0x00FF
		firmwareVersion = fmt.Sprintf("%d.%d", major, minor)
	} else {
		//2.0 and later
		major := (firmwareBytes & 0x7C00) >> 8
		minor := (firmwareBytes & 0x02E0) >> 4
		build := (firmwareBytes & 0x001F)
		firmwareVersion = fmt.Sprintf("%d.%d.%d", major, minor, build)
	}

	dataMap["Sensor Type Code"] = fmt.Sprintf("%d", typeCode)
	dataMap["Hardware Version"] = fmt.Sprintf("%d", hardwareVersion)
	dataMap["Firmware Version"] = firmwareVersion
	dataMap["Reset Code"] = fmt.Sprintf("%d", resetCode)

	s.alert = true
	s.data = dataMap
}

func (s *radioBridge) decodeSupervisoryEvent(data []byte) {
	dataMap := make(map[string]string)

	errorCodes := data[0]
	sensorState := data[1]
	batteryLevel := data[2]
	extenededSensorState := (int32(data[3]) << 24) | (int32(data[4]) << 16) | (int32(data[5]) << 8) | int32(data[6])
	accumulationCount := int16(data[7])<<8 | int16(data[8])

	tamperSinceLastReset := (errorCodes & 0x10) >> 4
	currentTamperState := (errorCodes & 0x08) >> 3
	downlinkError := (errorCodes & 0x04) >> 2
	batteryLow := (errorCodes & 0x02) >> 1
	radioCommErr := errorCodes & 0x01

	if tamperSinceLastReset == 1 {
		dataMap["Tamper Since Reset"] = "true"
	}
	if downlinkError == 1 {
		dataMap["Downlink Error"] = "true"
		s.alert = true
	}
	if batteryLow == 1 {
		dataMap["Batter Low"] = "true"
		s.alert = true
	}
	if radioCommErr == 1 {
		dataMap["Radio Communication Error"] = "true"
		s.alert = true
	}

	dataMap["Sensor State"] = fmt.Sprintf("%X %X", sensorState, extenededSensorState)
	dataMap["Battery Level"] = fmt.Sprintf("%d", batteryLevel)
	dataMap["Accumulation Count"] = fmt.Sprintf("%d", accumulationCount)
	dataMap["Current Tamper State"] = fmt.Sprintf("%d", currentTamperState)

	s.data = dataMap
}

func (s *radioBridge) decodeTamperEvent(data []byte) {
	dataMap := make(map[string]string)

	if data[0] == 0x00 {
		dataMap["Tamper Event"] = "Opened"
	} else {
		dataMap["Tamper Event"] = "Closed"
	}

	s.data = dataMap
	s.alert = true
}

func (s *radioBridge) decodeLinkQuality(data []byte) {
	dataMap := make(map[string]string)

	dataMap["Sub-Band"] = fmt.Sprintf("%d", data[0])
	dataMap["RSSI"] = fmt.Sprintf("%d", data[1])
	dataMap["SNR"] = fmt.Sprintf("%d", data[2])

	s.data = dataMap
}

func (s *radioBridge) decodeRateLimitEvent(data []byte) {
	dataMap := make(map[string]string)

	dataMap["Rate Limit Exceeded"] = "true"

	s.data = dataMap
	s.alert = true
}

func (s *radioBridge) decodeTestEvent(data []byte) {
	dataMap := make(map[string]string)

	dataMap["Sensor State"] = fmt.Sprintf("%d", data[0])

	s.data = dataMap
}

func (s radioBridge) DumpContents() {
	fmt.Printf("%+v", s)
}
