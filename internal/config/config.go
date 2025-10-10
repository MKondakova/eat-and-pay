package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"eats-backend/internal/models"
)

var (
	errDecodePem            = errors.New("can't decode pem")
	errKeyIsNotRsaPublicKey = errors.New("key is not RSA public key")
)

type Config struct {
	ListenPort string

	PublicKey  *rsa.PublicKey  `env:"PUBLIC_KEY,notEmpty"`
	PrivateKey *rsa.PrivateKey `env:"PRIVATE_KEY,notEmpty"`

	RevokedTokens []string

	InitialProductsData      []*models.Product
	InitialCategories        map[string]models.Category
	InitialProductCategories map[string][]string

	ServerOpts        ServerOpts
	FeedbacksPath     string
	CreatedTokensPath string
}

func GetConfig(logger *zap.SugaredLogger) (*Config, error) {
	cfg := &Config{
		ListenPort: ":8080",
		ServerOpts: ServerOpts{
			ReadTimeout:          60,
			WriteTimeout:         60,
			IdleTimeout:          60,
			MaxRequestBodySizeMb: 1,
		},
		CreatedTokensPath: "data/created_tokens.csv",
	}

	// Загружаем товары и преобразуем в указатели
	products, err := getInitData[models.Product]("data/products.json", logger)
	if err != nil {
		logger.Warnf("Can't load products from file: %v", err)
		cfg.InitialProductsData = []*models.Product{}
	} else {
		cfg.InitialProductsData = make([]*models.Product, len(products))
		for i := range products {
			cfg.InitialProductsData[i] = &products[i]
		}
	}

	// Загружаем категории и преобразуем в map
	categories, err := getInitData[models.Category]("data/categories.json", logger)
	if err != nil {
		logger.Warnf("Can't load categories from file: %v", err)
		cfg.InitialCategories = map[string]models.Category{}
	} else {
		cfg.InitialCategories = make(map[string]models.Category)
		for _, category := range categories {
			cfg.InitialCategories[category.ID] = category
		}
	}

	// Загружаем связки товаров и категорий
	productCategories, err := getProductCategories("data/product_categories.json", logger)
	if err != nil {
		logger.Warnf("Can't load product categories from file: %v", err)
		cfg.InitialProductCategories = map[string][]string{}
	} else {
		cfg.InitialProductCategories = productCategories
	}

	// Загружаем заблокированные токены
	bannedTokens, err := getInitData[string]("data/blocked_tokens.json", logger)
	if err != nil {
		logger.Warnf("Can't load banned tokens from file: %v", err)
		cfg.RevokedTokens = []string{}
	} else {
		cfg.RevokedTokens = bannedTokens
	}

	opts := env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(rsa.PublicKey{}):  ParsePubKey,
			reflect.TypeOf(rsa.PrivateKey{}): ParsePrivateKey,
		},
	}

	err = env.ParseWithOptions(cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("env.ParseWithOptions: %w", err)
	}

	return cfg, nil
}

type ServerOpts struct {
	ReadTimeout          int `json:"read_timeout"`
	WriteTimeout         int `json:"write_timeout"`
	IdleTimeout          int `json:"idle_timeout"`
	MaxRequestBodySizeMb int `json:"max_request_body_size_mb"`
}

// ParsePubKey public keys loader for github.com/caarlos0/env/v11 lib.
func ParsePubKey(value string) (any, error) {
	publicKey, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString: %w", err)
	}

	pubKey, err := ParseRSAPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("keys.ParseRSAPublicKey: %w", err)
	}

	return *pubKey, nil
}

// ParsePrivateKey pkcs1 private keys loader for github.com/caarlos0/env/v11 lib.
func ParsePrivateKey(value string) (any, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString: %w", err)
	}

	block, _ := pem.Decode(decoded)
	if block == nil {
		return nil, errDecodePem
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(decoded)
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseRSAPrivateKeyFromPEM: %w", err)
	}

	return *key, nil
}

func ParseRSAPublicKey(content []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, errDecodePem
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("can't parse PKIX public key: %w", err)
	}

	public, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errKeyIsNotRsaPublicKey
	}

	return public, nil
}

type loadable interface {
	string | models.Product | models.Category
}

func getInitData[T loadable](filePath string, logger *zap.SugaredLogger) ([]T, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Errorf("Error while closing data file: %v", err)
		}
	}(file)

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data []T
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}

func getProductCategories(filePath string, logger *zap.SugaredLogger) (map[string][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Errorf("Error while closing data file: %v", err)
		}
	}(file)

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data map[string][]string
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}
