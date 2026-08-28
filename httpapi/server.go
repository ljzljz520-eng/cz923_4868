package httpapi

import (
	"log"
	"net/http"
	"strings"
	"time"

	"pharmacycounter/catalog"
	"pharmacycounter/query"
	"pharmacycounter/service"
)

type Server struct {
	business   *service.Service
	queries    *query.Service
	catalog    *catalog.Catalog
	staticPath string
	logger     *log.Logger
}

func New(business *service.Service, queries *query.Service, medicines *catalog.Catalog, staticPath string, logger *log.Logger) *Server {
	return &Server{business: business, queries: queries, catalog: medicines, staticPath: staticPath, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/dashboard", s.handleDashboard)
	mux.HandleFunc("/api/orders", s.handleOrders)
	mux.HandleFunc("/api/orders/", s.handleOrderAction)
	mux.HandleFunc("/api/medicines", s.handleMedicines)
	mux.HandleFunc("/api/reports/daily", s.handleDailyReport)
	mux.HandleFunc("/api/reports/medicines", s.handleMedicineReport)
	static := http.FileServer(http.Dir(s.staticPath))
	mux.Handle("/", spaFallback(static, s.staticPath))
	return s.recoverMiddleware(s.loggingMiddleware(s.headersMiddleware(mux)))
}

func (s *Server) headersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(writer, request)
		if s.logger != nil {
			s.logger.Printf("method=%s path=%s duration=%s", request.Method, request.URL.Path, time.Since(started))
		}
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if s.logger != nil {
					s.logger.Printf("panic path=%s value=%v", request.URL.Path, recovered)
				}
				writeError(writer, http.StatusInternalServerError, "服务器内部错误")
			}
		}()
		next.ServeHTTP(writer, request)
	})
}

func spaFallback(static http.Handler, root string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			writeError(writer, http.StatusNotFound, "接口不存在")
			return
		}
		static.ServeHTTP(writer, request)
	})
}
