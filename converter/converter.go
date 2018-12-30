package converter

import (
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/koestler/go-mqtt-to-influxdb/influxDbClient"
	"github.com/koestler/go-mqtt-to-influxdb/mqttClient"
)

type Converter struct {
	config                     Config
	influxDbClientPoolInstance *influxDbClient.ClientPool
}

type Config interface {
	Name() string
	Implementation() string
	TargetMeasurement() string
	MqttTopics() []string
	InfluxDbClients() []string
	LogHandleOnce() bool
}

type Output interface {
	WriteRawPoints(rawPoints []influxDbClient.RawPoint, receiverNames []string)
}

func RunConverter(
	config Config,
	mqttClientInstance *mqttClient.MqttClient,
	influxDbClientPoolInstance *influxDbClient.ClientPool,
) (err error) {
	handleFunc, err := getHandler(config.Implementation())
	if err != nil {
		return
	}

	converter := Converter{
		config: config,
		influxDbClientPoolInstance: influxDbClientPoolInstance,
	}

	for _, mqttTopic := range config.MqttTopics() {
		if err := mqttClientInstance.Subscribe(mqttTopic, getMqttMessageHandler(&converter, handleFunc)); err != nil {
			return err
		}
	}

	return nil
}

func (c *Converter) Name() string {
	return c.config.Name()
}

func getMqttMessageHandler(converter *Converter, handleFunc HandleFunc) mqtt.MessageHandler {
	if converter.config.LogHandleOnce() {
		return func(client mqtt.Client, message mqtt.Message) {
			logTopicOnce(converter.Name(), message.Topic())
			handleFunc(converter.config, converter.influxDbClientPoolInstance, message)
		}
	}
	return func(client mqtt.Client, message mqtt.Message) {
		handleFunc(converter.config, converter.influxDbClientPoolInstance, message)
	}
}
