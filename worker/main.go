package main

import (
	"log"
	"os"

	"github.com/payment/activities"
	"github.com/payment/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	TaskQueue = "payment-task-queue"
)

func main() {
	// Get Temporal server address from environment or use default
	temporalAddr := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddr == "" {
		temporalAddr = "localhost:7233"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalAddr,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, TaskQueue, worker.Options{})

	// Register workflow and activities
	w.RegisterWorkflow(workflows.PaymentWorkflow)
	w.RegisterActivity(activities.SendWebhook)

	log.Println("Starting payment worker on task queue:", TaskQueue)
	log.Println("Temporal server:", temporalAddr)

	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
