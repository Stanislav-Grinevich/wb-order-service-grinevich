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
