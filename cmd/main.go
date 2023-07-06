package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	mqttclient "github.com/automatedhome/common/pkg/mqttclient"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type setpoints struct {
	Vref float64 // cannot be 0 !
	Tmin float64 // temperature on 0V
	Tmax float64 // temperature on Vref
}

var (
	publishTopic string
	settings     setpoints
	tempMetric   = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "analog_sensor_temperature",
		Help: "Temperature from analog sensor",
		//ConstLabels: map[string]string{
		//	"sensor": "ai1",  //TODO: this should be populated from user provided flags
		//},
	})
	voltMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "analog_sensor_voltage",
		Help: "Voltage from analog sensor",
	})
	msgReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "analog_sensor_messages_received",
		Help: "The total number of messages received",
	})
)

func onMessage(client mqtt.Client, message mqtt.Message) {
	voltage, err := strconv.ParseFloat(string(message.Payload()), 64)
	if err != nil {
		log.Printf("Received incorrect message payload: '%v'\n", message.Payload())
		return
	}
	voltMetric.Set(voltage)

	msgReceived.Inc()

	temperature := calculate(voltage)
	tempMetric.Set(temperature)
	send(client, temperature)
}

func send(client mqtt.Client, temperature float64) {
	topic := publishTopic
	if err := mqttclient.Publish(client, topic, 0, false, fmt.Sprintf("%.2f", temperature)); err != nil {
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

	// Expose metrics
	http.Handle("/metrics", promhttp.Handler())

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

	go func() {
		if err := http.ListenAndServe(":7005", nil); err != nil {
			panic("HTTP Server failed: " + err.Error())
		}
	}()

	// wait forever
	select {}
}
