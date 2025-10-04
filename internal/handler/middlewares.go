package handler

import (
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/zap"

	api "eats-backend/api/generated"
)

func Logging(logger *zap.SugaredLogger) api.Middleware {
	return func(
		req middleware.Request,
		next func(req middleware.Request) (middleware.Response, error),
	) (middleware.Response, error) {
		logger := logger.With(
			"operation", req.OperationName,
			"operationId", req.OperationID,
		)
		resp, err := next(req)
		if err != nil {
			logger.Error("Fail: ", err)
		} else {
			if tresp, ok := resp.Type.(interface{ GetStatusCode() int }); ok {
				logger = logger.With("status_code", tresp.GetStatusCode())
			}
			logger.Info("Success")
		}
		return resp, err
	}
}
