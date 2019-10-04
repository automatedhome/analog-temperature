package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strconv"

	mqttclient "github.com/automatedhome/common/pkg/mqttclient"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type setpoints struct {
	Vref float64 // cannot be 0 !
	Tmin float64 // temperature on 0V
	Tmax float64 // temperature on Vref
}

var publishTopic string
var settings setpoints

func onMessage(client mqtt.Client, message mqtt.Message) {
	voltage, err := strconv.ParseFloat(string(message.Payload()), 64)
	if err != nil {
		log.Printf("Received incorrect message payload: '%v'\n", message.Payload())
		return
	}

	temperature := calculate(voltage)
	if err := mqttclient.Publish(client, publishTopic, 0, false, fmt.Sprintf("%.2f", temperature)); err != nil {
		log.Fatalln("MQTT message couldn't be sent. Exiting.")
	}
}

func calculate(voltage float64) float64 {
	// normalize voltage
	temperature := (settings.Tmax-settings.Tmin)*voltage/settings.Vref + settings.Tmin
	return temperature
}

func main() {
	broker := flag.String("broker", "tcp://127.0.0.1:1883", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")
	clientID := flag.String("clientid", "analog-temperature", "A clientid for the connection")
	inTopic := flag.String("inTopic", "evok/ai/1/value", "MQTT topic with a current analog value state")
	outTopic := flag.String("outTopic", "solar/temperature/up", "MQTT topic to post a calculated temperature")
	flag.Parse()

	settings.Vref = 12.0
	settings.Tmin = 0   // input is at 0V (minimum)
	settings.Tmax = 200 // input is at 10V (hardware allowed maximum)
	publishTopic = *outTopic
	brokerURL, err := url.Parse(*broker)
	if err != nil {
		log.Fatal(err)
	}
	mqttclient.New(*clientID, brokerURL, []string{*inTopic}, onMessage)

	log.Printf("Connected to %s as %s and waiting for messages\n", *broker, *clientID)

	// wait forever
	select {}
}
