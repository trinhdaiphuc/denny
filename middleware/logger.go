package middleware

import (
	"fmt"
	"github.com/whatvn/denny"
	"github.com/whatvn/denny/log"
	"time"
)

var (
	logKey = "dennyLogger"
)

func Logger() denny.HandleFunc {
	return func(ctx *denny.Context) {
		logger := log.New(&log.JSONFormatter{})
		var (
			clientIP = ctx.ClientIP()
			method   = ctx.Request.Method

			userAgent = ctx.Request.UserAgent()
			uri       = ctx.Request.RequestURI
			errs      string
		)

		logger.WithFields(map[string]interface{}{
			"ClientIP":      clientIP,
			"RequestMethod": method,
			"UserAgent":     userAgent,
			"Uri":           uri,
		})
		ctx.Set(logKey, logger)
		ctx.Next()
		var (
			statusCode = ctx.Writer.Status()
		)
		logger.WithField("Status", statusCode)
		if ctx.Errors != nil {
			bs, err := ctx.Errors.MarshalJSON()
			if err == nil {
				errs = string(bs)
			}
		}
		if len(errs) > 0 {
			logger.WithField("Errors", errs)
		}
		logger.Infof(time.Now().Format(time.RFC3339))
	}
}

func GetLogger(ctx *denny.Context) *log.Log {
	logger, ok := ctx.Get(logKey)
	if !ok {
		return log.New()
	}
	if l, ok := logger.(*log.Log); ok {
		return l
	}
	panic(fmt.Errorf("%v is not logger", logger))
}
