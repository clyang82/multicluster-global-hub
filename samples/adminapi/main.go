package main

import (
	"context"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stolostron/multicluster-global-hub/samples/config"
)

var (
	counts = 40000
)

func main() {

	bootstrapServer, _, err := config.GetSaramaConfig()
	if err != nil {
		log.Fatalf("failed to get sarama config: %v", err)
	}
	// create admin client
	a, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServer,
		"security.protocol":        "SSL",
		"ssl.ca.location":          "ca.crt",
		"ssl.certificate.location": "client.crt",
		"ssl.key.location":         "client.key",
	})
	if err != nil {
		log.Fatalf("failed to create admin client: %s", err.Error())
	}
	for i := 0; i <= counts; i++ {
		_, err = a.CreateTopics(context.Background(), []kafka.TopicSpecification{
			{
				Topic:             fmt.Sprintf("mytopic-%d", i),
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		})
		if err != nil {
			log.Fatalf("failed to create topic: %s", err.Error())
		}

	}

}

func createTopicWithAcls(bootstrapServer string) {
	a, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServer,
		"security.protocol":        "SSL",
		"ssl.ca.location":          "ca.crt",
		"ssl.certificate.location": "client.crt",
		"ssl.key.location":         "client.key",
	})
	if err != nil {
		log.Fatalf("failed to create admin client: %s", err.Error())
	}

	_, err = a.CreateTopics(context.Background(), []kafka.TopicSpecification{
		{
			Topic:             "mytopic",
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	})
	if err != nil {
		log.Fatalf("failed to create topic: %s", err.Error())
	}

	_, err = a.CreateACLs(context.Background(), []kafka.ACLBinding{
		{
			Type:                kafka.ResourceTopic,
			Name:                "mytopic",
			ResourcePatternType: kafka.ResourcePatternTypeLiteral,
			Principal:           "User:CN=user1",
			Host:                "*",
			Operation:           kafka.ACLOperationAll,
			PermissionType:      kafka.ACLPermissionTypeAllow,
		},
	})
	if err != nil {
		log.Fatalf("failed to create acls: %s", err.Error())
	}
}
