package handler

import (
	"context"
	api "eats-backend/api/generated"
)

type SecurityHandler struct{}

func (h *SecurityHandler) HandleBearerAuth(ctx context.Context, _ api.OperationName, _ api.BearerAuth) (context.Context, error) {
	return ctx, nil
}
