package mqtt

import (
	"fmt"
	"github.com/surgemq/message"
	"github.com/suyashkumar/conduit/server/secrets"
	"github.com/suyashkumar/surgemq/service"
	"os"
	"regexp"
	"time"
)

var mClient *service.Client
var handlerMap = make(map[string]func(string, string))

func Register(name string, a func(string, string)) {
	handlerMap[name] = a
}

func onPublish(msg *message.PublishMessage) error {

	if val, ok := handlerMap[string(msg.Topic())]; ok {
		val(string(msg.Topic()), string(msg.Payload()))
		if os.ExpandEnv("LOGGING") != "" {
			fmt.Println("Topic:", string(msg.Topic()), "Payload:", string(msg.Payload()))
		}
	}
	// Look to see if the published message was a streaming data message
	// If so, persist the contents to an appropiate db
	var validDataStream = regexp.MustCompile(`^[^/]*/stream/.*`)
	if validDataStream.MatchString(string(msg.Topic())) {
		go PersistMessage(string(msg.Payload()), string(msg.Topic()))
	}
	return nil
}
func createServerClient() *service.Client {
	service.AllowedMap[secrets.SubSecret] = 1
	client := &service.Client{}
	msg := message.NewConnectMessage()
	msg.SetClientId([]byte(secrets.SubSecret))
	KeepAlive := 40
	msg.SetKeepAlive(uint16(KeepAlive))
	msg.SetCleanSession(true)
	msg.SetVersion(3)
	if err := client.Connect("tcp://:1883", msg); err != nil {
		fmt.Println("problem")
		fmt.Println(err)
	}

	go stayAlive(client, KeepAlive)

	submsg := message.NewSubscribeMessage()
	submsg.AddTopic([]byte("#"), 0)
	client.Subscribe(submsg, nil, onPublish)

	pubMsg := message.NewPublishMessage()
	pubMsg.SetTopic([]byte("suyash1"))
	pubMsg.SetPayload(make([]byte, 10))
	client.Publish(pubMsg, nil)
	return client
}
func stayAlive(c *service.Client, KeepAlive int) {
	for _ = range time.Tick(time.Duration(KeepAlive) * time.Second) {
		c.Ping(func(msg, ack message.Message, err error) error {
			return nil
		})
	}
}
func sendMessage(client *service.Client, device string, payload string) {
	pubMsg := message.NewPublishMessage()
	pubMsg.SetTopic([]byte(device))
	pubMsg.SetPayload([]byte(payload))
	client.Publish(pubMsg, nil)
}

func SendMessage(device string, payload string) {
	sendMessage(mClient, device, payload)
}

func RunServer() {
	fmt.Println("Starting up MQTT machinery...")
	svr := &service.Server{
		KeepAlive: 300,
	}
	go svr.ListenAndServe("tcp://:1883")
	time.Sleep(200 * time.Millisecond)
	mClient = createServerClient()
	fmt.Println("Started and listening")
}
