# KV Store

Профессиональное ключ-значение хранилище с сетевым API для Go.

## Возможности

- 🚀 **HTTP API** — удалённый доступ с любого сервера
- 📦 **Go-пакет** — удобная клиентская библиотека
- 🖥️ **CLI-утилита** — работа из командной строки
- 🐳 **Docker** — готов к развёртыванию
- 🔒 **Потокобезопасность** — RWMutex для конкурентного доступа
- 📝 **Индексация** — быстрая загрузка и поиск

## Структура проекта

```
KVStore/
├── cmd/
│   ├── server/     # Сервер с HTTP API
│   └── cli/        # CLI-утилита
├── pkg/
│   └── client/     # Клиентская библиотека
├── internal/
│   ├── storage/    # Хранилище
│   └── server/     # HTTP обработчики
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Быстрый старт

### Запуск сервера

```bash
# Локально
go run ./cmd/server -addr :8080 -data ./data

# Docker
docker-compose up -d
```

### Использование CLI

```bash
# Сборка CLI
go build -o kvcli ./cmd/cli

# Команды
./kvcli set mykey file.txt
./kvcli get mykey downloaded.txt
./kvcli delete mykey
./kvcli list
./kvcli exists mykey
./kvcli health

# Подключение к удалённому серверу
./kvcli -server http://192.168.1.100:8080 list
```

## Использование в Go коде

### Установка клиента

```bash
go get github.com/admin/kvstore/pkg/client
```

### Пример использования

```go
package main

import (
    "fmt"
    "github.com/admin/kvstore/pkg/client"
)

func main() {
    // Создание клиента
    cli, err := client.NewClient(client.Config{
        BaseURL: "http://localhost:8080",
        Timeout: 30 * time.Second,
    })
    if err != nil {
        panic(err)
    }
    defer cli.Close()

    // Сохранение строки
    err = cli.SetString("greeting", "Hello, World!")
    if err != nil {
        panic(err)
    }

    // Получение строки
    value, err := cli.GetString("greeting")
    if err != nil {
        panic(err)
    }
    fmt.Println(value) // Hello, World!

    // Сохранение байтов
    err = cli.SetBytes("data", []byte{1, 2, 3, 4})
    if err != nil {
        panic(err)
    }

    // Получение байтов
    data, err := cli.GetBytes("data")
    if err != nil {
        panic(err)
    }

    // Проверка существования
    exists, err := cli.Exists("greeting")
    if err != nil {
        panic(err)
    }
    fmt.Println(exists) // true

    // Список всех ключей
    keys, err := cli.List()
    if err != nil {
        panic(err)
    }
    fmt.Println(keys) // [greeting data]

    // Удаление
    err = cli.Delete("greeting")
    if err != nil {
        panic(err)
    }

    // Проверка здоровья
    healthy, err := cli.Health()
    if err != nil {
        panic(err)
    }
    fmt.Println(healthy) // true
}
```

## API Endpoints

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/v1/set?key=<key>&size=<size>` | Сохранить значение |
| GET | `/api/v1/get?key=<key>` | Получить значение |
| DELETE | `/api/v1/delete?key=<key>` | Удалить ключ |
| GET | `/api/v1/list` | Список всех ключей |
| GET | `/api/v1/exists?key=<key>` | Проверка существования |
| GET | `/health` | Проверка здоровья |

### Примеры curl

```bash
# Сохранить файл
curl -X POST "http://localhost:8080/api/v1/set?key=mykey&size=12" \
  --data-binary "Hello World!"

# Получить файл
curl -o output.txt "http://localhost:8080/api/v1/get?key=mykey"

# Удалить ключ
curl -X DELETE "http://localhost:8080/api/v1/delete?key=mykey"

# Список ключей
curl "http://localhost:8080/api/v1/list"

# Проверка существования
curl "http://localhost:8080/api/v1/exists?key=mykey"

# Health check
curl "http://localhost:8080/health"
```

## Формат хранения

Данные хранятся в бинарном формате:

```
[Header 17 bytes][Key][Value]
```

**Заголовок (17 байт):**
- 4 байта — длина ключа (uint32, LittleEndian)
- 8 байт — длина значения (uint64, LittleEndian)
- 1 байт — флаг удаления (0 = активно, 1 = удалено)

## Конфигурация сервера

```bash
./server -addr :8080 -data ./data
```

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| `-addr` | `:8080` | Адрес для прослушивания |
| `-data` | `./data` | Директория для данных |

## Docker

```bash
# Сборка образа
docker build -t kvstore .

# Запуск контейнера
docker run -d -p 8080:8080 -v kvdata:/root/data kvstore

# Docker Compose
docker-compose up -d
```

## Лицензия

MIT
