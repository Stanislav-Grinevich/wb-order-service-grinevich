package main

/*1 вариант кода мейн
func main() {
	dsn := "postgres://stas:12345017@localhost:5432/orderservice"
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}
	defer conn.Close(context.Background())
	fmt.Println("Подключение к PostgreSQL успешно!")
}*/

/*2 вариант кода мейн
func main() {
	ctx := context.Background()
	pool, err := db.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	fmt.Println("подключение к постгресс прошло успешно")
}*/

//3 вариант кода мейн
/*
import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/db"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/httpserver"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	rp := repo.NewOrdersRepo(pool)
	cc := cache.New()

	// прогрев кэша: если пусто — положим тестовый заказ
	orders, err := rp.LoadAllOrders(ctx, 200)
	if err != nil {
		log.Fatal(err)
	}
	if len(orders) == 0 {
		if err := rp.InsertTestOrder(ctx); err != nil {
			log.Fatal(err)
		}
		orders, _ = rp.LoadAllOrders(ctx, 1)
	}
	cc.Load(orders)
	log.Printf("cache warmup: %d orders", len(orders))

	srv := httpserver.New(cc, rp)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Println("HTTP server listening on :8081")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
*/

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/db"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/httpserver"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/kafkaconsumer"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// DB
	pool, err := db.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	rp := repo.NewOrdersRepo(pool)
	cc := cache.New()

	// прогрев кэша
	orders, err := rp.LoadAllOrders(ctx, 200)
	if err != nil {
		log.Fatal(err)
	}
	if len(orders) == 0 {
		if err := rp.InsertTestOrder(ctx); err != nil {
			log.Fatal(err)
		}
		orders, _ = rp.LoadAllOrders(ctx, 1)
	}
	cc.Load(orders)
	log.Printf("cache warmup: %d orders", len(orders))

	// HTTP
	srv := httpserver.New(cc, rp)
	server := &http.Server{
		Addr:              ":8081",
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		fmt.Println("HTTP server listening on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Kafka consumer
	kcfg := kafkaconsumer.Config{
		Brokers: splitCSV(os.Getenv("KAFKA_BROKERS")),
		Topic:   os.Getenv("KAFKA_TOPIC"),
		GroupID: os.Getenv("KAFKA_GROUP"),
	}
	consumer := kafkaconsumer.New(kcfg, rp, cc)
	defer consumer.Close()
	go consumer.Run(ctx)

	<-ctx.Done()
	log.Println("shutting down...")

	// аккуратный стоп HTTP
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
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
