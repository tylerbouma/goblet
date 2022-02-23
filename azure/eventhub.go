package azure

import (
	"context"
	"log"
	"time"

	aad "github.com/Azure/azure-amqp-common-go/aad"
	eventhubs "github.com/Azure/azure-event-hubs-go"
)

func sendMsg(msg string) {
	tokenProvider, err := aad.NewJWTProvider(aad.JWTProviderWithEnvironmentVars())
	if err != nil {
		log.Fatal("failed to configure AAD JWT provider")
	}

	hub, err := eventhubs.NewHub("namespaceName", "hubName", tokenProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer hub.Close(ctx)
	if err != nil {
		log.Fatal("failed to get hub")
	}

	ctx = context.Background()
	hub.Send(ctx, eventhubs.NewEventFromString(msg))

	defer cancel()
}
