package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

const HEADER_SIZE = 17

type Entry struct {
	Key      string
	Size     int64
	Deleted  bool
	Position int64
}

type Store interface {
	Set(key string, r io.Reader, size int64) error
	Get(key string) (io.Reader, int64, error)
	Delete(key string) error
	List() []string
	Exists(key string) bool
	Close() error
	LoadIndex() error
}

type LocalStore struct {
	mu    sync.RWMutex
	file  *os.File
	index map[string]*Entry
	path  string
}

func NewLocalStore(path string) (Store, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	dbPath := path + "/data.db"
	f, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	store := &LocalStore{
		file:  f,
		index: make(map[string]*Entry),
		path:  dbPath,
	}

	return store, nil
}

func (s *LocalStore) Set(key string, r io.Reader, size int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos, err := s.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	if len(key) > 1<<32-1 {
		return errors.New("ключ слишком длинный")
	}

	header := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint64(header[4:12], uint64(size))
	header[12] = 0

	if _, err := s.file.Write(header); err != nil {
		return err
	}

	if _, err := s.file.WriteString(key); err != nil {
		s.file.Truncate(pos)
		return err
	}

	if _, err := io.CopyN(s.file, r, size); err != nil {
		s.file.Truncate(pos)
		return err
	}

	if err := s.file.Sync(); err != nil {
		return err
	}

	s.index[key] = &Entry{
		Key:      key,
		Size:     size,
		Deleted:  false,
		Position: pos,
	}

	return nil
}

func (s *LocalStore) Get(key string) (io.Reader, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.index[key]
	if !ok {
		return nil, 0, fmt.Errorf("ключ не найден: %s", key)
	}

	if entry.Deleted {
		return nil, 0, fmt.Errorf("запись удалена: %s", key)
	}

	header := make([]byte, HEADER_SIZE)
	if _, err := s.file.ReadAt(header, entry.Position); err != nil {
		return nil, 0, err
	}

	if header[12] != 0 {
		return nil, 0, fmt.Errorf("запись удалена: %s", key)
	}

	keylen := binary.LittleEndian.Uint32(header[0:4])
	vallen := binary.LittleEndian.Uint64(header[4:12])

	keybytes := make([]byte, keylen)
	if _, err := s.file.ReadAt(keybytes, entry.Position+HEADER_SIZE); err != nil {
		return nil, 0, err
	}

	if string(keybytes) != key {
		return nil, 0, errors.New("повреждение данных: ключ не совпадает")
	}

	valpos := entry.Position + HEADER_SIZE + int64(keylen)
	reader := io.NewSectionReader(s.file, valpos, int64(vallen))

	return reader, int64(vallen), nil
}

func (s *LocalStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.index[key]
	if !ok {
		return fmt.Errorf("ключ не найден: %s", key)
	}

	if entry.Deleted {
		return nil
	}

	entry.Deleted = true

	header := make([]byte, 1)
	header[0] = 1

	_, err := s.file.WriteAt(header, entry.Position+12)
	if err != nil {
		return err
	}

	delete(s.index, key)

	return nil
}

func (s *LocalStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.index))
	for key, entry := range s.index {
		if !entry.Deleted {
			keys = append(keys, key)
		}
	}
	return keys
}

func (s *LocalStore) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.index[key]
	if !ok {
		return false
	}
	return !entry.Deleted
}

func (s *LocalStore) LoadIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	s.index = make(map[string]*Entry)

	for {
		pos, err := s.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}

		header := make([]byte, HEADER_SIZE)
		n, err := s.file.Read(header)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n != HEADER_SIZE {
			return fmt.Errorf("повреждённый файл: ожидалось %d байт, получено %d", HEADER_SIZE, n)
		}

		keylen := binary.LittleEndian.Uint32(header[0:4])
		vallen := binary.LittleEndian.Uint64(header[4:12])
		deleted := header[12] != 0

		keybytes := make([]byte, keylen)
		n, err = s.file.Read(keybytes)
		if err != nil || n != int(keylen) {
			return fmt.Errorf("ошибка чтения ключа на позиции %d", pos)
		}

		key := string(keybytes)

		if !deleted {
			s.index[key] = &Entry{
				Key:      key,
				Size:     int64(vallen),
				Deleted:  false,
				Position: pos,
			}
		}

		if _, err := s.file.Seek(int64(vallen), io.SeekCurrent); err != nil {
			return fmt.Errorf("ошибка пропуска значения для ключа %s", key)
		}
	}

	return nil
}

func (s *LocalStore) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
