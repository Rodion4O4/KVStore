package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/admin/kvstore/internal/server"
	"github.com/admin/kvstore/internal/storage"
)

func main() {
	addr := flag.String("addr", ":8080", "адрес для прослушивания")
	dataDir := flag.String("data", "./data", "директория для данных")
	flag.Parse()

	store, err := storage.NewLocalStore(*dataDir)
	if err != nil {
		log.Fatalf("ошибка инициализации хранилища: %v", err)
	}
	defer store.Close()

	if err := store.LoadIndex(); err != nil {
		log.Printf("предупреждение при загрузке индекса: %v", err)
	}

	kvServer := server.NewKVServer(store)
	mux := http.NewServeMux()
	kvServer.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		fmt.Printf("🚀 KV Store сервер запущен на %s\n", *addr)
		fmt.Printf("📁 Директория данных: %s\n", *dataDir)
		fmt.Println()
		fmt.Println("API endpoints:")
		fmt.Println("  POST   /api/v1/set?key=<key>&size=<size>  - сохранить значение")
		fmt.Println("  GET    /api/v1/get?key=<key>              - получить значение")
		fmt.Println("  DELETE /api/v1/delete?key=<key>           - удалить ключ")
		fmt.Println("  GET    /api/v1/list                       - список всех ключей")
		fmt.Println("  GET    /api/v1/exists?key=<key>           - проверка существования")
		fmt.Println("  GET    /health                            - проверка здоровья")
		fmt.Println()

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ошибка сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Остановка сервера...")
}
