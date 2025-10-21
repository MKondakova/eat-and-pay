package storage

import (
	"bytes"
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

var (
	// JXL magic bytes для "naked" codestream формата
	jxlNakedSignature = []byte{0xFF, 0x0A}

	// JXL magic bytes для container (ISO BMFF) формата
	jxlContainerSignature = []byte{0x00, 0x00, 0x00, 0x0C, 0x4A, 0x58, 0x4C, 0x20, 0x0D, 0x0A, 0x87, 0x0A}
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

// isValidJXL проверяет, является ли содержимое файла действительным JXL файлом
func isValidJXL(data []byte) bool {
	// Проверяем минимальный размер
	if len(data) < 2 {
		return false
	}

	// Проверяем naked codestream формат (FF 0A)
	if bytes.HasPrefix(data, jxlNakedSignature) {
		return true
	}

	// Проверяем container формат
	if len(data) >= len(jxlContainerSignature) {
		if bytes.HasPrefix(data, jxlContainerSignature) {
			return true
		}
	}

	return false
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
			return "", fmt.Errorf("upload failed: %w", err)
		}
		if name != "" {
			savedFile = name
			break
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

	// Читаем файл в буфер (максимум 5MB уже ограничен в SaveFile)
	fileData, err := io.ReadAll(part)
	if err != nil {
		return "", fmt.Errorf("can't read file data: %w", err)
	}

	// Проверяем, что это действительно JXL файл по содержимому
	if !isValidJXL(fileData) {
		s.logger.Warnf("rejected file %s: not a valid JXL file", part.FileName())
		return "", fmt.Errorf("%w: file is not a valid JXL image", models.ErrBadRequest)
	}

	// Создаем файл для сохранения
	fullPath := filepath.Join(s.dir, tempName+ext)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("can't create file: %w", err)
	}
	defer func() {
		if err := dst.Close(); err != nil {
			s.logger.Warnf("can't close file: %v", err)
		}
	}()

	// Записываем проверенные данные
	if _, err := dst.Write(fileData); err != nil {
		// Удаляем файл при ошибке записи
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("can't write file: %w", err)
	}

	s.logger.Infof("validated and saved JXL file: %s", tempName+ext)
	return tempName + ext, nil
}
