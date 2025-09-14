package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

func main() {
	brokers := env("KAFKA_BROKERS", "localhost:9092")
	topic := env("KAFKA_TOPIC", "wb_orders")

	if len(os.Args) < 2 {
		log.Fatalf("usage: producer <path-to-json> (например: model.json)")
	}
	path := os.Args[1]
	body, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(splitCSV(brokers)...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer w.Close()

	msg := kafka.Message{
		Value: body,
		Time:  time.Now(),
	}
	if err := w.WriteMessages(context.Background(), msg); err != nil {
		log.Fatal("write:", err)
	}
	fmt.Println("message sent to", topic, "from file", path)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
