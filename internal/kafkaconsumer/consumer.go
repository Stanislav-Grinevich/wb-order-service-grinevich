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

type Consumer struct {
	reader *kafka.Reader
	repo   *repo.OrdersRepo
	cache  *cache.Cache
}

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

func New(cfg Config, r *repo.OrdersRepo, c *cache.Cache) *Consumer {
	rd := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		StartOffset:    kafka.LastOffset, // читаем только новые, если оффсета нет
		CommitInterval: 0,                // ВАЖНО: выключаем авто-коммит — коммитим вручную только при успехе
	})
	return &Consumer{reader: rd, repo: r, cache: c}
}

func (c *Consumer) Close() error { return c.reader.Close() }

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
			// не коммитим → сообщение будет перечитано этой же группой позже (по таймауту/ребалансу)
			continue
		}
		if err := validate(&o); err != nil {
			log.Printf("[kafka] invalid order (offset %d): %v", m.Offset, err)
			// можно коммитить, если хотим «похоронить» мусорные сообщения.
			// но по ТЗ допускается игнор/логирование. Оставим без коммита, чтобы не терять при желании расследовать.
			continue
		}

		// Пишем в БД одной транзакцией
		if err := c.repo.InsertOrUpdateOrder(ctx, o); err != nil {
			log.Printf("[kafka] db error (offset %d): %v", m.Offset, err)
			// оффсет не коммитим → сообщение обработается повторно позже
			continue
		}

		// Обновляем кэш
		c.cache.Set(o)

		// Видимый лог успеха
		log.Printf("[kafka] stored order %s (offset %d)", o.OrderUID, m.Offset)

		// Ручной коммит только после успешной записи
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("[kafka] commit error (offset %d): %v", m.Offset, err)
			// Ничего не делаем: при ошибке коммита сообщение придёт снова
		}
	}
}
