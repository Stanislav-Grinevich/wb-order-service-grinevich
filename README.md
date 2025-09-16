# WB Order Service

## Предпосылки
- Docker + Docker Compose
- Go 1.24+

## Запуск Kafka
# используется Kafka в режиме KRaft (ZooKeeper не нужен)
docker compose -f docker-compose.kafka.yml up -d

## Запуск Postgres
# контейнер уже создан как demo_postgres
docker start demo_postgres

## Переменные окружения (Windows PowerShell)
$env:POSTGRES_DSN="postgres://USER:PASSWORD@localhost:5432/orderservice?sslmode=disable"
$env:KAFKA_BROKERS="127.0.0.1:9092"
$env:KAFKA_TOPIC="wb_orders"
$env:KAFKA_GROUP="wb-orders-consumer"

## Запуск сервиса
go run .

## Отправка тестового заказа
go run ./cmd/producer model.json

## API
GET http://localhost:8081/order/<order_uid>
GET http://localhost:8081/healthz
UI: http://localhost:8081/

## Troubleshooting
- Если продюсер/консьюмер лезет в [::1]:9092  выставь KAFKA_BROKERS=127.0.0.1:9092.
- Если отправляешь старое сообщение новой группе  StartOffset=LastOffset, пришли заново или смени группу.
- Если JSON с BOM  консьюмер его обрежет (есть TrimPrefix).