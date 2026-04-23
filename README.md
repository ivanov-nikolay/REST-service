# REST-service
REST-сервис для агрегации данных об онлайн подписках пользователей.     

Сгенерировать Swagger:
```bash
 swag init -g cmd/app/main.go -o docs
```
Запустить:
```bash
 docker-compose up --build
```
Сервис доступен:
```text
http://localhost:8080
```
Swagger доступен:
```text
http://localhost:8080/swagger/index.html
```
Примеры curl:
```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "start_date": "01-2026",
    "end_date": "12-2026"
  }'
```
```bash
curl http://localhost:8080/subscriptions/55031c10-728a-4d89-b03b-b75dc688e954
```

```bash
curl -X PUT http://localhost:8080/subscriptions/55031c10-728a-4d89-b03b-b75dc688e954 \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Яндекс Плюс",
    "price": 150,
    "start_date": "01-2026",
    "end_date": "10-2026"
  }'
```

```bash
curl -X DELETE http://localhost:8080/subscriptions/55031c10-728a-4d89-b03b-b75dc688e954
```

```bash
curl "http://localhost:8080/subscriptions?page=1&page_size=10"
```
```bash
curl "http://localhost:8080/subscriptions?user_id=550e8400-e29b-41d4-a716-446655440000&page=1&page_size=10"
```
```bash
curl "http://localhost:8080/subscriptions?service_name=Яндекс&start_date_from=01-2026&start_date_to=06-2026"
```

```bash
curl "http://localhost:8080/subscriptions/total-cost?user_id=550e8400-e29b-41d4-a716-446655440000&start_date=01-2026&end_date=12-2026&service_name=Яндекс"
```
```bash
curl "http://localhost:8080/subscriptions/total-cost?user_id=550e8400-e29b-41d4-a716-446655440000&start_date=01-2026&end_date=12-2026"
```