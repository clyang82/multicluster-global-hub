package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/stolostron/multicluster-global-hub/samples/config"
)

var (
	counts = 100
)

func main() {
	bootstrapServer, saramaConfig, err := config.GetSaramaConfig()
	if err != nil {
		log.Fatalf("failed to get sarama config: %v", err)
	}
	// if set this to false, it will consume message from beginning when restart the client,
	// otherwise it will consume message from the last committed offset.
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	for i := 0; i <= counts; i++ {
		go receiver(bootstrapServer, fmt.Sprintf("mytopic-%d", i), fmt.Sprintf("mygroup-%d", i), saramaConfig)
	}

	time.Sleep(30 * time.Minute)
}

func receiver(bootstrapServer, topic, groupId string, saramaConfig *sarama.Config) {
	receiver, err := kafka_sarama.NewConsumer([]string{bootstrapServer}, saramaConfig, groupId, topic)
	if err != nil {
		log.Fatalf("failed to create protocol: %s", err.Error())
	}

	defer receiver.Close(context.Background())

	c, err := cloudevents.NewClient(receiver)
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	log.Printf("will listen consuming topic: %s\n", topic)
	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
		fmt.Printf("%s \n", event.Data())
	})
	if err != nil {
		log.Fatalf("failed to start receiver: %s", err)
	} else {
		log.Printf("receiver stopped\n")
	}
}
