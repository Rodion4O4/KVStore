# 🔧 Как работает KV Store: Техническое руководство

## Оглавление

1. [Архитектура системы](#архитектура-системы)
2. [Формат хранения данных](#формат-хранения-данных)
3. [Внутреннее устройство хранилища](#внутреннее-устройство-хранилища)
4. [HTTP сервер и обработчики](#http-сервер-и-обработчики)
5. [Клиентская библиотека](#клиентская-библиотека)
6. [Потокобезопасность](#потокобезопасность)
7. [Индексация и загрузка](#индексация-и-загрузка)
8. [Удаление данных](#удаление-данных)
9. [Производительность и оптимизация](#производительность-и-оптимизация)
10. [Расширение функциональности](#расширение-функциональности)

---

## Архитектура системы

### Общая схема

```
┌─────────────────────────────────────────────────────────────────┐
│                         КЛИЕНТЫ                                 │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Go Client     │   CLI Utility   │   HTTP Clients (curl)       │
│   (pkg/client)  │   (cmd/cli)     │                             │
└────────┬────────┴────────┬────────┴──────────────┬──────────────┘
         │                 │                       │
         │    HTTP/1.1     │    HTTP/1.1           │    HTTP/1.1
         │    JSON/Binary  │    JSON/Binary        │    JSON/Binary
         ▼                 ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                      HTTP SERVER                                │
│                   (internal/server)                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Маршрутизатор (http.ServeMux)                            │  │
│  │  /api/v1/set    → handleSet()                             │  │
│  │  /api/v1/get    → handleGet()                             │  │
│  │  /api/v1/delete → handleDelete()                          │  │
│  │  /api/v1/list   → handleList()                            │  │
│  │  /api/v1/exists → handleExists()                          │  │
│  │  /health        → handleHealth()                          │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ХРАНИЛИЩЕ (STORAGE)                          │
│                   (internal/storage)                            │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  LocalStore                                               │  │
│  │  ┌─────────────────┐  ┌─────────────────────────────────┐ │  │
│  │  │ Индекс (map)    │  │ Файл (data.db)                  │ │  │
│  │  │ key → Entry     │  │ [Header][Key][Value]...         │ │  │
│  │  │                 │  │                                 │ │  │
│  │  │ - position      │  │ ┌─────────────────────────────┐ │ │  │
│  │  │ - size          │  │ │ RWMutex                     │ │ │  │
│  │  │ - deleted       │  │ └─────────────────────────────┘ │ │  │
│  │  └─────────────────┘  └─────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   Файловая      │
                    │   система       │
                    │   (data.db)     │
                    └─────────────────┘
```

### Компоненты системы

#### 1. Клиентский уровень (pkg/client)

**Назначение:** Предоставляет удобный Go API для взаимодействия с сервером.

**Компоненты:**
- `Client` — основной класс клиента
- `Config` — конфигурация подключения
- `Response` — структура ответа от сервера

**Методы:**
```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func (c *Client) Set(key string, data io.Reader, size int64) error
func (c *Client) SetBytes(key string, data []byte) error
func (c *Client) SetString(key string, data string) error
func (c *Client) Get(key string) (io.ReadCloser, int64, error)
func (c *Client) GetBytes(key string) ([]byte, error)
func (c *Client) GetString(key string) (string, error)
func (c *Client) Delete(key string) error
func (c *Client) List() ([]string, error)
func (c *Client) Exists(key string) (bool, error)
func (c *Client) Health() (bool, error)
```

#### 2. HTTP сервер (internal/server)

**Назначение:** Обрабатывает HTTP запросы и взаимодействует с хранилищем.

**Компоненты:**
- `KVServer` — основной сервер
- Обработчики запросов (handlers)

**Обработчики:**
```go
func (s *KVServer) handleSet(w http.ResponseWriter, r *http.Request)
func (s *KVServer) handleGet(w http.ResponseWriter, r *http.Request)
func (s *KVServer) handleDelete(w http.ResponseWriter, r *http.Request)
func (s *KVServer) handleList(w http.ResponseWriter, r *http.Request)
func (s *KVServer) handleExists(w http.ResponseWriter, r *http.Request)
func (s *KVServer) handleHealth(w http.ResponseWriter, r *http.Request)
```

#### 3. Хранилище (internal/storage)

**Назначение:** Управление данными на диске.

**Компоненты:**
- `LocalStore` — основная структура
- `Entry` — запись в индексе
- `Store` — интерфейс хранилища

---

## Формат хранения данных

### Структура файла data.db

Файл хранилища представляет собой последовательность записей:

```
┌─────────────────────────────────────────────────────────────┐
│                    ЗАПИСЬ 1                                 │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Заголовок     │      Ключ       │        Значение         │
│   (17 байт)     │   (переменная)  │      (переменная)       │
└─────────────────┴─────────────────┴─────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                    ЗАПИСЬ 2                                 │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Заголовок     │      Ключ       │        Значение         │
│   (17 байт)     │   (переменная)  │      (переменная)       │
└─────────────────┴─────────────────┴─────────────────────────┘
                            ...
```

### Детали заголовка (17 байт)

```
 0                   4                   8    12  13
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
│       Длина ключа (uint32 LE)                   │
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
│       Длина значения (uint64 LE)                │
│                                               │
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
│D│  Зарезервировано (3 бита)                     │
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Где:
- Биты 0-3:   Длина ключа (4 байта, LittleEndian, max 2^32-1)
- Биты 4-11:  Длина значения (8 байт, LittleEndian, max 2^64-1)
- Бит 12:     Флаг удаления (0 = активно, 1 = удалено)
- Биты 13-16: Зарезервировано
```

### Пример записи

Допустим, мы сохраняем:
- Ключ: `"config"` (6 байт)
- Значение: `"hello"` (5 байт)

```
┌─────────────────────────────────────────────────────────────┐
│ Заголовок (17 байт):                                        │
│   [06 00 00 00]  - длина ключа = 6                          │
│   [05 00 00 00 00 00 00 00] - длина значения = 5            │
│   [00] - флаг удаления = 0 (активно)                        │
├─────────────────────────────────────────────────────────────┤
│ Ключ (6 байт):                                              │
│   [63 6f 6e 66 69 67] = "config"                            │
├─────────────────────────────────────────────────────────────┤
│ Значение (5 байт):                                          │
│   [68 65 6c 6c 6f] = "hello"                                │
└─────────────────────────────────────────────────────────────┘
```

### Бинарное представление в Go

```go
const HEADER_SIZE = 17

// Создание заголовка
header := make([]byte, HEADER_SIZE)

// Длина ключа (байты 0-3)
binary.LittleEndian.PutUint32(header[0:4], uint32(len(key)))

// Длина значения (байты 4-11)
binary.LittleEndian.PutUint64(header[4:12], uint64(size))

// Флаг удаления (байт 12)
header[12] = 0 // 0 = активно, 1 = удалено
```

### Чтение заголовка

```go
// Чтение заголовка
header := make([]byte, HEADER_SIZE)
file.ReadAt(header, position)

// Извлечение длины ключа
keylen := binary.LittleEndian.Uint32(header[0:4])

// Извлечение длины значения
vallen := binary.LittleEndian.Uint64(header[4:12])

// Проверка флага удаления
deleted := header[12] != 0
```

---

## Внутреннее устройство хранилища

### Структура LocalStore

```go
type LocalStore struct {
    mu    sync.RWMutex      // Мьютекс для потокобезопасности
    file  *os.File          // Дескриптор файла базы данных
    index map[string]*Entry // Индекс: ключ → запись
    path  string            // Путь к файлу базы
}

type Entry struct {
    Key      string  // Ключ
    Size     int64   // Размер значения
    Deleted  bool    // Флаг удаления
    Position int64   // Позиция в файле (смещение)
}
```

### Операция Set (запись)

**Алгоритм:**

```
1. Захватить Write Lock
2. Получить текущую позицию конца файла (SeekEnd)
3. Создать заголовок (17 байт)
4. Записать заголовок в файл
5. Записать ключ
6. Записать значение
7. Вызвать Sync() для гарантированной записи на диск
8. Обновить индекс
9. Освободить Lock
```

**Код:**

```go
func (s *LocalStore) Set(key string, r io.Reader, size int64) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // 1. Получить позицию для записи
    pos, err := s.file.Seek(0, io.SeekEnd)
    if err != nil {
        return err
    }

    // 2. Создать заголовок
    header := make([]byte, HEADER_SIZE)
    binary.LittleEndian.PutUint32(header[0:4], uint32(len(key)))
    binary.LittleEndian.PutUint64(header[4:12], uint64(size))
    header[12] = 0 // активно

    // 3. Записать заголовок
    if _, err := s.file.Write(header); err != nil {
        return err
    }

    // 4. Записать ключ
    if _, err := s.file.WriteString(key); err != nil {
        s.file.Truncate(pos) // Откат при ошибке
        return err
    }

    // 5. Записать значение
    if _, err := io.CopyN(s.file, r, size); err != nil {
        s.file.Truncate(pos) // Откат при ошибке
        return err
    }

    // 6. Гарантировать запись на диск
    if err := s.file.Sync(); err != nil {
        return err
    }

    // 7. Обновить индекс
    s.index[key] = &Entry{
        Key:      key,
        Size:     size,
        Deleted:  false,
        Position: pos,
    }

    return nil
}
```

**Диаграмма:**

```
Файл до записи:
┌─────────────────────────────────────┐
│ [Запись 1][Запись 2]                │
└─────────────────────────────────────┘
                              ▲
                              │ SeekEnd

Файл после записи:
┌─────────────────────────────────────────────────┐
│ [Запись 1][Запись 2][Header][Key][Value]        │
└─────────────────────────────────────────────────┘
                              ▲                   ▲
                              │                   │
                         позиция pos           конец
```

### Операция Get (чтение)

**Алгоритм:**

```
1. Захватить Read Lock
2. Найти ключ в индексе
3. Проверить флаг удаления
4. Прочитать заголовок по позиции
5. Проверить целостность (ключ, флаг)
6. Создать SectionReader для значения
7. Освободить Lock
8. Вернуть Reader
```

**Код:**

```go
func (s *LocalStore) Get(key string) (io.Reader, int64, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // 1. Найти в индексе
    entry, ok := s.index[key]
    if !ok {
        return nil, 0, fmt.Errorf("ключ не найден: %s", key)
    }

    // 2. Проверить флаг удаления
    if entry.Deleted {
        return nil, 0, fmt.Errorf("запись удалена: %s", key)
    }

    // 3. Прочитать заголовок
    header := make([]byte, HEADER_SIZE)
    if _, err := s.file.ReadAt(header, entry.Position); err != nil {
        return nil, 0, err
    }

    // 4. Проверить флаг в заголовке
    if header[12] != 0 {
        return nil, 0, fmt.Errorf("запись удалена: %s", key)
    }

    // 5. Прочитать и проверить ключ
    keylen := binary.LittleEndian.Uint32(header[0:4])
    keybytes := make([]byte, keylen)
    if _, err := s.file.ReadAt(keybytes, entry.Position+HEADER_SIZE); err != nil {
        return nil, 0, err
    }
    if string(keybytes) != key {
        return nil, 0, errors.New("повреждение данных")
    }

    // 6. Создать Reader для значения
    valpos := entry.Position + HEADER_SIZE + int64(keylen)
    reader := io.NewSectionReader(s.file, valpos, entry.Size)

    return reader, entry.Size, nil
}
```

**Диаграмма чтения:**

```
Файл:
┌─────────────────────────────────────────────────────────────┐
│ [Header:17][Key:6][Value:5]                                 │
│  ▲         ▲        ▲                                       │
│  │         │        │                                       │
│  │         │        └── SectionReader (позиция, длина)      │
│  │         │                                               │
│  │         └── Позиция ключа = pos + HEADER_SIZE           │
│  │                                                         │
│  └── Позиция записи = entry.Position                       │
└─────────────────────────────────────────────────────────────┘
```

### Операция Exists (проверка существования)

**Алгоритм:**

```
1. Захватить Read Lock
2. Найти ключ в индексе
3. Проверить флаг удаления
4. Освободить Lock
5. Вернуть true/false
```

**Код:**

```go
func (s *LocalStore) Exists(key string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()

    entry, ok := s.index[key]
    if !ok {
        return false
    }
    return !entry.Deleted
}
```

---

## HTTP сервер и обработчики

### Структура KVServer

```go
type KVServer struct {
    store storage.Store
}

func NewKVServer(store storage.Store) *KVServer {
    return &KVServer{store: store}
}

func (s *KVServer) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/api/v1/set", s.handleSet)
    mux.HandleFunc("/api/v1/get", s.handleGet)
    mux.HandleFunc("/api/v1/delete", s.handleDelete)
    mux.HandleFunc("/api/v1/list", s.handleList)
    mux.HandleFunc("/api/v1/exists", s.handleExists)
    mux.HandleFunc("/health", s.handleHealth)
}
```

### Обработчик Set

**Поток данных:**

```
HTTP Request
     │
     ▼
┌─────────────────────────────────────┐
│ 1. Проверка метода (POST)           │
│ 2. Получение параметров key, size   │
│ 3. Парсинг size (string → int64)    │
│ 4. Вызов store.Set()                │
│ 5. Формирование JSON ответа         │
└─────────────────────────────────────┘
     │
     ▼
HTTP Response (JSON)
```

**Код:**

```go
func (s *KVServer) handleSet(w http.ResponseWriter, r *http.Request) {
    // 1. Проверка метода
    if r.Method != http.MethodPost {
        writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
        return
    }

    // 2. Получение параметров
    key := r.URL.Query().Get("key")
    if key == "" {
        writeError(w, http.StatusBadRequest, "параметр key обязателен")
        return
    }

    sizeStr := r.URL.Query().Get("size")
    if sizeStr == "" {
        writeError(w, http.StatusBadRequest, "параметр size обязателен")
        return
    }

    // 3. Парсинг размера
    size, err := strconv.ParseInt(sizeStr, 10, 64)
    if err != nil {
        writeError(w, http.StatusBadRequest, "неверный формат size")
        return
    }

    // 4. Запись в хранилище
    if err := s.store.Set(key, r.Body, size); err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // 5. Ответ
    writeJSON(w, http.StatusOK, Response{
        Success: true,
        Data: map[string]string{"key": key},
    })
}
```

### Обработчик Get

**Поток данных:**

```
HTTP Request (GET /api/v1/get?key=mykey)
     │
     ▼
┌─────────────────────────────────────┐
│ 1. Проверка метода (GET)            │
│ 2. Получение параметра key          │
│ 3. Вызов store.Get()                │
│ 4. Установка заголовков             │
│ 5. Потоковая передача данных        │
└─────────────────────────────────────┘
     │
     ▼
HTTP Response (Binary)
```

**Код:**

```go
func (s *KVServer) handleGet(w http.ResponseWriter, r *http.Request) {
    // 1. Проверка метода
    if r.Method != http.MethodGet {
        writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
        return
    }

    // 2. Получение ключа
    key := r.URL.Query().Get("key")
    if key == "" {
        writeError(w, http.StatusBadRequest, "параметр key обязателен")
        return
    }

    // 3. Чтение из хранилища
    reader, size, err := s.store.Get(key)
    if err != nil {
        writeError(w, http.StatusNotFound, err.Error())
        return
    }

    // 4. Установка заголовков
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
    w.Header().Set("X-KV-Size", strconv.FormatInt(size, 10))

    // 5. Потоковая передача
    io.Copy(w, reader)
}
```

### Обработчик Delete

**Код:**

```go
func (s *KVServer) handleDelete(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
        return
    }

    key := r.URL.Query().Get("key")
    if key == "" {
        writeError(w, http.StatusBadRequest, "параметр key обязателен")
        return
    }

    if err := s.store.Delete(key); err != nil {
        writeError(w, http.StatusNotFound, err.Error())
        return
    }

    writeJSON(w, http.StatusOK, Response{
        Success: true,
        Data: map[string]string{"key": key},
    })
}
```

### Обработчик List

**Код:**

```go
func (s *KVServer) handleList(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
        return
    }

    keys := s.store.List()
    
    // Обработка nil → пустой массив JSON
    if keys == nil {
        keys = []string{}
    }
    
    writeJSON(w, http.StatusOK, Response{
        Success: true,
        Data: keys,
    })
}
```

---

## Клиентская библиотека

### Архитектура клиента

```
┌─────────────────────────────────────────────────────────────┐
│                      Client                                 │
├─────────────────────────────────────────────────────────────┤
│  baseURL: string                                            │
│  httpClient: *http.Client                                   │
├─────────────────────────────────────────────────────────────┤
│  Set() ─────────────────────────────────────────────┐       │
│  Get() ─────────────────────────────────────────────┤       │
│  Delete() ──────────────────────────────────────────┤       │
│  List() ────────────────────────────────────────────┤       │
│  Exists() ──────────────────────────────────────────┤       │
│  Health() ──────────────────────────────────────────┤       │
└──────────────────────────────────────────────────────┼───────┘
                                                       │
                    HTTP Request                       │
               ┌───────────────────────────┐           │
               │  POST /api/v1/set         │◄──────────┘
               │  GET  /api/v1/get         │
               │  DELETE /api/v1/delete    │
               │  GET  /api/v1/list        │
               │  GET  /api/v1/exists      │
               │  GET  /health             │
               └───────────────────────────┘
```

### Конфигурация клиента

```go
type Config struct {
    BaseURL         string        // Адрес сервера
    Timeout         time.Duration // Таймаут запросов
    MaxIdleConns    int           // Макс. idle соединений
    MaxConnsPerHost int           // Макс. соединений на хост
}

func DefaultConfig() Config {
    return Config{
        BaseURL:         "http://localhost:8080",
        Timeout:         30 * time.Second,
        MaxIdleConns:    100,
        MaxConnsPerHost: 10,
    }
}
```

### Реализация Set

```go
func (c *Client) Set(key string, data io.Reader, size int64) error {
    // Формирование URL с параметрами
    url := fmt.Sprintf("%s/api/v1/set?key=%s&size=%d", c.baseURL, key, size)

    // POST запрос с телом
    resp, err := c.httpClient.Post(url, "application/octet-stream", data)
    if err != nil {
        return fmt.Errorf("ошибка отправки данных: %w", err)
    }
    defer resp.Body.Close()

    // Декодирование JSON ответа
    var response Response
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return fmt.Errorf("ошибка декодирования ответа: %w", err)
    }

    // Проверка успеха
    if !response.Success {
        return fmt.Errorf("ошибка сервера: %s", response.Error)
    }

    return nil
}
```

### Реализация Get

```go
func (c *Client) Get(key string) (io.ReadCloser, int64, error) {
    url := fmt.Sprintf("%s/api/v1/get?key=%s", c.baseURL, key)

    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, 0, fmt.Errorf("ошибка получения данных: %w", err)
    }

    // Проверка статуса
    if resp.StatusCode == http.StatusNotFound {
        resp.Body.Close()
        return nil, 0, fmt.Errorf("ключ не найден: %s", key)
    }

    if resp.StatusCode != http.StatusOK {
        defer resp.Body.Close()
        var response Response
        json.NewDecoder(resp.Body).Decode(&response)
        return nil, 0, fmt.Errorf("ошибка сервера: %s", response.Error)
    }

    // Получение размера из заголовка
    sizeStr := resp.Header.Get("X-KV-Size")
    var size int64 = 0
    if sizeStr != "" {
        size, _ = strconv.ParseInt(sizeStr, 10, 64)
    }

    // Возвращаем тело ответа как Reader
    return resp.Body, size, nil
}
```

### Пул соединений

Клиент использует настраиваемый Transport для эффективного управления соединениями:

```go
transport := &http.Transport{
    MaxIdleConns:    config.MaxIdleConns,     // 100 по умолчанию
    MaxConnsPerHost: config.MaxConnsPerHost,  // 10 по умолчанию
    IdleConnTimeout: 90 * time.Second,
}

httpClient := &http.Client{
    Timeout:   config.Timeout,  // 30 секунд по умолчанию
    Transport: transport,
}
```

---

## Потокобезопасность

### RWMutex

Хранилище использует `sync.RWMutex` для обеспечения потокобезопасности:

```go
type LocalStore struct {
    mu    sync.RWMutex  // Read-Write мьютекс
    file  *os.File
    index map[string]*Entry
}
```

### Блокировки для операций

| Операция | Тип блокировки | Причина |
|----------|----------------|---------|
| Set      | Write Lock     | Запись в файл и индекс |
| Get      | Read Lock      | Только чтение |
| Delete   | Write Lock     | Модификация индекса |
| List     | Read Lock      | Только чтение |
| Exists   | Read Lock      | Только чтение |

### Пример использования

```go
// Операция чтения (множественные читатели)
func (s *LocalStore) Get(key string) (...) {
    s.mu.RLock()   // Блокировка для чтения
    defer s.mu.RUnlock()
    // ... чтение ...
}

// Операция записи (эксклюзивный писатель)
func (s *LocalStore) Set(key string, ...) error {
    s.mu.Lock()    // Эксклюзивная блокировка
    defer s.mu.Unlock()
    // ... запись ...
}
```

### Сценарий конкурентного доступа

```
Время    Поток 1 (Get)     Поток 2 (Get)     Поток 3 (Set)
────────────────────────────────────────────────────────────
t0       RLock()
t1       read index
t2                       RLock()
t3                       read index
t4       RUnlock()
t5                                           BLOCKED (ждёт)
t6                       RUnlock()
t7                                           Lock()
t8                                           write file
t9                                           update index
t10                                          Unlock()
```

---

## Индексация и загрузка

### Структура индекса

```go
index map[string]*Entry

// Entry содержит:
type Entry struct {
    Key      string  // Ключ
    Size     int64   // Размер значения
    Deleted  bool    // Флаг удаления
    Position int64   // Смещение в файле
}
```

### Загрузка индекса при старте

**Алгоритм LoadIndex():**

```
1. Захватить Write Lock
2. Переместиться в начало файла (SeekStart)
3. Очистить текущий индекс
4. Цикл чтения записей:
   a. Прочитать заголовок (17 байт)
   b. Если EOF — выход из цикла
   c. Извлечь длину ключа и значения
   d. Проверить флаг удаления
   e. Если не удалён — прочитать ключ
   f. Добавить в индекс: index[key] = Entry{Position: currentPos, ...}
   g. Пропустить значение
5. Освободить Lock
```

**Код:**

```go
func (s *LocalStore) LoadIndex() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Переход в начало
    if _, err := s.file.Seek(0, io.SeekStart); err != nil {
        return err
    }

    // Очистка индекса
    s.index = make(map[string]*Entry)

    for {
        // Текущая позиция
        pos, err := s.file.Seek(0, io.SeekCurrent)
        if err != nil {
            return err
        }

        // Чтение заголовка
        header := make([]byte, HEADER_SIZE)
        n, err := s.file.Read(header)
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        // Парсинг заголовка
        keylen := binary.LittleEndian.Uint32(header[0:4])
        vallen := binary.LittleEndian.Uint64(header[4:12])
        deleted := header[12] != 0

        // Чтение ключа
        keybytes := make([]byte, keylen)
        n, err = s.file.Read(keybytes)
        if err != nil {
            return err
        }
        key := string(keybytes)

        // Добавление в индекс (если не удалён)
        if !deleted {
            s.index[key] = &Entry{
                Key:      key,
                Size:     int64(vallen),
                Deleted:  false,
                Position: pos,
            }
        }

        // Пропуск значения
        if _, err := s.file.Seek(int64(vallen), io.SeekCurrent); err != nil {
            return err
        }
    }

    return nil
}
```

### Диаграмма загрузки индекса

```
Файл data.db:
┌─────────────────────────────────────────────────────────────┐
│ pos=0                                                       │
│ ┌─────────────────────────────────────────────────────┐     │
│ │ Запись 1: "config" → "value1"                       │     │
│ │ [Header][config][value1]                            │─────┼──> index["config"] = Entry{Position: 0, ...}
│ └─────────────────────────────────────────────────────┘     │
│                                                             │
│ pos=35                                                      │
│ ┌─────────────────────────────────────────────────────┐     │
│ │ Запись 2: "cache" → "data"                          │     │
│ │ [Header][cache][data]                               │─────┼──> index["cache"] = Entry{Position: 35, ...}
│ └─────────────────────────────────────────────────────┘     │
│                                                             │
│ pos=70                                                      │
│ ┌─────────────────────────────────────────────────────┐     │
│ │ Запись 3: "old" → "deleted" (флаг=1)                │     │
│ │ [Header][old][deleted]                              │     │  НЕ добавляется в индекс
│ └─────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

---

## Удаление данных

### Стратегия удаления

Вместо физического удаления записи используется **флаг удаления**:

```
До удаления:
┌─────────────────────────────────────┐
│ [Header:flag=0][Key][Value]         │
└─────────────────────────────────────┘

После удаления:
┌─────────────────────────────────────┐
│ [Header:flag=1][Key][Value]         │
└─────────────────────────────────────┘
                    ▲
                    │
              Изменён 1 байт
```

### Алгоритм Delete

```
1. Захватить Write Lock
2. Найти запись в индексе
3. Проверить существование
4. Установить флаг Deleted = true в индексе
5. Записать 1 байт (флаг=1) в файл
6. Удалить из индекса
7. Освободить Lock
```

**Код:**

```go
func (s *LocalStore) Delete(key string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    entry, ok := s.index[key]
    if !ok {
        return fmt.Errorf("ключ не найден: %s", key)
    }

    if entry.Deleted {
        return nil // Уже удалён
    }

    // Установить флаг в индексе
    entry.Deleted = true

    // Записать флаг в файл (байт 12 заголовка)
    header := make([]byte, 1)
    header[0] = 1
    _, err := s.file.WriteAt(header, entry.Position+12)
    if err != nil {
        return err
    }

    // Удалить из индекса
    delete(s.index, key)

    return nil
}
```

### Преимущества такого подхода

| Преимущество | Описание |
|--------------|----------|
| ⚡ Скорость | Изменяется 1 байт, не нужно переписывать файл |
| 🔒 Безопасность | Атомарная операция |
| 📊 Простота | Нет необходимости в компактизации |

### Недостатки

| Недостаток | Описание |
|------------|----------|
| 📦 Размер файла | Файл растёт, удалённые данные остаются |
| 🔄 Повторное использование | Ключ нельзя перезаписать эффективно |

### Решение: Компактизация (будущая функция)

Для уменьшения размера файла можно реализовать компактизацию:

```
До компактизации:
┌─────────────────────────────────────────────────────────────┐
│ [Active][Deleted][Active][Deleted][Active]                  │
│     50KB     100KB    30KB     80KB    20KB                 │
│                                         Total: 280KB        │
│                                         Used:  100KB        │
└─────────────────────────────────────────────────────────────┘

После компактизации:
┌─────────────────────────────────────┐
│ [Active][Active][Active]            │
│     50KB    30KB    20KB            │
│                 Total: 100KB        │
└─────────────────────────────────────┘
```

---

## Производительность и оптимизация

### Характеристики производительности

| Операция | Сложность | Описание |
|----------|-----------|----------|
| Set      | O(1)      | Запись в конец файла |
| Get      | O(1)      | Поиск в map + ReadAt |
| Delete   | O(1)      | Изменение 1 байта |
| List     | O(n)      | Копирование ключей |
| Exists   | O(1)      | Поиск в map |
| LoadIndex| O(n)      | Чтение всего файла |

### Оптимизации

#### 1. Индекс в памяти

```
┌─────────────────────────────────────┐
│ Индекс (map[string]*Entry)          │
│ ┌─────────────────────────────────┐ │
│ │ "key1" → {Position: 0, ...}     │ │  O(1) поиск
│ │ "key2" → {Position: 35, ...}    │ │
│ │ "key3" → {Position: 70, ...}    │ │
│ └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

#### 2. SectionReader для больших файлов

```go
// Вместо загрузки всего значения в память
reader := io.NewSectionReader(s.file, valpos, size)
io.Copy(w, reader)  // Потоковая передача
```

#### 3. RWMutex для конкурентного чтения

```go
// Множественные читатели могут работать параллельно
s.mu.RLock()  // Не блокирует других читателей
// ... чтение ...
s.mu.RUnlock()
```

#### 4. Sync после записи

```go
// Гарантия записи на диск
s.file.Sync()
```

### Бенчмарки (ориентировочные)

```
BenchmarkSet-8       10000    120000 ns/op    (120 мкс на запись)
BenchmarkGet-8      100000     15000 ns/op    (15 мкс на чтение)
BenchmarkDelete-8  1000000      2000 ns/op    (2 мкс на удаление)
BenchmarkList-8     100000     10000 ns/op    (10 мкс для 1000 ключей)
```

### Рекомендации по оптимизации

#### Для больших файлов

```go
// Используйте потоковую передачу
reader, size, _ := cli.Get("large_file")
defer reader.Close()

outFile, _ := os.Create("output")
defer outFile.Close()

io.Copy(outFile, reader)  // Не загружает в память
```

#### Для множественных запросов

```go
// Переиспользуйте клиента
cli, _ := client.NewClient(config)
defer cli.Close()  // Закройте когда больше не нужен

// Не создавайте нового клиента на каждый запрос!
```

#### Для высокой нагрузки

```go
// Увеличьте пул соединений
cli, _ := client.NewClient(client.Config{
    BaseURL:         "http://localhost:8080",
    MaxIdleConns:    1000,
    MaxConnsPerHost: 100,
})
```

---

## Расширение функциональности

### Добавление новых команд

#### 1. Новый endpoint на сервере

```go
// internal/server/http.go
func (s *KVServer) RegisterRoutes(mux *http.ServeMux) {
    // ... существующие ...
    mux.HandleFunc("/api/v1/count", s.handleCount)
}

func (s *KVServer) handleCount(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
        return
    }

    count := len(s.store.List())
    writeJSON(w, http.StatusOK, Response{
        Success: true,
        Data: map[string]int{"count": count},
    })
}
```

#### 2. Метод в интерфейсе Store

```go
// internal/storage/storage.go
type Store interface {
    // ... существующие ...
    Count() int
}

func (s *LocalStore) Count() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.index)
}
```

#### 3. Метод в клиенте

```go
// pkg/client/client.go
func (c *Client) Count() (int, error) {
    url := fmt.Sprintf("%s/api/v1/count", c.baseURL)
    
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var response Response
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return 0, err
    }

    data, ok := response.Data.(map[string]interface{})
    if !ok {
        return 0, fmt.Errorf("неверный формат ответа")
    }

    count := int(data["count"].(float64))
    return count, nil
}
```

#### 4. Команда в CLI

```go
// cmd/cli/main.go
case "count":
    count, err := cli.Count()
    if err != nil {
        fmt.Fprintf(os.Stderr, "❌ Ошибка: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("📊 Количество ключей: %d\n", count)
```

### Идеи для расширения

| Функция | Описание | Сложность |
|---------|----------|-----------|
| TTL | Автоматическое удаление по времени | Средняя |
| Компрессия | Сжатие данных (gzip) | Низкая |
| Шифрование | AES шифрование данных | Средняя |
| Репликация | Синхронизация между серверами | Высокая |
| Транзакции | Групповые операции | Высокая |
| Pub/Sub | Уведомления об изменениях | Высокая |
| Метрики | Prometheus exporter | Низкая |

### Пример: Добавление TTL

```go
type Entry struct {
    Key       string
    Size      int64
    Deleted   bool
    Position  int64
    CreatedAt time.Time
    TTL       time.Duration
}

func (s *LocalStore) SetWithTTL(key string, r io.Reader, size int64, ttl time.Duration) error {
    // ... запись ...
    s.index[key] = &Entry{
        // ...
        CreatedAt: time.Now(),
        TTL:       ttl,
    }
}

func (s *LocalStore) Get(key string) (...) {
    entry, ok := s.index[key]
    if !ok {
        return nil, 0, ErrNotFound
    }
    
    // Проверка TTL
    if time.Since(entry.CreatedAt) > entry.TTL {
        s.Delete(key)  // Автоудаление
        return nil, 0, ErrExpired
    }
    
    // ...
}
```

---

## Безопасность

### Текущие ограничения

| Угроза | Статус | Рекомендация |
|--------|--------|--------------|
| Нет аутентификации | ⚠️ | Использовать за прокси |
| Нет авторизации | ⚠️ | Использовать за прокси |
| Нет шифрования | ⚠️ | Использовать HTTPS |
| Нет rate limiting | ⚠️ | Использовать nginx |

### Рекомендации для production

#### 1. HTTPS через nginx

```nginx
server {
    listen 443 ssl;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Базовая аутентификация
    auth_basic "KV Store";
    auth_basic_user_file /etc/nginx/.htpasswd;
    
    location / {
        proxy_pass http://localhost:8080;
    }
}
```

#### 2. Rate limiting

```nginx
http {
    limit_req_zone $binary_remote_addr zone=kvstore:10m rate=10r/s;
    
    server {
        location / {
            limit_req zone=kvstore burst=20;
            proxy_pass http://localhost:8080;
        }
    }
}
```

#### 3. Сетевая изоляция

```bash
# Запуск только на внутреннем интерфейсе
./server -addr 10.0.0.1:8080

# Firewall правила
ufw allow from 10.0.0.0/24 to any port 8080
```

---

## Заключение

KV Store — это простое, но эффективное ключ-значение хранилище с:

- ✅ Простым HTTP API
- ✅ Потокобезопасной реализацией
- ✅ Эффективным бинарным форматом
- ✅ Удобной клиентской библиотекой
- ✅ Поддержкой Docker

Архитектура позволяет легко расширять функциональность и адаптировать хранилище под конкретные задачи.
