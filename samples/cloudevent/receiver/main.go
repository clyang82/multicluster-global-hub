package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudeventsclient "github.com/cloudevents/sdk-go/v2/client"

	"github.com/stolostron/multicluster-global-hub/samples/config"
)

var (
	counts = 1450
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

	start := rand.Intn(10)
	for i := 0; i < counts; i++ {
		go receiver(bootstrapServer, fmt.Sprintf("mytopic-%d", counts*start+i), fmt.Sprintf("mygroup-%d", i), saramaConfig)
	}

	time.Sleep(30 * time.Minute)
}

func receiver(bootstrapServer, topic, groupId string, saramaConfig *sarama.Config) error {
	receiver, err := kafka_sarama.NewConsumer([]string{bootstrapServer}, saramaConfig, groupId, topic)
	if err != nil {
		log.Printf("failed to create protocol: %s", err.Error())
		return err
	}

	defer receiver.Close(context.Background())

	c, err := cloudevents.NewClient(receiver, cloudeventsclient.WithPollGoroutines(1))
	if err != nil {
		log.Printf("failed to create client, %v", err)
		return err
	}

	log.Printf("will listen consuming topic: %s\n", topic)
	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
		fmt.Printf("%s \n", event.Data())
	})
	if err != nil {
		log.Printf("failed to start receiver: %s", err)
		return err
	} else {
		log.Printf("receiver stopped\n")
	}

	return nil
}
