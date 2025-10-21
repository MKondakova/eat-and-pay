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

	// User data
	InitialUserProfiles map[string]*models.UserProfile
	InitialCartItems    map[string]map[string]*models.CartItem
	InitialFavourites   map[string][]string
	InitialOrders       map[string][]*models.Order
	InitialWalletData   models.WalletData

	ServerOpts        ServerOpts
	FeedbacksPath     string
	CreatedTokensPath string
	Host              string
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
		Host:              "http://eats-pages.ddns.net/uploads/",
	}

	// Загружаем товары и преобразуем в указатели
	products, err := getInitData[models.Product]("data/products.json", logger)
	if err != nil {
		logger.Warnf("Can't load products from file: %v", err)
		cfg.InitialProductsData = []*models.Product{}
	} else {
		cfg.InitialProductsData = make([]*models.Product, len(products))
		for i := range products {
			products[i].Image = cfg.Host + products[i].Image
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

	// Загружаем профили пользователей
	userProfiles, err := getUserProfiles("data/user_profiles.json", logger)
	if err != nil {
		logger.Warnf("Can't load user profiles from file: %v", err)
		cfg.InitialUserProfiles = make(map[string]*models.UserProfile)
	} else {
		cfg.InitialUserProfiles = userProfiles
	}

	// Загружаем корзины пользователей
	cartItems, err := getCartItems("data/cart_items.json", logger)
	if err != nil {
		logger.Warnf("Can't load cart items from file: %v", err)
		cfg.InitialCartItems = make(map[string]map[string]*models.CartItem)
	} else {
		cfg.InitialCartItems = cartItems
	}

	// Загружаем избранное пользователей
	favourites, err := getFavourites("data/user_favourites.json", logger)
	if err != nil {
		logger.Warnf("Can't load favourites from file: %v", err)
		cfg.InitialFavourites = make(map[string][]string)
	} else {
		cfg.InitialFavourites = favourites
	}

	// Загружаем заказы пользователей
	orders, err := getOrders("data/orders.json", logger)
	if err != nil {
		logger.Warnf("Can't load orders from file: %v", err)
		cfg.InitialOrders = make(map[string][]*models.Order)
	} else {
		cfg.InitialOrders = orders
	}

	// Загружаем данные кошелька
	walletData, err := getWalletData("data/wallet_data.json", logger)
	if err != nil {
		logger.Warnf("Can't load wallet data from file: %v", err)
		// Инициализируем пустые данные кошелька
		cfg.InitialWalletData = models.WalletData{
			Accounts:     make(map[string]map[string]*models.Account),
			Transactions: make(map[string][]models.Transaction),
			DailyTopups:  make(map[string]map[string]int),
			UserPhones:   make(map[string]string),
		}
	} else {
		cfg.InitialWalletData = walletData
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

// loadJSONFile - обобщенная функция для загрузки JSON из файла
func loadJSONFile[T any](filePath string, logger *zap.SugaredLogger) (T, error) {
	var result T

	file, err := os.Open(filePath)
	if err != nil {
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Errorf("Error while closing file %s: %v", filePath, err)
		}
	}()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return result, fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(bytes, &result); err != nil {
		return result, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

type loadable interface {
	string | models.Product | models.Category
}

func getInitData[T loadable](filePath string, logger *zap.SugaredLogger) ([]T, error) {
	return loadJSONFile[[]T](filePath, logger)
}

func getProductCategories(filePath string, logger *zap.SugaredLogger) (map[string][]string, error) {
	return loadJSONFile[map[string][]string](filePath, logger)
}

// getUserProfiles загружает профили пользователей из файла
func getUserProfiles(filePath string, logger *zap.SugaredLogger) (map[string]*models.UserProfile, error) {
	return loadJSONFile[map[string]*models.UserProfile](filePath, logger)
}

// getCartItems загружает корзины пользователей из файла
func getCartItems(filePath string, logger *zap.SugaredLogger) (map[string]map[string]*models.CartItem, error) {
	return loadJSONFile[map[string]map[string]*models.CartItem](filePath, logger)
}

// getFavourites загружает избранное пользователей из файла
func getFavourites(filePath string, logger *zap.SugaredLogger) (map[string][]string, error) {
	return loadJSONFile[map[string][]string](filePath, logger)
}

// getOrders загружает заказы пользователей из файла
func getOrders(filePath string, logger *zap.SugaredLogger) (map[string][]*models.Order, error) {
	return loadJSONFile[map[string][]*models.Order](filePath, logger)
}

// getWalletData загружает данные кошелька из файла
func getWalletData(filePath string, logger *zap.SugaredLogger) (models.WalletData, error) {
	return loadJSONFile[models.WalletData](filePath, logger)
}
