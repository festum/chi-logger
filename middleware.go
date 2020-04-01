package chilogger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type chilogger struct {
	logZ *zap.Logger
	logS *zap.SugaredLogger
	logL *logrus.Logger
	name string
}

// NewLogrusMiddleware returns a new Logrus Middleware handler.
func NewLogrusMiddleware(name string, logger *logrus.Logger) func(next http.Handler) http.Handler {
	return chilogger{
		logL: logger,
		name: name,
	}.middleware
}

// NewZapMiddleware returns a new Zap Middleware handler.
func NewZapMiddleware(name string, logger *zap.Logger) func(next http.Handler) http.Handler {
	return chilogger{
		logZ: logger,
		name: name,
	}.middleware
}

// NewZapSugaredMiddleware returns a new ZapSugaredLogger Middleware handler.
func NewZapSugaredMiddleware(name string, logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return chilogger{
		logS: logger,
		name: name,
	}.middleware
}

func (c chilogger) middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var requestID string
		if reqID := r.Context().Value(middleware.RequestIDKey); reqID != nil {
			requestID = reqID.(string)
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		latency := time.Since(start)

		switch {
		case c.logZ != nil:
			fields := []zapcore.Field{
				zap.Int("status", ww.Status()),
				zap.Duration("took", latency),
				zap.Int64(fmt.Sprintf("measure#%s.latency", c.name), latency.Nanoseconds()),
				zap.String("remote", r.RemoteAddr),
				zap.String("request", r.RequestURI),
				zap.String("method", r.Method),
			}
			if requestID != "" {
				fields = append(fields, zap.String("request-id", requestID))
			}
			c.logZ.Info("request completed", fields...)
		case c.logS != nil:
			fields := []interface{}{
				zap.Int("status", ww.Status()),
				zap.Duration("took", latency),
				zap.Int64(fmt.Sprintf("measure#%s.latency", c.name), latency.Nanoseconds()),
				zap.String("remote", r.RemoteAddr),
				zap.String("request", r.RequestURI),
				zap.String("method", r.Method),
			}
			if requestID != "" {
				fields = append(fields, zap.String("request-id", requestID))
			}
			c.logS.Infow("request completed", fields...)
		case c.logL != nil:
			fields := logrus.Fields{
				"status": ww.Status(),
				"took":   latency,
				fmt.Sprintf("measure#%s.latency", c.name): latency.Nanoseconds(),
				"remote":  r.RemoteAddr,
				"request": r.RequestURI,
				"method":  r.Method,
			}
			if requestID != "" {
				fields["request-id"] = requestID
			}
			c.logL.WithFields(fields).Info("request completed")
		}
	}
	return http.HandlerFunc(fn)
}
