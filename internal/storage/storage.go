package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Storage struct {
	logger *zap.SugaredLogger
	dir    string
}

func NewStorage(logger *zap.SugaredLogger, dir string) *Storage {
	return &Storage{
		logger: logger,
		dir:    dir,
	}
}

func (s *Storage) SaveFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB max

	reader, err := r.MultipartReader()
	if err != nil {
		s.logger.Errorf("invalid multipart request: %v", err)
		http.Error(w, "invalid multipart request", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(s.dir, os.ModePerm); err != nil {
		s.logger.Errorf("can't create upload dir: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tempName := uuid.NewString()
	var savedFile string

	for {
		name, err := s.loadPart(reader, tempName)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.logger.Errorf("upload failed: %v", err)
			http.Error(w, "upload failed", http.StatusInternalServerError)
			return
		}
		if name != "" {
			savedFile = name
		}
	}

	if savedFile == "" {
		http.Error(w, "no file part found", http.StatusBadRequest)
		return
	}

	s.logger.Infof("uploaded file %s to %s successfully", savedFile, s.dir)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"file": savedFile})
	if err != nil {
		s.logger.Errorf("can't encode response: %v", err)
	}
}

func (s *Storage) loadPart(reader *multipart.Reader, tempName string) (string, error) {
	part, err := reader.NextPart()
	if part != nil {
		fmt.Println(err, part.FormName(), part.FileName())
	} else {
		fmt.Println(err, errors.Is(err, io.EOF))
	}
	if errors.Is(err, io.EOF) {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("can't read next part: %w", err)
	}

	if part.FormName() != "file" {
		return "", nil
	}

	ext := filepath.Ext(part.FileName())
	if ext == "" {
		ext = ".bin"
	}
	fullPath := filepath.Join(s.dir, tempName+ext)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("can't create file: %w", err)
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			s.logger.Warnf("can't close file: %v", err)
		}
	}(dst)

	if _, err := io.Copy(dst, part); err != nil {
		return "", fmt.Errorf("can't copy data: %w", err)
	}

	return tempName + ext, nil
}
