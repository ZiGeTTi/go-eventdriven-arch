package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoDBConnectionString string
	MongoDBDatabaseName     string
	RabbitMQHostName        string
	RabbitMQExchange        string
	RabbitMQQueueName       string
}

func LoadConfig() (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables only")
		// Continue without .env file, use environment variables
	}

	config := &Config{
		MongoDBConnectionString: os.Getenv("MONGODB_CONNECTION_STRING"),
		MongoDBDatabaseName:     os.Getenv("MONGODB_DATABASE_NAME"),
		RabbitMQHostName:        os.Getenv("RABBITMQ_HOSTNAME"),
		RabbitMQExchange:        os.Getenv("RABBITMQ_EXCHANGE"),
		RabbitMQQueueName:       os.Getenv("RABBITMQ_QUEUENAME"),
	}

	// Set default values if environment variables are not set
	if config.MongoDBDatabaseName == "" {
		config.MongoDBDatabaseName = "order-db"
	}
	if config.RabbitMQExchange == "" {
		config.RabbitMQExchange = "order_events"
	}
	if config.RabbitMQQueueName == "" {
		config.RabbitMQQueueName = "order_events_queue"
	}

	return config, nil
}
