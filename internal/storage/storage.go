package storage

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"eats-backend/internal/models"
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

func (s *Storage) SaveFile(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB max

	reader, err := r.MultipartReader()
	if err != nil {
		return "", fmt.Errorf("%w: invalid multipart request: %w", models.ErrBadRequest, err)
	}

	if err := os.MkdirAll(s.dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("%w: can't create upload dir: %w", models.ErrInternalServer, err)
	}

	tempName := uuid.NewString()
	var savedFile string

	for {
		name, err := s.loadPart(reader, tempName)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%w: upload failed: %w", models.ErrInternalServer, err)
		}
		if name != "" {
			savedFile = name
		}
	}

	if savedFile == "" {
		return "", fmt.Errorf("%w: no file part found: %w", models.ErrBadRequest, err)
	}

	s.logger.Infof("uploaded file %s to %s successfully", savedFile, s.dir)

	return savedFile, nil
}

func (s *Storage) loadPart(reader *multipart.Reader, tempName string) (string, error) {
	part, err := reader.NextPart()
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
	if ext != ".jxl" {
		return "", fmt.Errorf("wrong extension, should be .jxl: %w", models.ErrBadRequest)
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
