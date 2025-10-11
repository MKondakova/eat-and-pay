package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"eats-backend/internal/models"
	"github.com/google/uuid"
)

type WalletService struct {
	// В реальном приложении это были бы базы данных
	accounts     map[string]map[string]*models.Account // userID -> accountID -> account
	transactions map[string][]models.Transaction       // userID -> transactions
	dailyTopups  map[string]map[string]int             // userID -> date -> total amount
	userPhones   map[string]string                     // userID -> phone
	userData     *UserData                             // для получения номеров телефонов

	mux sync.RWMutex
}

func NewWalletService(userData *UserData) *WalletService {
	ws := &WalletService{
		accounts:     make(map[string]map[string]*models.Account),
		transactions: make(map[string][]models.Transaction),
		dailyTopups:  make(map[string]map[string]int),
		userPhones:   make(map[string]string),
		userData:     userData,
	}

	// Инициализируем тестовые данные
	ws.initTestData()

	return ws
}

// getOrCreateUserPhone получает или создает номер телефона для пользователя
func (ws *WalletService) getOrCreateUserPhone(ctx context.Context) (string, error) {
	userID := models.ClaimsFromContext(ctx).ID

	// Сначала проверяем в кэше userPhones
	if phone, exists := ws.userPhones[userID]; exists {
		return phone, nil
	}

	// Если нет в кэше, получаем из UserData
	profile, err := ws.userData.GetProfile(ctx)
	if err != nil {
		return "", err
	}

	// Сохраняем в кэш
	ws.userPhones[userID] = profile.Phone
	return profile.Phone, nil
}

func (ws *WalletService) initTestData() {
	// Тестовый пользователь с картой
	userID := "4479081e-fd93-499c-bf8b-1ad190b052e6"
	ws.userPhones[userID] = "123123"

	cardID := uuid.New().String()
	ws.accounts[userID] = map[string]*models.Account{
		cardID: {
			ID:      cardID,
			Type:    models.AccountTypeCard,
			Balance: 1500, // 1500 рублей
		},
	}

	// Добавляем несколько тестовых транзакций
	ws.transactions[userID] = []models.Transaction{
		{
			Amount: -250,
			Title:  "Покупка в магазине",
			Time:   time.Now().Add(-2 * time.Hour),
			Icon:   "https://example.com/shop-icon.png",
		},
		{
			Amount: -100,
			Title:  "Кофе",
			Time:   time.Now().Add(-1 * time.Hour),
			Icon:   "https://example.com/coffee-icon.png",
		},
		{
			Amount: 500,
			Title:  "Пополнение счета",
			Time:   time.Now().Add(-30 * time.Minute),
			Icon:   "https://example.com/topup-icon.png",
		},
	}
}

func (ws *WalletService) GetWallet(ctx context.Context) (*models.Wallet, error) {
	userID := models.ClaimsFromContext(ctx).ID

	ws.mux.RLock()
	defer ws.mux.RUnlock()

	userAccounts, exists := ws.accounts[userID]
	if !exists {
		return &models.Wallet{Accounts: []models.Account{}}, nil
	}

	accounts := make([]models.Account, 0, len(userAccounts))
	for _, account := range userAccounts {
		accounts = append(accounts, *account)
	}

	return &models.Wallet{Accounts: accounts}, nil
}

func (ws *WalletService) GetTransactions(ctx context.Context, page, pageSize int) (*models.TransactionsResponse, error) {
	userID := models.ClaimsFromContext(ctx).ID

	ws.mux.RLock()
	defer ws.mux.RUnlock()

	userTransactions, exists := ws.transactions[userID]
	if !exists {
		return &models.TransactionsResponse{
			CurrentPage: page,
			TotalPages:  0,
			Data:        make(models.TransactionsByDate),
		}, nil
	}

	// Сортируем транзакции по времени (новые сначала)
	sort.Slice(userTransactions, func(i, j int) bool {
		return userTransactions[i].Time.After(userTransactions[j].Time)
	})

	// Применяем пагинацию к количеству транзакций
	totalTransactions := len(userTransactions)
	totalPages := int(math.Ceil(float64(totalTransactions) / float64(pageSize)))

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= totalTransactions {
		return &models.TransactionsResponse{
			CurrentPage: page,
			TotalPages:  totalPages,
			Data:        make(models.TransactionsByDate),
		}, nil
	}

	if end > totalTransactions {
		end = totalTransactions
	}

	// Берем только нужную страницу транзакций
	paginatedTransactions := userTransactions[start:end]

	// Перегруппировываем только нужные транзакции
	paginatedByDate := make(models.TransactionsByDate)
	for _, transaction := range paginatedTransactions {
		date := transaction.Time.Format("2006-01-02")
		paginatedByDate[date] = append(paginatedByDate[date], transaction)
	}

	return &models.TransactionsResponse{
		CurrentPage: page,
		TotalPages:  totalPages,
		Data:        paginatedByDate,
	}, nil
}

func (ws *WalletService) TopupAccount(ctx context.Context, req models.TopupRequest) (*models.TopupResponse, error) {
	userID := models.ClaimsFromContext(ctx).ID

	// Проверяем лимит пополнения (1000 рублей в сутки)
	today := time.Now().Format("2006-01-02")

	ws.mux.Lock()
	defer ws.mux.Unlock()

	// Проверяем дневной лимит
	if ws.dailyTopups[userID] == nil {
		ws.dailyTopups[userID] = make(map[string]int)
	}

	if ws.dailyTopups[userID][today]+req.Amount > 1000 {
		return nil, fmt.Errorf("%w: daily topup limit exceeded (1000 rubles per day)", models.ErrBadRequest)
	}

	// Проверяем существование счета
	userAccounts, exists := ws.accounts[userID]
	if !exists {
		return nil, fmt.Errorf("%w: account not found", models.ErrNotFound)
	}

	account, exists := userAccounts[req.AccountID]
	if !exists {
		return nil, fmt.Errorf("%w: account not found", models.ErrNotFound)
	}

	// Обновляем баланс
	account.Balance += req.Amount

	// Обновляем дневной лимит
	ws.dailyTopups[userID][today] += req.Amount

	// Добавляем транзакцию
	transaction := models.Transaction{
		Amount: req.Amount,
		Title:  "Пополнение счета",
		Time:   time.Now(),
	}

	if ws.transactions[userID] == nil {
		ws.transactions[userID] = []models.Transaction{}
	}
	ws.transactions[userID] = append(ws.transactions[userID], transaction)

	return &models.TopupResponse{Balance: account.Balance}, nil
}

func (ws *WalletService) TransferMoney(ctx context.Context, req models.TransferRequest) (*models.TransferResponse, error) {
	fromUserID := models.ClaimsFromContext(ctx).ID

	ws.mux.Lock()
	defer ws.mux.Unlock()

	// Проверяем существование счета отправителя
	fromUserAccounts, exists := ws.accounts[fromUserID]
	if !exists {
		return nil, fmt.Errorf("%w: sender account not found", models.ErrNotFound)
	}

	fromAccount, exists := fromUserAccounts[req.FromAccountID]
	if !exists {
		return nil, fmt.Errorf("%w: sender account not found", models.ErrNotFound)
	}

	// Проверяем достаточность средств
	if fromAccount.Balance < req.Amount {
		return nil, fmt.Errorf("%w: insufficient funds", models.ErrBadRequest)
	}

	// Находим получателя по номеру телефона
	toUserID, found := ws.userData.GetUserIDByPhone(req.ToPhoneNumber)
	if !found {
		return nil, fmt.Errorf("%w: recipient not found", models.ErrNotFound)
	}

	if toUserID == fromUserID {
		return nil, fmt.Errorf("%w: cannot transfer to yourself", models.ErrBadRequest)
	}

	// Проверяем существование счета получателя
	toUserAccounts, exists := ws.accounts[toUserID]
	if !exists {
		return nil, fmt.Errorf("%w: recipient account not found", models.ErrNotFound)
	}

	// Ищем первый счет получателя (в реальном приложении можно было бы выбрать конкретный счет)
	var toAccount *models.Account
	for _, account := range toUserAccounts {
		toAccount = account
		break
	}

	if toAccount == nil {
		return nil, fmt.Errorf("%w: recipient has no accounts", models.ErrNotFound)
	}

	// Выполняем перевод
	fromAccount.Balance -= req.Amount
	toAccount.Balance += req.Amount

	// Добавляем транзакции
	transferTime := time.Now()

	// Транзакция отправителя (отрицательная)
	fromTransaction := models.Transaction{
		Amount: -req.Amount,
		Title:  fmt.Sprintf("Перевод на номер %s", req.ToPhoneNumber),
		Time:   transferTime,
	}

	if ws.transactions[fromUserID] == nil {
		ws.transactions[fromUserID] = []models.Transaction{}
	}
	ws.transactions[fromUserID] = append(ws.transactions[fromUserID], fromTransaction)

	// Транзакция получателя (положительная)
	fromUserPhone, err := ws.getOrCreateUserPhone(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender phone: %w", err)
	}
	toTransaction := models.Transaction{
		Amount: req.Amount,
		Title:  fmt.Sprintf("Перевод от номера %s", fromUserPhone),
		Time:   transferTime,
	}

	if ws.transactions[toUserID] == nil {
		ws.transactions[toUserID] = []models.Transaction{}
	}
	ws.transactions[toUserID] = append(ws.transactions[toUserID], toTransaction)

	return &models.TransferResponse{Balance: fromAccount.Balance}, nil
}

// GetBackupData возвращает данные для бэкапа
func (ws *WalletService) GetBackupData() interface{} {
	ws.mux.RLock()
	defer ws.mux.RUnlock()

	// Создаем структуру для бэкапа
	backupData := struct {
		Accounts     map[string]map[string]*models.Account `json:"accounts"`
		Transactions map[string][]models.Transaction       `json:"transactions"`
		DailyTopups  map[string]map[string]int             `json:"daily_topups"`
		UserPhones   map[string]string                     `json:"user_phones"`
	}{
		Accounts:     make(map[string]map[string]*models.Account),
		Transactions: make(map[string][]models.Transaction),
		DailyTopups:  make(map[string]map[string]int),
		UserPhones:   make(map[string]string),
	}

	// Копируем аккаунты
	for userID, accounts := range ws.accounts {
		backupAccounts := make(map[string]*models.Account)
		for accountID, account := range accounts {
			backupAccount := &models.Account{
				ID:      account.ID,
				Type:    account.Type,
				Balance: account.Balance,
			}
			backupAccounts[accountID] = backupAccount
		}
		backupData.Accounts[userID] = backupAccounts
	}

	// Копируем транзакции
	for userID, transactions := range ws.transactions {
		backupTransactions := make([]models.Transaction, len(transactions))
		for i, transaction := range transactions {
			backupTransactions[i] = models.Transaction{
				Amount: transaction.Amount,
				Title:  transaction.Title,
				Time:   transaction.Time,
				Icon:   transaction.Icon,
			}
		}
		backupData.Transactions[userID] = backupTransactions
	}

	// Копируем дневные пополнения
	for userID, dailyTopups := range ws.dailyTopups {
		backupDailyTopups := make(map[string]int)
		for date, amount := range dailyTopups {
			backupDailyTopups[date] = amount
		}
		backupData.DailyTopups[userID] = backupDailyTopups
	}

	// Копируем номера телефонов
	for userID, phone := range ws.userPhones {
		backupData.UserPhones[userID] = phone
	}

	return backupData
}

// GetBackupFileName возвращает имя файла для бэкапа
func (ws *WalletService) GetBackupFileName() string {
	return "wallet_data"
}
