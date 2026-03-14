# 📘 Руководство пользователя KV Store

## Оглавление

1. [Введение](#введение)
2. [Быстрый старт](#быстрый-старт)
3. [Установка и запуск сервера](#установка-и-запуск-сервера)
4. [Работа через CLI](#работа-через-cli)
5. [Использование в Go-коде](#использование-в-go-коде)
6. [HTTP API](#http-api)
7. [Docker и развёртывание](#docker-и-развёртывание)
8. [Примеры использования](#примеры-использования)
9. [Устранение неполадок](#устранение-неполадок)
10. [Частые вопросы](#частые-вопросы)

---

## Введение

**KV Store** — это распределённое ключ-значение хранилище, которое позволяет хранить и получать данные по сети. 

### Для чего это нужно?

Представьте ситуацию:
- У вас есть **Сервер А** (например, веб-сервер с приложением)
- У вас есть **Сервер Б** (отдельный сервер для хранения данных)

Вы запускаете KV Store на Сервере Б, и с Сервера А можете:
- Сохранять файлы, конфиги, кэш
- Получать сохранённые данные
- Управлять хранилищем удалённо

### Основные возможности

| Возможность | Описание |
|-------------|----------|
| 📤 **Set** | Сохранение данных по ключу |
| 📥 **Get** | Получение данных по ключу |
| 🗑️ **Delete** | Удаление данных по ключу |
| 📋 **List** | Получение списка всех ключей |
| ✅ **Exists** | Проверка существования ключа |
| 🏥 **Health** | Проверка доступности сервера |

### Архитектура

```
┌─────────────────┐         HTTP          ┌─────────────────┐
│   Клиентское    │  ──────────────────>  │   KV Store      │
│   приложение    │  <──────────────────  │   Сервер        │
│   (Go, CLI,     │       JSON/Binary     │   (порт 8080)   │
│    curl)        │                       │                 │
└─────────────────┘                       │   ┌─────────┐   │
                                          │   │ Храни-  │   │
                                          │   │ лище    │   │
                                          │   │ data.db │   │
                                          │   └─────────┘   │
                                          └─────────────────┘
```

---

## Быстрый старт

### Шаг 1: Запуск сервера

```bash
# Перейдите в директорию проекта
cd /path/to/KVStore

# Запустите сервер
./server

# Или с параметрами
./server -addr :8080 -data ./data
```

Сервер запущен и готов к работе!

### Шаг 2: Проверка работы

```bash
# Проверьте здоровье сервера
curl http://localhost:8080/health

# Ответ: {"success":true,"data":{"status":"ok"}}
```

### Шаг 3: Сохранение данных

```bash
# Через CLI
./kvcli set mykey "Hello, World!"

# Через curl
curl -X POST "http://localhost:8080/api/v1/set?key=mykey&size=13" \
  --data-binary "Hello, World!"
```

### Шаг 4: Получение данных

```bash
# Через CLI
./kvcli get mykey

# Через curl
curl "http://localhost:8080/api/v1/get?key=mykey"
```

---

## Установка и запуск сервера

### Вариант 1: Запуск из исходников

```bash
# 1. Убедитесь, что установлен Go 1.21+
go version

# 2. Перейдите в директорию проекта
cd /path/to/KVStore

# 3. Скачайте зависимости
go mod download

# 4. Соберите сервер
go build -o server ./cmd/server

# 5. Запустите
./server
```

### Вариант 2: Запуск через go run

```bash
go run ./cmd/server
```

### Вариант 3: Docker

```bash
# Сборка образа
docker build -t kvstore .

# Запуск контейнера
docker run -d \
  --name kvstore \
  -p 8080:8080 \
  -v kvdata:/root/data \
  kvstore

# Или через docker-compose
docker-compose up -d
```

### Параметры командной строки

| Параметр | Короткий | По умолчанию | Описание |
|----------|----------|--------------|----------|
| `--addr` | `-addr` | `:8080` | Адрес и порт для прослушивания |
| `--data` | `-data` | `./data` | Директория для хранения данных |
| `--help` | `-h` | - | Показать справку |

### Примеры запуска

```bash
# Стандартный запуск
./server

# На конкретном порту
./server -addr :9000

# С другой директорией для данных
./server -data /var/lib/kvstore

# На конкретном интерфейсе
./server -addr 192.168.1.100:8080

# Для внешнего доступа
./server -addr 0.0.0.0:8080
```

### Остановка сервера

```bash
# Нажмите Ctrl+C в терминале
# Или отправьте сигнал SIGTERM
kill <PID>
```

---

## Работа через CLI

CLI (Command Line Interface) — утилита для работы с хранилищем из командной строки.

### Установка CLI

```bash
# Сборка
go build -o kvcli ./cmd/cli

# Копирование в PATH (опционально)
cp kvcli /usr/local/bin/
```

### Команды CLI

#### 1. `set` — Сохранение данных

```bash
# Сохранить файл
kvcli set <ключ> <файл>

# Примеры
kvcli set config ./config.json
kvcli set backup ./backup.tar.gz
kvcli set image ./photo.png
```

**Выход:**
```
✅ Файл './config.json' сохранён как 'config' (1234 байт)
```

#### 2. `get` — Получение данных

```bash
# Получить в файл
kvcli get <ключ> [выходной_файл]

# Примеры
kvcli get config
kvcli get config restored_config.json
kvcli get backup ./restored_backup.tar.gz
```

**Выход:**
```
✅ Файл сохранён как 'restored_config.json' (1234/1234 байт)
```

#### 3. `delete` — Удаление данных

```bash
# Удалить по ключу
kvcli delete <ключ>

# Примеры
kvcli delete config
kvcli delete old_backup
```

**Выход:**
```
✅ Ключ 'config' удалён
```

#### 4. `list` — Список всех ключей

```bash
kvcli list
```

**Выход:**
```
📦 Найдено 5 ключей:
  1. config
  2. backup
  3. image
  4. cache
  5. settings
```

#### 5. `exists` — Проверка существования

```bash
kvcli exists <ключ>

# Примеры
kvcli exists config
```

**Выход:**
```
✅ Ключ 'config' существует
```
или
```
❌ Ключ 'config' не найден
```

#### 6. `health` — Проверка здоровья сервера

```bash
kvcli health
```

**Выход:**
```
✅ Сервер здоров
```

### Подключение к удалённому серверу

```bash
# Используйте флаг --server
kvcli --server http://192.168.1.100:8080 list

# Примеры
kvcli -server http://myserver.com:8080 set key file.txt
kvcli -server https://secure-server.com:443 get key
kvcli -server http://10.0.0.5:9000 delete oldkey
```

### Справка CLI

```bash
kvcli -h
```

**Выход:**
```
KV Store CLI - клиент для удалённого хранилища

Использование:
  kvcli set <ключ> <файл>     - сохранить файл
  kvcli get <ключ> [файл]     - получить файл
  kvcli delete <ключ>         - удалить ключ
  kvcli list                  - список ключей
  kvcli exists <ключ>         - проверка существования
  kvcli health                - проверка сервера

Опции:
  -server string
        адрес KV сервера (по умолчанию "http://localhost:8080")
```

---

## Использование в Go-коде

### Установка пакета

```bash
go get github.com/admin/kvstore/pkg/client
```

### Базовый пример

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/admin/kvstore/pkg/client"
)

func main() {
    // Создание клиента
    cli, err := client.NewClient(client.Config{
        BaseURL: "http://localhost:8080",
        Timeout: 30 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Close()

    // Сохранение строки
    err = cli.SetString("greeting", "Hello, World!")
    if err != nil {
        log.Fatal(err)
    }

    // Получение строки
    value, err := cli.GetString("greeting")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(value) // Hello, World!
}
```

### Подробная документация по методам

#### Создание клиента

```go
// Простой способ
cli, err := client.NewClientSimple("http://localhost:8080")

// С полной конфигурацией
cli, err := client.NewClient(client.Config{
    BaseURL:         "http://localhost:8080",
    Timeout:         30 * time.Second,
    MaxIdleConns:    100,
    MaxConnsPerHost: 10,
})
```

#### Метод Set

```go
// Сохранение строки
err := cli.SetString("key", "значение")

// Сохранение байтов
err := cli.SetBytes("key", []byte{1, 2, 3, 4})

// Сохранение из io.Reader (для больших файлов)
file, _ := os.Open("large_file.zip")
defer file.Close()
stat, _ := file.Stat()
err := cli.Set("key", file, stat.Size())
```

#### Метод Get

```go
// Получение строки
value, err := cli.GetString("key")

// Получение байтов
data, err := cli.GetBytes("key")

// Получение потока (для больших файлов)
reader, size, err := cli.Get("key")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Сохранение в файл
outFile, _ := os.Create("restored.zip")
defer outFile.Close()
io.Copy(outFile, reader)
```

#### Метод Delete

```go
err := cli.Delete("key")
```

#### Метод List

```go
keys, err := cli.List()
if err != nil {
    log.Fatal(err)
}
for _, key := range keys {
    fmt.Println(key)
}
```

#### Метод Exists

```go
exists, err := cli.Exists("key")
if err != nil {
    log.Fatal(err)
}
if exists {
    fmt.Println("Ключ существует")
} else {
    fmt.Println("Ключ не найден")
}
```

#### Метод Health

```go
healthy, err := cli.Health()
if err != nil {
    log.Fatal(err)
}
if healthy {
    fmt.Println("Сервер доступен")
}
```

### Продвинутые примеры

#### Пример 1: Кэширование данных

```go
package cache

import (
    "time"
    "github.com/admin/kvstore/pkg/client"
)

type Cache struct {
    client *client.Client
    prefix string
}

func NewCache(serverURL, prefix string) (*Cache, error) {
    cli, err := client.NewClientSimple(serverURL)
    if err != nil {
        return nil, err
    }
    return &Cache{client: cli, prefix: prefix}, nil
}

func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
    fullKey := c.prefix + ":" + key
    return c.client.SetBytes(fullKey, value)
}

func (c *Cache) Get(key string) ([]byte, error) {
    fullKey := c.prefix + ":" + key
    return c.client.GetBytes(fullKey)
}

func (c *Cache) Delete(key string) error {
    fullKey := c.prefix + ":" + key
    return c.client.Delete(fullKey)
}

func (c *Cache) Close() {
    c.client.Close()
}
```

#### Пример 2: Резервное копирование

```go
package backup

import (
    "archive/tar"
    "compress/gzip"
    "io"
    "os"
    "github.com/admin/kvstore/pkg/client"
)

func CreateBackup(cli *client.Client, backupKey string, files []string) error {
    // Создаём временный файл для архива
    tmpFile, err := os.CreateTemp("", "backup-*.tar.gz")
    if err != nil {
        return err
    }
    defer os.Remove(tmpFile.Name())
    defer tmpFile.Close()

    // Создаём tar.gz архив
    gw := gzip.NewWriter(tmpFile)
    tw := tar.NewWriter(gw)

    for _, filePath := range files {
        if err := addToArchive(tw, filePath); err != nil {
            return err
        }
    }

    tw.Close()
    gw.Close()

    // Получаем размер архива
    stat, err := tmpFile.Stat()
    if err != nil {
        return err
    }

    // Отправляем в хранилище
    tmpFile.Seek(0, 0)
    return cli.Set(backupKey, tmpFile, stat.Size())
}

func addToArchive(tw *tar.Writer, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        return err
    }

    header, err := tar.FileInfoHeader(stat, filePath)
    if err != nil {
        return err
    }

    if err := tw.WriteHeader(header); err != nil {
        return err
    }

    _, err = io.Copy(tw, file)
    return err
}
```

#### Пример 3: Распределённая конфигурация

```go
package config

import (
    "encoding/json"
    "github.com/admin/kvstore/pkg/client"
)

type ConfigManager struct {
    client *client.Client
    serviceName string
}

func NewConfigManager(serverURL, serviceName string) (*ConfigManager, error) {
    cli, err := client.NewClientSimple(serverURL)
    if err != nil {
        return nil, err
    }
    return &ConfigManager{
        client: cli,
        serviceName: serviceName,
    }, nil
}

func (cm *ConfigManager) SaveConfig(config interface{}) error {
    data, err := json.Marshal(config)
    if err != nil {
        return err
    }
    key := "config:" + cm.serviceName
    return cm.client.SetBytes(key, data)
}

func (cm *ConfigManager) LoadConfig(config interface{}) error {
    key := "config:" + cm.serviceName
    data, err := cm.client.GetBytes(key)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, config)
}

func (cm *ConfigManager) Close() {
    cm.client.Close()
}
```

---

## HTTP API

### Обзор

KV Store предоставляет REST-like HTTP API для всех операций.

### Базовый URL

```
http://<host>:<port>/api/v1/
```

### Endpoints

#### 1. POST /api/v1/set

Сохраняет данные в хранилище.

**Параметры query:**
- `key` (обязательный) — ключ для хранения
- `size` (обязательный) — размер данных в байтах

**Тело запроса:** бинарные данные

**Пример:**
```bash
curl -X POST "http://localhost:8080/api/v1/set?key=mykey&size=13" \
  -H "Content-Type: application/octet-stream" \
  --data-binary "Hello, World!"
```

**Ответ успеха:**
```json
{
  "success": true,
  "data": {
    "key": "mykey"
  }
}
```

**Ответ ошибки:**
```json
{
  "success": false,
  "error": "описание ошибки"
}
```

#### 2. GET /api/v1/get

Получает данные из хранилища.

**Параметры query:**
- `key` (обязательный) — ключ для получения

**Пример:**
```bash
curl -o output.txt "http://localhost:8080/api/v1/get?key=mykey"
```

**Заголовки ответа:**
- `Content-Type: application/octet-stream`
- `Content-Length: <размер>`
- `X-KV-Size: <размер>`

**Тело ответа:** бинарные данные

**Коды состояния:**
- `200 OK` — успех
- `404 Not Found` — ключ не найден

#### 3. DELETE /api/v1/delete

Удаляет данные из хранилища.

**Параметры query:**
- `key` (обязательный) — ключ для удаления

**Пример:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/delete?key=mykey"
```

**Ответ успеха:**
```json
{
  "success": true,
  "data": {
    "key": "mykey"
  }
}
```

#### 4. GET /api/v1/list

Получает список всех ключей.

**Пример:**
```bash
curl "http://localhost:8080/api/v1/list"
```

**Ответ:**
```json
{
  "success": true,
  "data": ["key1", "key2", "key3"]
}
```

#### 5. GET /api/v1/exists

Проверяет существование ключа.

**Параметры query:**
- `key` (обязательный) — ключ для проверки

**Пример:**
```bash
curl "http://localhost:8080/api/v1/exists?key=mykey"
```

**Ответ:**
```json
{
  "success": true,
  "data": {
    "exists": true
  }
}
```

#### 6. GET /health

Проверяет здоровье сервера.

**Пример:**
```bash
curl "http://localhost:8080/health"
```

**Ответ:**
```json
{
  "success": true,
  "data": {
    "status": "ok"
  }
}
```

### Коды состояния HTTP

| Код | Описание |
|-----|----------|
| 200 | Успех |
| 400 | Неверный запрос (отсутствуют параметры) |
| 404 | Ключ не найден |
| 405 | Метод не поддерживается |
| 500 | Внутренняя ошибка сервера |

---

## Docker и развёртывание

### Dockerfile

Проект включает готовый Dockerfile для сборки образа.

### Сборка образа

```bash
docker build -t kvstore:latest .
```

### Запуск контейнера

```bash
docker run -d \
  --name kvstore \
  -p 8080:8080 \
  -v /path/to/data:/root/data \
  --restart unless-stopped \
  kvstore:latest
```

### Docker Compose

Создайте файл `docker-compose.yml`:

```yaml
version: '3.8'

services:
  kvstore:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - kvstore_data:/root/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  kvstore_data:
```

Запуск:
```bash
docker-compose up -d
```

Остановка:
```bash
docker-compose down
```

### Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `KV_ADDR` | `:8080` | Адрес для прослушивания |
| `KV_DATA` | `/root/data` | Директория для данных |

### Production развёртывание

#### 1. Настройка nginx (опционально)

```nginx
server {
    listen 443 ssl;
    server_name kvstore.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Увеличиваем лимиты для больших файлов
        client_max_body_size 1G;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

#### 2. Systemd сервис

Создайте файл `/etc/systemd/system/kvstore.service`:

```ini
[Unit]
Description=KV Store Server
After=network.target

[Service]
Type=simple
User=kvstore
Group=kvstore
WorkingDirectory=/opt/kvstore
ExecStart=/opt/kvstore/server -addr :8080 -data /var/lib/kvstore
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Активация:
```bash
sudo systemctl daemon-reload
sudo systemctl enable kvstore
sudo systemctl start kvstore
sudo systemctl status kvstore
```

---

## Примеры использования

### Пример 1: Хранение конфигов микросервисов

```go
// Сервис конфигурации
type ServiceConfig struct {
    DatabaseURL     string `json:"database_url"`
    RedisURL        string `json:"redis_url"`
    LogLevel        string `json:"log_level"`
    MaxConnections  int    `json:"max_connections"`
}

func SaveConfig(cli *client.Client, serviceName string, cfg *ServiceConfig) error {
    data, _ := json.Marshal(cfg)
    return cli.SetBytes("config:"+serviceName, data)
}

func LoadConfig(cli *client.Client, serviceName string) (*ServiceConfig, error) {
    data, err := cli.GetBytes("config:" + serviceName)
    if err != nil {
        return nil, err
    }
    var cfg ServiceConfig
    err = json.Unmarshal(data, &cfg)
    return &cfg, err
}
```

### Пример 2: Распределённый кэш

```go
type DistributedCache struct {
    client *client.Client
    ttl    time.Duration
}

func (c *DistributedCache) Set(key string, value []byte) error {
    // Добавляем метку времени для TTL
    entry := CacheEntry{
        Data:      value,
        ExpiresAt: time.Now().Add(c.ttl).Unix(),
    }
    data, _ := json.Marshal(entry)
    return c.client.SetBytes("cache:"+key, data)
}

func (c *DistributedCache) Get(key string) ([]byte, error) {
    data, err := c.client.GetBytes("cache:" + key)
    if err != nil {
        return nil, err
    }
    
    var entry CacheEntry
    if err := json.Unmarshal(data, &entry); err != nil {
        return nil, err
    }
    
    if time.Now().Unix() > entry.ExpiresAt {
        c.client.Delete("cache:" + key)
        return nil, ErrExpired
    }
    
    return entry.Data, nil
}
```

### Пример 3: Очереди задач

```go
type TaskQueue struct {
    client *client.Client
    queueName string
}

func (q *TaskQueue) Push(task []byte) error {
    // Получаем текущую длину очереди
    keys, _ := q.client.List()
    index := len(keys)
    
    key := fmt.Sprintf("queue:%s:%d", q.queueName, index)
    return q.client.SetBytes(key, task)
}

func (q *TaskQueue) Pop() ([]byte, error) {
    // Получаем первый элемент
    keys, err := q.client.List()
    if err != nil || len(keys) == 0 {
        return nil, ErrEmptyQueue
    }
    
    // Находим ключи очереди
    var queueKeys []string
    for _, k := range keys {
        if strings.HasPrefix(k, "queue:"+q.queueName+":") {
            queueKeys = append(queueKeys, k)
        }
    }
    
    if len(queueKeys) == 0 {
        return nil, ErrEmptyQueue
    }
    
    // Получаем и удаляем первый элемент
    data, err := q.client.GetBytes(queueKeys[0])
    if err != nil {
        return nil, err
    }
    
    q.client.Delete(queueKeys[0])
    return data, nil
}
```

### Пример 4: Синхронизация файлов между серверами

```bash
#!/bin/bash
# sync.sh - синхронизация файлов между серверами

KV_SERVER="http://storage-server:8080"
LOCAL_DIR="./files"

# Загрузка файлов в хранилище
upload() {
    for file in $LOCAL_DIR/*; do
        filename=$(basename "$file")
        echo "Загрузка $filename..."
        kvcli -server $KV_SERVER set "file:$filename" "$file"
    done
}

# Скачивание файлов из хранилища
download() {
    keys=$(kvcli -server $KV_SERVER list | grep "file:" | awk '{print $2}')
    for key in $keys; do
        filename=${key#file:}
        echo "Скачивание $filename..."
        kvcli -server $KV_SERVER get "$key" "$LOCAL_DIR/$filename"
    done
}

case "$1" in
    upload) upload ;;
    download) download ;;
    *) echo "Использование: $0 {upload|download}" ;;
esac
```

---

## Устранение неполадок

### Сервер не запускается

**Проблема:** Порт уже занят
```
listen tcp :8080: bind: address already in use
```

**Решение:**
```bash
# Найти процесс на порту
lsof -i :8080

# Убить процесс
kill -9 <PID>

# Или использовать другой порт
./server -addr :9000
```

### Ошибка подключения клиента

**Проблема:** Connection refused

**Причины:**
1. Сервер не запущен
2. Неправильный адрес/порт
3. Брандмауэр блокирует соединение

**Решение:**
```bash
# Проверить статус сервера
curl http://localhost:8080/health

# Проверить доступность порта
telnet localhost 8080

# Проверить брандмауэр
sudo ufw status
```

### Ошибка "ключ не найден"

**Причины:**
1. Ключ действительно не существует
2. Опечатка в имени ключа
3. Ключ был удалён

**Решение:**
```bash
# Проверить список ключей
kvcli list

# Проверить существование
kvcli exists <ключ>
```

### Медленная работа

**Причины:**
1. Большой размер данных
2. Медленный диск
3. Сетевые задержки

**Решение:**
1. Увеличить таймаут клиента
2. Использовать сжатие данных
3. Разместить сервер ближе к клиентам

### Повреждение данных

**Симптомы:**
- Ошибки при чтении
- Несоответствие ключей

**Решение:**
```bash
# Остановить сервер
# Удалить повреждённый файл
rm data/data.db

# Запустить сервер (создастся новый файл)
./server
```

---

## Частые вопросы

### Q: Какой максимальный размер данных можно хранить?

**A:** Теоретически до 4 ГБ на запись (ограничение uint32 для размера). Рекомендуется хранить данные до 100 МБ для лучшей производительности.

### Q: Сколько ключей можно хранить?

**A:** Ограничений нет, но при большом количестве ключей (>1 млн) рекомендуется использовать шардинг.

### Q: Поддерживается ли шифрование?

**A:** В текущей версии нет. Для шифрования используйте HTTPS прокси (nginx) или шифруйте данные на стороне клиента.

### Q: Как сделать резервную копию?

**A:** Просто скопируйте файл `data/data.db`:
```bash
cp data/data.db backup_$(date +%Y%m%d).db
```

### Q: Можно ли изменить данные по существующему ключу?

**A:** Да, просто вызовите `Set` с тем же ключом. Новые данные добавятся в файл, а старая запись будет помечена как удалённая.

### Q: Как очистить всё хранилище?

**A:** Удалите файл данных и перезапустите сервер:
```bash
rm data/data.db
./server
```

### Q: Поддерживаются ли транзакции?

**A:** Нет, хранилище не поддерживает транзакции. Каждая операция атомарна.

### Q: Как мониторить сервер?

**A:** Используйте endpoint `/health` и логи сервера. Можно настроить Prometheus exporter.

---

## Контакты и поддержка

- GitHub: https://github.com/admin/kvstore
- Документация: см. файл HOW_IT_WORKS.md
