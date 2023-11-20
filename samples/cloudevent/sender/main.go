package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stolostron/multicluster-global-hub/samples/config"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	counts = 100
)

func main() {
	bootstrapServer, saramaConfig, err := config.GetSaramaConfig()
	if err != nil {
		log.Fatalf("failed to get sarama config: %v", err)
	}
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.MaxMessageBytes = 1024 * 1000 // 1024KB
	for i := 0; i <= counts; i++ {
		go sender(bootstrapServer, fmt.Sprintf("mytopic-%d", i), saramaConfig)
	}
	time.Sleep(30 * time.Minute)
}

func sender(bootstrapServer, topic string, saramaConfig *sarama.Config) {
	sender, err := kafka_sarama.NewSender([]string{bootstrapServer}, saramaConfig, topic)
	if err != nil {
		log.Fatalf("failed to create protocol: %s", err.Error())
	}

	defer sender.Close(context.Background())

	c, err := cloudevents.NewClient(sender, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	if err := wait.PollImmediate(1*time.Second, 30*time.Minute, func() (bool, error) {
		e := cloudevents.NewEvent()
		e.SetID(uuid.New().String())
		e.SetType("com.cloudevents.sample.sent")
		e.SetSource("https://github.com/cloudevents/sdk-go/samples/kafka/sender")
		_ = e.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
			"message": "Hello, World!",
		})

		if result := c.Send(
			// Set the producer message key
			kafka_sarama.WithMessageKey(context.Background(), sarama.StringEncoder(e.ID())),
			e,
		); cloudevents.IsUndelivered(result) {
			log.Printf("failed to send: %v", result)
			return false, nil
		} else {
			log.Printf("accepted: %t", cloudevents.IsACK(result))
			return false, nil
		}

	}); err != nil {
		log.Fatalf("failed to send message, %v", err)
	}

}
