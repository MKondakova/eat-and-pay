package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"eats-backend/internal/config"
	"eats-backend/internal/models"

	"github.com/rs/cors"
	"go.uber.org/zap"
)

var (
	errInvalidPaginationParameter = errors.New("invalid pagination parameter")
	errEmptyID                    = errors.New("empty id")
	errEmptyName                  = errors.New("empty name")
)

type FileSaver interface {
	SaveFile(w http.ResponseWriter, r *http.Request) (string, error)
}

type UserData interface {
	GetProfile(ctx context.Context) (*models.UserProfile, error)
	UpdateProfile(ctx context.Context, data models.UpdateUserRequest) error
	DeleteProfile(ctx context.Context) error
}

type AddressService interface {
	GetAddresses(ctx context.Context) []*models.Address
	AddAddress(ctx context.Context, address *models.Address) error
	RemoveAddress(ctx context.Context, addressID string) error
	UpdateAddress(ctx context.Context, newAddress *models.Address) error
}

type ProductsService interface {
	GetProductsList(ctx context.Context, page, pageSize int, category string) (models.ProductsList, error)
	GetProductByID(ctx context.Context, id string) (models.Product, error)
	GetCategories() []models.Category
	AddReview(ctx context.Context, review models.PostReviewRequest, productID string) error
	AddFavourite(ctx context.Context, id string) error
	RemoveFavourite(ctx context.Context, id string) error
}

type CartService interface {
	GetCart(ctx context.Context) (models.CartResponse, error)
	AddItem(ctx context.Context, productID string) (int, error)
	RemoveItem(ctx context.Context, productID string) (int, error)
}

type OrderService interface {
	GetOrders(ctx context.Context) ([]*models.Order, error)
	MakeNewOrder(ctx context.Context, orderRequest *models.OrderRequest) error
}

type TokenService interface {
	GenerateToken(ctx context.Context, username string, isTeacher bool) (string, error)
}

type Router struct {
	*http.Server
	router *http.ServeMux

	productsService ProductsService
	userData        UserData
	addressService  AddressService
	cartService     CartService
	orderService    OrderService
	tokenService    TokenService
	fileSaver       FileSaver

	logger *zap.SugaredLogger
}

func NewRouter(
	cfg config.ServerOpts,
	productsService ProductsService,
	userData UserData,
	addressService AddressService,
	cartService CartService,
	orderService OrderService,
	tokenService TokenService,
	fileSaver FileSaver,
	authMiddleware func(next http.HandlerFunc) http.HandlerFunc,
	logger *zap.SugaredLogger,
) *Router {
	innerRouter := http.NewServeMux()

	appRouter := &Router{
		Server: &http.Server{
			Handler:      cors.AllowAll().Handler(innerRouter),
			ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
			IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
		},
		router:          innerRouter,
		productsService: productsService,
		userData:        userData,
		addressService:  addressService,
		cartService:     cartService,
		orderService:    orderService,
		tokenService:    tokenService,
		logger:          logger,
		fileSaver:       fileSaver,
	}

	innerRouter.HandleFunc("GET /users/me", authMiddleware(appRouter.getUser))
	innerRouter.HandleFunc("PUT /users/me", authMiddleware(appRouter.updateProfile))
	innerRouter.HandleFunc("DELETE /users/me", authMiddleware(appRouter.deleteUser))

	innerRouter.HandleFunc("POST /logout", authMiddleware(appRouter.logout))

	innerRouter.HandleFunc("GET /products", authMiddleware(appRouter.getProductsList))
	innerRouter.HandleFunc("GET /products/{id}", authMiddleware(appRouter.getProductByID))

	innerRouter.HandleFunc("POST /products/{id}/favourite", authMiddleware(appRouter.addFavourite))
	innerRouter.HandleFunc("DELETE /products/{id}/favourite", authMiddleware(appRouter.deleteFavourite))

	innerRouter.HandleFunc("POST /products/{id}/reviews", authMiddleware(appRouter.addReview))

	innerRouter.HandleFunc("GET /categories", authMiddleware(appRouter.getCategories))

	innerRouter.HandleFunc("GET /cart", authMiddleware(appRouter.getCart))
	innerRouter.HandleFunc("POST /cart/items", authMiddleware(appRouter.addToCart))
	innerRouter.HandleFunc("DELETE /cart/items/{id}", authMiddleware(appRouter.removeFromCart))

	innerRouter.HandleFunc("GET /orders", authMiddleware(appRouter.getOrders))
	innerRouter.HandleFunc("POST /orders", authMiddleware(appRouter.makeOrder))

	innerRouter.HandleFunc("GET /addresses", authMiddleware(appRouter.getAddresses))
	innerRouter.HandleFunc("POST /addresses", authMiddleware(appRouter.addAddress))
	innerRouter.HandleFunc("PUT /addresses/{id}", authMiddleware(appRouter.updateAddress))
	innerRouter.HandleFunc("DELETE /addresses/{id}", authMiddleware(appRouter.deleteAddress))

	innerRouter.HandleFunc("POST /createToken", authMiddleware(appRouter.createToken))
	innerRouter.HandleFunc("POST /createTeacherToken", authMiddleware(appRouter.createTeacherToken))

	uploadsDir := http.Dir("data/uploads")
	innerRouter.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(uploadsDir)))
	innerRouter.HandleFunc("POST /uploads", authMiddleware(appRouter.saveFile))

	innerRouter.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "redoc-static.html")
	})

	return appRouter
}

func (r *Router) sendResponse(response http.ResponseWriter, request *http.Request, code int, buf []byte) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)
	_, err := response.Write(buf)
	if err != nil {
		r.logger.With(
			"module", "api",
			"request_url", request.Method+": "+request.URL.Path,
		).Errorf("Error sending error response: %v", err)
	}
}

func (r *Router) sendErrorResponse(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, models.ErrBadRequest):
		response.WriteHeader(http.StatusBadRequest)
		r.logger.With(
			"module", "api",
			"request_url", request.Method+": "+request.URL.Path,
		).Warn(err)
		r.writeError(response, request, err)

		return
	case errors.Is(err, models.ErrNotFound):
		response.WriteHeader(http.StatusNotFound)
		r.logger.With(
			"module", "api",
			"request_url", request.Method+": "+request.URL.Path,
		).Warn(err)

		r.writeError(response, request, err)

		return
	case errors.Is(err, models.ErrForbidden):
		response.WriteHeader(http.StatusForbidden)
		r.logger.With(
			"module", "api",
			"request_url", request.Method+": "+request.URL.Path,
		).Warn(err)

		r.writeError(response, request, err)

		return
	case errors.Is(err, models.ErrUnauthorized):
		response.WriteHeader(http.StatusUnauthorized)
		r.logger.With(
			"module", "api",
			"request_url", request.Method+": "+request.URL.Path,
		).Warn(err)

		r.writeError(response, request, err)

		return
	}

	response.WriteHeader(http.StatusInternalServerError)
	r.logger.With(
		"module", "api",
		"request_url", request.Method+": "+request.URL.Path,
	).Error(err)

	r.writeError(response, request, err)
}

func (r *Router) writeError(response http.ResponseWriter, request *http.Request, err error) {
	body := map[string]string{"error": err.Error()}

	result, err := json.Marshal(body)
	if err != nil {
		r.logger.With("request_url", request.Method+": "+request.URL.Path).
			Error(fmt.Errorf("error marshalling error body: %v", err))
	}

	_, err = response.Write(result)
	if err != nil {
		r.logger.With(
			"module", "api",
			"request_url", request.Method+": "+request.URL.Path,
		).Errorf("Error sending error response: %v", err)
	}
}

func (r *Router) saveFile(writer http.ResponseWriter, request *http.Request) {
	filename, err := r.fileSaver.SaveFile(writer, request)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("SaveFile: %w", err))

		return
	}

	responseBody := map[string]string{"file": filename}

	buf, err := json.Marshal(responseBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) getProductsList(writer http.ResponseWriter, request *http.Request) {
	page, err := getPaginationParameter(request, "page", 1)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	pageSize, err := getPaginationParameter(request, "pageSize", models.DefaultPageSize)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	category := request.URL.Query().Get("category")

	result, err := r.productsService.GetProductsList(request.Context(), page, pageSize, category)
	if err != nil {
		r.sendErrorResponse(writer, request, err)

		return
	}

	buf, err := json.Marshal(result)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) getProductByID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	product, err := r.productsService.GetProductByID(request.Context(), id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("GetProductByID: %w", err))

		return
	}

	buf, err := json.Marshal(product)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) addReview(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}
	var requestBody models.PostReviewRequest

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	err = r.productsService.AddReview(request.Context(), requestBody, id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("AddReview: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) addFavourite(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	err := r.productsService.AddFavourite(request.Context(), id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("AddFavourite: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) deleteFavourite(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	err := r.productsService.RemoveFavourite(request.Context(), id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("RemoveFavourite: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) getUser(writer http.ResponseWriter, request *http.Request) {
	result, err := r.userData.GetProfile(request.Context())
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("GetProfile: %w", err))

		return
	}

	buf, err := json.Marshal(result)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) deleteUser(writer http.ResponseWriter, request *http.Request) {
	err := r.userData.DeleteProfile(request.Context())
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("DeleteProfile: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) updateProfile(writer http.ResponseWriter, request *http.Request) {
	var requestBody models.UpdateUserRequest

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	err = r.userData.UpdateProfile(request.Context(), requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("UpdateProfile: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) logout(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func (r *Router) getAddresses(writer http.ResponseWriter, request *http.Request) {
	addresses := r.addressService.GetAddresses(request.Context())

	buf, err := json.Marshal(addresses)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) addAddress(writer http.ResponseWriter, request *http.Request) {
	var requestBody models.Address

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	err = r.addressService.AddAddress(request.Context(), &requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("AddAddress: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) updateAddress(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	var requestBody models.Address

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	requestBody.ID = id

	err = r.addressService.UpdateAddress(request.Context(), &requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("UpdateAddress: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) deleteAddress(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	err := r.addressService.RemoveAddress(request.Context(), id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("RemoveAddress: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) getCategories(writer http.ResponseWriter, request *http.Request) {
	result := r.productsService.GetCategories()

	buf, err := json.Marshal(result)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) getCart(writer http.ResponseWriter, request *http.Request) {
	cart, err := r.cartService.GetCart(request.Context())
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("GetCart: %w", err))

		return
	}

	buf, err := json.Marshal(cart)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) addToCart(writer http.ResponseWriter, request *http.Request) {
	id := request.URL.Query().Get("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	amount, err := r.cartService.AddItem(request.Context(), id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("AddToCart: %w", err))

		return
	}

	response := map[string]any{
		"total": amount,
	}

	buf, err := json.Marshal(response)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) removeFromCart(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyID))

		return
	}

	amount, err := r.cartService.RemoveItem(request.Context(), id)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("RemoveItem: %w", err))

		return
	}

	response := map[string]any{
		"total": amount,
	}

	buf, err := json.Marshal(response)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) getOrders(writer http.ResponseWriter, request *http.Request) {
	orders, err := r.orderService.GetOrders(request.Context())
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("GetOrders: %w", err))

		return
	}

	buf, err := json.Marshal(orders)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func (r *Router) makeOrder(writer http.ResponseWriter, request *http.Request) {
	var requestBody models.OrderRequest

	err := json.NewDecoder(request.Body).Decode(&requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, err))

		return
	}

	err = r.orderService.MakeNewOrder(request.Context(), &requestBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("MakeNewOrder: %w", err))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (r *Router) createToken(writer http.ResponseWriter, request *http.Request) {
	name := request.URL.Query().Get("name")
	if name == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyName))

		return
	}

	token, err := r.tokenService.GenerateToken(request.Context(), name, false)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("CreateToken: %w", err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, []byte(token))
}

func (r *Router) createTeacherToken(writer http.ResponseWriter, request *http.Request) {
	name := request.URL.Query().Get("name")
	if name == "" {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrBadRequest, errEmptyName))

		return
	}

	token, err := r.tokenService.GenerateToken(request.Context(), name, true)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("CreateToken: %w", err))

		return
	}

	responseBody := TokenResponse{
		Token: token,
	}

	buf, err := json.Marshal(responseBody)
	if err != nil {
		r.sendErrorResponse(writer, request, fmt.Errorf("%w: %w", models.ErrInternalServer, err))

		return
	}

	r.sendResponse(writer, request, http.StatusOK, buf)
}

func getPaginationParameter(request *http.Request, parameterName string, defaultValue int) (int, error) {
	parameter := request.URL.Query().Get(parameterName)

	if parameter == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(parameter)
	if err != nil {
		return 0, fmt.Errorf("%w %s: %w", errInvalidPaginationParameter, parameterName, err)
	}

	if value <= 0 {
		return 0, fmt.Errorf("%w %s: %d", errInvalidPaginationParameter, parameterName, value)
	}

	return value, nil
}
