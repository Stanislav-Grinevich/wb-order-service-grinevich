// Package kafkaconsumer реализует чтение сообщений из Kafka и их обработку.
package kafkaconsumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"

	kafka "github.com/segmentio/kafka-go"
)

// Consumer обрабатывает сообщения.
type Consumer struct {
	reader *kafka.Reader
	repo   *repo.OrdersRepo
	cache  *cache.Cache
}

// Config задаёт параметры подключения к Kafka.
type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

// New создаёт консьюмера с ручным коммитом оффсетов.
func New(cfg Config, r *repo.OrdersRepo, c *cache.Cache) *Consumer {
	rd := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 0,
	})
	return &Consumer{reader: rd, repo: r, cache: c}
}

func (c *Consumer) Close() error { return c.reader.Close() }

// validate проверяет обязательные поля заказа.
func validate(o *models.Order) error {
	if o.OrderUID == "" {
		return errors.New("order_uid empty")
	}
	if o.Payment.Transaction == "" {
		return errors.New("payment.transaction empty")
	}
	if o.DateCreated.IsZero() {
		return errors.New("date_created empty")
	}
	return nil
}

// Run запускает бесконечный цикл чтения и обработки сообщений Kafka.
func (c *Consumer) Run(ctx context.Context) {
	log.Println("[kafka] consumer started")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[kafka] stopped:", ctx.Err())
				return
			}
			log.Println("[kafka] read error:", err)
			continue
		}

		// убираем BOM, если есть
		payload := bytes.TrimPrefix(m.Value, []byte{0xEF, 0xBB, 0xBF})

		var o models.Order
		if err := json.Unmarshal(payload, &o); err != nil {
			log.Printf("[kafka] bad json (offset %d): %v", m.Offset, err)
			continue
		}
		if err := validate(&o); err != nil {
			log.Printf("[kafka] invalid order (offset %d): %v", m.Offset, err)
			continue
		}

		// запись в БД
		if err := c.repo.InsertOrUpdateOrder(ctx, o); err != nil {
			log.Printf("[kafka] db error (offset %d): %v", m.Offset, err)
			continue // без коммита → сообщение повторится
		}

		// обновление кэша
		c.cache.Set(o)

		// лог успешной обработки
		log.Printf("[kafka] stored order %s (offset %d)", o.OrderUID, m.Offset)

		// ручной коммит оффсета
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("[kafka] commit error (offset %d): %v", m.Offset, err)
		}
	}
}
