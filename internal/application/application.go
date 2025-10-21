package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"eats-backend/internal/api"
	"eats-backend/internal/config"
	"eats-backend/internal/service"
	"eats-backend/internal/storage"
	"eats-backend/pkg/runner"
)

type Application struct {
	cfg *config.Config

	addressService    *service.AddressService
	cartService       *service.Cart
	favouritesService *service.Favourites
	orderService      *service.OrderService
	productService    *service.ProductsService
	tokenService      *service.TokenService
	userData          *service.UserData
	walletService     *service.WalletService
	fileSaver         *storage.Storage
	backupService     *service.BackupService
	logger            *zap.SugaredLogger

	errChan chan error
	wg      sync.WaitGroup
	ready   bool
}

func New() *Application {
	return &Application{
		errChan: make(chan error),
	}
}

func (a *Application) Start(ctx context.Context) error {
	if err := a.initConfigAndLogger(); err != nil {
		return err
	}

	if err := a.initServices(); err != nil {
		return err
	}

	if err := a.initRouter(ctx); err != nil {
		return err
	}

	// Запускаем сервис бэкапа в отдельной горутине
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.backupService.Start(ctx)
	}()

	return nil
}

func (a *Application) Wait(ctx context.Context, cancel context.CancelFunc) error {
	var appErr error

	errWg := sync.WaitGroup{}

	errWg.Add(1)

	go func() {
		defer errWg.Done()

		for err := range a.errChan {
			cancel()
			a.logger.Error(err)
			appErr = err
		}
	}()

	<-ctx.Done()
	a.wg.Wait()
	close(a.errChan)
	errWg.Wait()

	return appErr
}

func (a *Application) Ready() bool {
	return a.ready
}

func (a *Application) HandleGracefulShutdown(ctx context.Context, cancel context.CancelFunc) error {
	var appErr error

	errWg := sync.WaitGroup{}

	errWg.Add(1)

	go func() {
		defer errWg.Done()

		for err := range a.errChan {
			cancel()
			a.logger.Error(err)
			appErr = err
		}
	}()

	<-ctx.Done()
	a.wg.Wait()
	close(a.errChan)
	errWg.Wait()

	return appErr
}

func (a *Application) initConfigAndLogger() error {
	if err := a.initLogger(); err != nil {
		return fmt.Errorf("can't init logger: %w", err)
	}

	if err := a.initConfig(); err != nil {
		return fmt.Errorf("can't init config: %w", err)
	}

	return nil
}

func (a *Application) initConfig() error {
	var err error

	a.cfg, err = config.GetConfig(a.logger)
	if err != nil {
		return fmt.Errorf("can't parse config: %w", err)
	}

	return nil
}

func (a *Application) initLogger() error {
	zapLog, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("can't create logger: %w", err)
	}

	a.logger = zapLog.Sugar()

	return nil
}

func (a *Application) initServices() error {
	a.addressService = service.NewAddressService()

	// Инициализируем сервисы с данными из конфига
	a.favouritesService = service.NewFavouritesService(a.cfg.InitialFavourites)
	a.userData = service.NewUserData(a.cfg.InitialUserProfiles)

	a.fileSaver = storage.NewStorage(a.logger, "data/uploads")
	a.productService = service.NewProductsService(
		a.favouritesService,
		a.cfg.InitialProductsData,
		a.cfg.InitialProductCategories,
		a.cfg.InitialCategories,
	)

	a.cartService = service.NewCart(a.productService, a.logger, a.cfg.InitialCartItems)
	a.orderService = service.NewOrderService(a.addressService, a.cartService, a.cfg.InitialOrders)
	a.tokenService = service.NewTokenService(a.cfg.PrivateKey, a.cfg.CreatedTokensPath)
	a.walletService = service.NewWalletService(a.userData)

	// Инициализируем сервис бэкапа (каждые 24 часа)
	a.backupService = service.NewBackupService(a.logger, "data", 24*time.Hour)

	// Регистрируем все сервисы для бэкапа
	a.backupService.RegisterBackupable(a.userData)
	a.backupService.RegisterBackupable(a.cartService)
	a.backupService.RegisterBackupable(a.favouritesService)
	a.backupService.RegisterBackupable(a.orderService)
	a.backupService.RegisterBackupable(a.walletService)

	return nil
}

func (a *Application) initRouter(ctx context.Context) error {
	authMiddleware := api.NewAuthMiddleware(a.cfg.PublicKey, a.logger, a.cfg.RevokedTokens).JWTAuth
	loggingMiddleware := api.NewLoggerMiddleware(a.logger).Middleware

	router := api.NewRouter(
		a.cfg.ServerOpts,
		a.productService,
		a.userData,
		a.addressService,
		a.cartService,
		a.orderService,
		a.tokenService,
		a.walletService,
		a.fileSaver,
		authMiddleware,
		loggingMiddleware,
		a.logger,
	)

	if err := runner.RunServer(ctx, router, a.cfg.ListenPort, a.errChan, &a.wg); err != nil {
		return fmt.Errorf("can't run public router: %w", err)
	}

	return nil
}
