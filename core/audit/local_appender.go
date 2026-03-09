package audit

import (
	"encoding/json"
	"os"
	"sync"
)

type LocalAppender struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
}

func NewLocalAppender(path string) *LocalAppender {
	return &LocalAppender{filePath: path}
}

func (a *LocalAppender) Open() error {
	f, err := os.OpenFile(a.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	a.file = f
	return nil
}

func (a *LocalAppender) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

func (a *LocalAppender) AppendBlock(b Block) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = a.file.Write(data)
	return err
}
