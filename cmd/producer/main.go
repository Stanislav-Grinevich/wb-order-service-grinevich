// Package main реализует простой продюсер, который читает JSON и публикует его в топик.
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
	// Читаем параметры из переменных окружения:
	// KAFKA_BROKERS — список брокеров, KAFKA_TOPIC   — название топика
	brokers := env("KAFKA_BROKERS", "localhost:9092")
	topic := env("KAFKA_TOPIC", "wb_orders")

	// Аргумент командной строки должен содержать путь до JSONa.
	if len(os.Args) < 2 {
		log.Fatalf("usage: producer <path-to-json> (например: model.json)")
	}
	path := os.Args[1]

	// Читаем содержимое файла в память.
	body, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	// Создаём и настриваем Kafka writer.
	w := &kafka.Writer{
		Addr:         kafka.TCP(splitCSV(brokers)...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer w.Close()

	// Собираем сообщение.
	msg := kafka.Message{
		Value: body,
		Time:  time.Now(),
	}

	// Пишем сообщение.
	if err := w.WriteMessages(context.Background(), msg); err != nil {
		log.Fatal("write:", err)
	}
	fmt.Println("message sent to", topic, "from file", path)
}

// env возвращает значение переменной окружения k или же def, если переменная не установлена.
func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// splitCSV разбивает строку в массив строк, убирая пробелы и пустые элементы.
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
