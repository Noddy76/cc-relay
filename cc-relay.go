package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
)

type powerChannel struct {
	Reading string `xml:"watts"`
}

type message struct {
	XMLName     xml.Name     `xml:"msg"`
	Ch1         powerChannel `xml:"ch1"`
	Ch2         powerChannel `xml:"ch2"`
	Ch3         powerChannel `xml:"ch3"`
	Temperature float32      `xml:"tmpr"`
}

func connect(clientID string, broker string) mqtt.Client {
	opts := createClientOptions(clientID, broker)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

func createClientOptions(clientID string, broker string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	return opts
}

func sendValue(client mqtt.Client, control string, value string) {
	topic := fmt.Sprintf("/devices/%s/controls/%s", deviceName, control)
	token := client.Publish(topic, 1, false, value)
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
}

func relayBlob(client mqtt.Client, blob string) {
	var msg message
	err := xml.Unmarshal([]byte(blob), &msg)
	if err != nil {
		return
	}

	fmt.Println(msg)

	ch1 := strings.TrimLeft(msg.Ch1.Reading, "0")
	if ch1 == "" {
		ch1 = "0"
	}
	sendValue(client, "ch1", ch1)

	ch2 := strings.TrimLeft(msg.Ch2.Reading, "0")
	if ch2 == "" {
		ch2 = "0"
	}
	sendValue(client, "ch2", ch2)

	ch3 := strings.TrimLeft(msg.Ch3.Reading, "0")
	if ch3 == "" {
		ch3 = "0"
	}
	sendValue(client, "ch3", ch3)

	sendValue(client, "temperature", fmt.Sprintf("%f", msg.Temperature))
}

var deviceName string
var serialPort string
var mqttHost string

func main() {
	flag.StringVar(&deviceName, "device", "", "MQTT Device name")
	flag.StringVar(&serialPort, "serial", "", "Serial device for CC")
	flag.StringVar(&mqttHost, "host", "", "MQTT Broker")
	flag.Parse()

	if deviceName == "" {
		panic("Missing required argument device")
	}
	if serialPort == "" {
		panic("Missing required argument serial")
	}
	if mqttHost == "" {
		panic("Missing required argument host")
	}

	client := connect("cc-relay", mqttHost)

	c := &serial.Config{Name: serialPort, Baud: 9600}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(s)
	for scanner.Scan() {
		blob := scanner.Text()
		relayBlob(client, blob)
	}
}
