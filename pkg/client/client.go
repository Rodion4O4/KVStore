package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Config struct {
	BaseURL       string
	Timeout       time.Duration
	MaxIdleConns  int
	MaxConnsPerHost int
}

func DefaultConfig() Config {
	return Config{
		BaseURL:         "http://localhost:8080",
		Timeout:         30 * time.Second,
		MaxIdleConns:    100,
		MaxConnsPerHost: 10,
	}
}

func NewClient(config Config) (*Client, error) {
	if config.BaseURL == "" {
		cfg := DefaultConfig()
		config = cfg
	}

	transport := &http.Transport{
		MaxIdleConns:    config.MaxIdleConns,
		MaxConnsPerHost: config.MaxConnsPerHost,
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	return &Client{
		baseURL:    config.BaseURL,
		httpClient: httpClient,
	}, nil
}

func NewClientSimple(baseURL string) (*Client, error) {
	return NewClient(Config{BaseURL: baseURL})
}

func (c *Client) Set(key string, data io.Reader, size int64) error {
	url := fmt.Sprintf("%s/api/v1/set?key=%s&size=%d", c.baseURL, key, size)

	resp, err := c.httpClient.Post(url, "application/octet-stream", data)
	if err != nil {
		return fmt.Errorf("ошибка отправки данных: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("ошибка сервера: %s", response.Error)
	}

	return nil
}

func (c *Client) SetBytes(key string, data []byte) error {
	return c.Set(key, bytes.NewReader(data), int64(len(data)))
}

func (c *Client) SetString(key string, data string) error {
	return c.SetBytes(key, []byte(data))
}

func (c *Client) Get(key string) (io.ReadCloser, int64, error) {
	url := fmt.Sprintf("%s/api/v1/get?key=%s", c.baseURL, key)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка получения данных: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("ключ не найден: %s", key)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, 0, fmt.Errorf("ошибка декодирования ответа: %w", err)
		}
		resp.Body.Close()
		return nil, 0, fmt.Errorf("ошибка сервера: %s", response.Error)
	}

	sizeStr := resp.Header.Get("X-KV-Size")
	var size int64 = 0
	if sizeStr != "" {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}

	return resp.Body, size, nil
}

func (c *Client) GetBytes(key string) ([]byte, error) {
	reader, _, err := c.Get(key)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения данных: %w", err)
	}

	return data, nil
}

func (c *Client) GetString(key string) (string, error) {
	data, err := c.GetBytes(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) Delete(key string) error {
	url := fmt.Sprintf("%s/api/v1/delete?key=%s", c.baseURL, key)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка удаления: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("ошибка сервера: %s", response.Error)
	}

	return nil
}

func (c *Client) List() ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/list", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("ошибка сервера: %s", response.Error)
	}

	data, ok := response.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("неверный формат ответа")
	}

	keys := make([]string, len(data))
	for i, v := range data {
		if s, ok := v.(string); ok {
			keys[i] = s
		}
	}

	return keys, nil
}

func (c *Client) Exists(key string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/exists?key=%s", c.baseURL, key)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !response.Success {
		return false, fmt.Errorf("ошибка сервера: %s", response.Error)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("неверный формат ответа")
	}

	exists, ok := data["exists"].(bool)
	if !ok {
		return false, fmt.Errorf("неверный формат поля exists")
	}

	return exists, nil
}

func (c *Client) Health() (bool, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки здоровья: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("сервер недоступен: %s", resp.Status)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return response.Success && response.Data != nil, nil
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
