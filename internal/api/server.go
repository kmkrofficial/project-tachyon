package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"project-tachyon/internal/config"
	"project-tachyon/internal/core"
	"project-tachyon/internal/security"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ControlServer struct {
	engine     *core.TachyonEngine
	cfg        *config.ConfigManager
	audit      *security.AuditLogger
	router     *chi.Mux
	activeReqs int64
}

func NewControlServer(engine *core.TachyonEngine, cfg *config.ConfigManager, audit *security.AuditLogger) *ControlServer {
	s := &ControlServer{
		engine: engine,
		cfg:    cfg,
		audit:  audit,
		router: chi.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// ... (Start and setupRoutes remain same, but Concurrency Middleware changes)

func (s *ControlServer) concurrencyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		max := int64(s.cfg.GetAIMaxConcurrent())
		if max <= 0 {
			max = 1 // Safety default
		}

		// Increment and check
		current := atomic.AddInt64(&s.activeReqs, 1)
		defer atomic.AddInt64(&s.activeReqs, -1)

		if current > max {
			s.audit.Log("127.0.0.1", r.UserAgent(), "Overloaded "+r.URL.Path, 429, "Max Concurrent Reached")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *ControlServer) Start(port int) {
	// 1. Feature Flag Check at Startup
	if !s.cfg.GetEnableAI() {
		return // Do not start if disabled
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("Control Server listening on %s", addr)

	go func() {
		// Enforce loopback for the listener itself as an extra layer
		conn, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("Control Server failed to bind: %v", err)
			return
		}

		if err := http.Serve(conn, s.router); err != nil {
			log.Printf("Control Server failed: %v", err)
		}
	}()
}

func (s *ControlServer) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// Security Middleware Chain
	s.router.Use(s.securityMiddleware)
	s.router.Use(s.concurrencyLimitMiddleware)

	s.router.Post("/v1/queue", s.handleQueueDownload)
	s.router.Post("/v1/browser/trigger", s.handleBrowserTrigger)
	s.router.Get("/v1/tasks/{id}", s.handleGetTask)
	s.router.Post("/v1/tasks/{id}/control", s.handleTaskControl)
	s.router.Get("/v1/status", s.handleGetStatus)
}

func (s *ControlServer) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		userAgent := r.UserAgent()
		action := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

		// 1. Feature Flag Check (Runtime)
		if !s.cfg.GetEnableAI() {
			// Even if listener is running (dynamic disable), reject
			s.audit.Log(sourceIP, userAgent, action, 503, "Feature Disabled")
			http.Error(w, "AI Interface Disabled", http.StatusServiceUnavailable)
			return
		}

		// 2. Localhost Enforcement
		// Note: net.SplitHostPort might return "::1" or "127.0.0.1"
		if sourceIP != "127.0.0.1" && sourceIP != "::1" {
			s.audit.Log(sourceIP, userAgent, action, 403, "External Access Denied")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 3. Token Auth
		token := r.Header.Get("X-Tachyon-Token")
		expectedToken := s.cfg.GetAIToken()

		if token != expectedToken {
			s.audit.Log(sourceIP, userAgent, action, 401, "Invalid Token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Success Log
		s.audit.Log(sourceIP, userAgent, action, 200, "Authorized")
		next.ServeHTTP(w, r)
	})
}

// Request/Response Models
type EnqueueRequest struct {
	URL      string `json:"url"`
	Path     string `json:"path"`     // Optional custom path
	Filename string `json:"filename"` // Optional custom filename
	Priority int    `json:"priority"` // Optional 1-3
}

type EnqueueResponse struct {
	TaskID string `json:"task_id"`
}

type ControlRequest struct {
	Action string `json:"action"` // "pause", "resume", "cancel", "delete"
}

func (s *ControlServer) handleQueueDownload(w http.ResponseWriter, r *http.Request) {
	var req EnqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.audit.Log("127.0.0.1", r.UserAgent(), "POST /queue", 400, "Bad Request JSON")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := s.engine.StartDownload(req.URL, req.Path, req.Filename, nil)
	if err != nil {
		s.audit.Log("127.0.0.1", r.UserAgent(), "POST /queue", 500, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Priority > 0 {
		s.engine.SetPriority(id, req.Priority)
	}

	json.NewEncoder(w).Encode(EnqueueResponse{TaskID: id})
}

func (s *ControlServer) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := s.engine.GetTask(id)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(task)
}

func (s *ControlServer) handleTaskControl(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var err error
	switch req.Action {
	case "pause":
		err = s.engine.PauseDownload(id)
	case "resume":
		err = s.engine.ResumeDownload(id)
	case "cancel", "stop":
		err = s.engine.StopDownload(id)
	case "delete":
		err = s.engine.DeleteDownload(id, false)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ControlServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status": "running"}`))
}
