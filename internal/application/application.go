package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"eats-backend/internal/api"
	"eats-backend/internal/config"
	"eats-backend/internal/models"
	"eats-backend/internal/service"
	"eats-backend/internal/storage"
	"eats-backend/pkg/runner"
)

type Application struct {
	cfg *config.Config

	productService *service.ProductsService
	tokenService   *service.TokenService
	userData       *service.UserData
	fileSaver      *storage.Storage
	logger         *zap.SugaredLogger

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
	a.userData = service.NewUserData()
	a.fileSaver = storage.NewStorage(a.logger, "data/uploads")
	a.productService = service.NewProductsService(
		a.userData,
		[]*models.Product{
			{
				ID:          "ff25265d-9dfc-49c3-bd01-678c6baa001f",
				Image:       "https://basket-01.wbbasket.ru/vol100/part10039/10039442/images/big/1.webp",
				Name:        "Мука",
				Weight:      123,
				Price:       1000,
				Rating:      5.6,
				Description: "Норм",
				Discount:    0,
				Reviews:     []models.Review{{Rating: 5, Author: "sdsadas", CreatedAt: time.Now(), Content: "ssdsdfsdfa"}},
			},
		}, map[string][]string{
			"favourite": {"ff25265d-9dfc-49c3-bd01-678c6baa001f"},
		}, map[string]models.Category{
			"favourite": {
				ID:    "favourite",
				Name:  "Любимое",
				Image: "https://basket-01.wbbasket.ru/vol100/part10039/10039442/images/big/1.webp",
			},
		},
	)

	a.tokenService = service.NewTokenService(a.cfg.PrivateKey, a.cfg.CreatedTokensPath)

	return nil
}

func (a *Application) initRouter(ctx context.Context) error {
	authMiddleware := api.NewAuthMiddleware(a.cfg.PublicKey, a.logger, a.cfg.RevokedTokens).JWTAuth

	router := api.NewRouter(
		a.cfg.ServerOpts,
		a.productService,
		a.userData,
		a.tokenService,
		a.fileSaver,
		authMiddleware,
		a.logger,
	)

	if err := runner.RunServer(ctx, router, a.cfg.ListenPort, a.errChan, &a.wg); err != nil {
		return fmt.Errorf("can't run public router: %w", err)
	}

	return nil
}
