package converter

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/koestler/go-mqtt-to-influxdb/influxDbClient"
	"log"
	"regexp"
	"time"
)

type TemperatureHumidity struct {
	Temperature float64
	Humidity    float64
}

type SensorMessage struct {
	Time    string
	AM2301  *TemperatureHumidity
	SI7021  *TemperatureHumidity
	DS18B20 *struct {
		Temperature float64
	}
	TempUnit string
}

const tasmotaSensorTopicRegex = "^([^/]*/)*tele/(.*)/SENSOR$"
var tasmotaSensorTopicMatcher = regexp.MustCompile(tasmotaSensorTopicRegex)

func init() {
	registerHandler("tasmota-sensor", tasmotaSensorHandler)
}

func tasmotaSensorHandler(c Config, oup Output, msg mqtt.Message) {
	// parse topic
	matches := tasmotaSensorTopicMatcher.FindStringSubmatch(msg.Topic())
	if len(matches) < 3 {
		log.Printf("tasmota-sensor[%s]: cannot extract device from topic='%s', regex='%s'",
			c.Name(), msg.Topic(), tasmotaSensorTopicRegex,
		)
		return
	}
	device := matches[2]

	// parse payload
	var message SensorMessage
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		log.Printf("tasmota-sensor[%s]: cannot json decode: %s", c.Name(), err)
		return
	}

	// create points
	points := message.toPoints(device)
	if len(points) < 1 {
		log.Printf(
			"tasmota-sensor[%s]: could not extract any sensor data; "+
				"sensor type is probably unknown; known sensors are AM2301, SI7021, DS18B20; payload='%s'",
			c.Name(), msg.Payload(),
		)
		return
	}

	timeStamp, err := parseTime(message.Time)
	if err != nil {
		timeStamp = time.Now()
	}

	oup.WriteRawPoints(points.ToRaw(c.TargetMeasurement(), timeStamp), c.InfluxDbClients())
}

func (v SensorMessage) toPoints(device string) influxDbClient.Points {
	ret := make(influxDbClient.Points, 0, 2)

	if v.AM2301 != nil {
		ret = append(ret, influxDbClient.Point{
			Tags: map[string]string{
				"device": device,
				"field":  "Temperature",
				"unit":   v.TempUnit,
				"sensor": "AM2301",
			},
			Fields: map[string]interface{}{
				"value": v.AM2301.Temperature,
			},
		})

		ret = append(ret, influxDbClient.Point{
			Tags: map[string]string{
				"device": device,
				"field":  "Humidity",
				"unit":   "%",
				"sensor": "AM2301",
			},
			Fields: map[string]interface{}{
				"value": v.AM2301.Humidity,
			},
		})
	}

	if v.SI7021 != nil {
		ret = append(ret, influxDbClient.Point{
			Tags: map[string]string{
				"device": device,
				"field":  "Temperature",
				"unit":   v.TempUnit,
				"sensor": "SI7021",
			},
			Fields: map[string]interface{}{
				"value": v.SI7021.Temperature,
			},
		})

		ret = append(ret, influxDbClient.Point{
			Tags: map[string]string{
				"device": device,
				"field":  "Humidity",
				"unit":   "%",
				"sensor": "SI7021",
			},
			Fields: map[string]interface{}{
				"value": v.SI7021.Humidity,
			},
		})
	}

	if v.DS18B20 != nil {
		ret = append(ret, influxDbClient.Point{
			Tags: map[string]string{
				"device": device,
				"field":  "Temperature",
				"unit":   v.TempUnit,
				"sensor": "DS18B20",
			},
			Fields: map[string]interface{}{
				"value": v.DS18B20.Temperature,
			},
		})
	}

	return ret
}
