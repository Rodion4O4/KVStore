package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/admin/kvstore/pkg/client"
)

func main() {
	serverAddr := flag.String("server", "http://localhost:8080", "адрес KV сервера")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "KV Store CLI - клиент для удалённого хранилища\n\n")
		fmt.Fprintf(os.Stderr, "Использование:\n")
		fmt.Fprintf(os.Stderr, "  kvcli set <ключ> <файл>     - сохранить файл\n")
		fmt.Fprintf(os.Stderr, "  kvcli get <ключ> [файл]     - получить файл\n")
		fmt.Fprintf(os.Stderr, "  kvcli delete <ключ>         - удалить ключ\n")
		fmt.Fprintf(os.Stderr, "  kvcli list                  - список ключей\n")
		fmt.Fprintf(os.Stderr, "  kvcli exists <ключ>         - проверка существования\n")
		fmt.Fprintf(os.Stderr, "  kvcli health                - проверка сервера\n\n")
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cli, err := client.NewClientSimple(*serverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка создания клиента: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	cmd := args[0]

	switch cmd {
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "❌ Использование: kvcli set <ключ> <файл>")
			os.Exit(1)
		}
		key := args[1]
		filePath := args[2]

		file, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка открытия файла: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка получения размера: %v\n", err)
			os.Exit(1)
		}

		if err := cli.Set(key, file, stat.Size()); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка сохранения: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Файл '%s' сохранён как '%s' (%d байт)\n", filePath, key, stat.Size())

	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "❌ Использование: kvcli get <ключ> [файл]")
			os.Exit(1)
		}
		key := args[1]
		saveAs := key

		if len(args) > 2 {
			saveAs = args[2]
		}

		reader, size, err := cli.Get(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка чтения: %v\n", err)
			os.Exit(1)
		}
		defer reader.Close()

		outFile, err := os.Create(saveAs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка создания файла: %v\n", err)
			os.Exit(1)
		}
		defer outFile.Close()

		n, err := io.Copy(outFile, reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка записи: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Файл сохранён как '%s' (%d/%d байт)\n", saveAs, n, size)

	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "❌ Использование: kvcli delete <ключ>")
			os.Exit(1)
		}
		key := args[1]

		if err := cli.Delete(key); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка удаления: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Ключ '%s' удалён\n", key)

	case "list":
		keys, err := cli.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка получения списка: %v\n", err)
			os.Exit(1)
		}
		if len(keys) == 0 {
			fmt.Println("📭 Хранилище пусто")
		} else {
			fmt.Printf("📦 Найдено %d ключей:\n", len(keys))
			for i, key := range keys {
				fmt.Printf("  %d. %s\n", i+1, key)
			}
		}

	case "exists":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "❌ Использование: kvcli exists <ключ>")
			os.Exit(1)
		}
		key := args[1]

		exists, err := cli.Exists(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка проверки: %v\n", err)
			os.Exit(1)
		}
		if exists {
			fmt.Printf("✅ Ключ '%s' существует\n", key)
		} else {
			fmt.Printf("❌ Ключ '%s' не найден\n", key)
		}

	case "health":
		healthy, err := cli.Health()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка проверки: %v\n", err)
			os.Exit(1)
		}
		if healthy {
			fmt.Println("✅ Сервер здоров")
		} else {
			fmt.Println("❌ Сервер недоступен")
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "❓ Неизвестная команда: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}
