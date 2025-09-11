package main

import (
	"context"
	"fmt"
	"log"

	"github.com/stani/wb-order-service-grinevich/internal/db"
)

/*func main() {
	dsn := "postgres://stas:12345017@localhost:5432/orderservice"
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}
	defer conn.Close(context.Background())
	fmt.Println("Подключение к PostgreSQL успешно!")
}*/

func main() {
	ctx := context.Background()
	pool, err := db.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	fmt.Println("подключение к постгресс прошло успешно")
}
