/*
The Senet package decodes commonly used sensors that are supported on the
Senet Network. Each sensor extends its companies protocol as well as the
Senet Network protocol. Each sensor should be created based on the Company
and Sensor Type to be used in AWS.
*/
package senet

import (
	"encoding/json"
	"log"
	"senet/radioBridge"
)

type SenetSensor interface {
	Decode(string, string)
	DumpContents()
	HasAlert() bool
	GetData() map[string]string
}

type SenetPacket struct {
	Ack        bool    `json:ack`
	AckDnMsgId int     `json:ackDnMsgId`
	DevClass   string  `json:devClass`
	DevEui     string  `json:devEui`
	GwEui      string  `json:gwEui`
	JoinId     int     `json:joinId`
	Pdu        string  `json:pdu`
	Port       int     `json:port`
	SeqNo      int     `json:seqNo`
	Txtime     string  `json:txtime`
	Channel    int     `json:"channel"`
	Datarate   int     `json:"datarate"`
	Freq       float64 `json:"freq"`
	Rssi       int     `json:rssi`
	Snr        int     `json:snr`
}

func New(company string, sensorType string, payload string) SenetSensor {

	//Create and return the payload structure
	var sensor SenetSensor
	if company == "RadioBridge" {
		rB := radioBridge.New()
		rB.Decode(sensorType, payload)
		sensor = &rB
	}
	return sensor
}

func DecodeSenetPacket(body string) SenetPacket {
	var senetPayload SenetPacket
	err := json.Unmarshal([]byte(body), &senetPayload)
	if err != nil {
		log.Println(err)
	}

	return senetPayload
}

func (p *SenetPacket) GetDevEUI() string {
	return p.DevEui
}

func (p *SenetPacket) GetPdu() string {
	return p.Pdu
}
