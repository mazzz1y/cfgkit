package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"cfgkit/internal/config"
	"cfgkit/internal/renderer"
)

type Logger interface {
	Info(string, ...any)
	LogRequest(ctx context.Context, r *http.Request, status int, user string, err error)
}

type Server struct {
	configDir  string
	workingDir string
	port       string
	logger     Logger
	limiter    *renderer.Limiter
}

func New(configDir, workingDir, port string, maxChecks int, logger Logger) *Server {
	return &Server{
		configDir:  configDir,
		workingDir: workingDir,
		port:       port,
		logger:     logger,
		limiter:    renderer.NewLimiter(maxChecks),
	}
}

func (s *Server) Start() error {
	s.logger.Info("listening", "addr", ":"+s.port)

	return http.ListenAndServe(":"+s.port, s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load(s.configDir)
	user, pass, ok := r.BasicAuth()

	if err != nil {
		s.errorResponse(w, r, 500, user, err)
		return
	}

	if !ok || !s.auth(cfg, user, pass) {
		s.errorResponse(w, r, 403, user, fmt.Errorf("auth failed"))
		return
	}

	rnd, err := renderer.New(cfg, s.workingDir, user, renderer.WithLimiter(s.limiter))
	if err != nil {
		s.errorResponse(w, r, 500, user, err)
		return
	}

	res, err := rnd.Render(r.Context())
	if errors.Is(err, renderer.ErrCheckBusy) {
		s.errorResponse(w, r, http.StatusServiceUnavailable, user, err)
		return
	}
	if err != nil {
		s.errorResponse(w, r, 500, user, err)
		return
	}

	w.Header().Set("Content-Type", res.ContentType)

	if _, e := w.Write(res.Data.Bytes()); e != nil {
		s.errorResponse(w, r, 500, user, e)
		return
	}

	s.logger.LogRequest(r.Context(), r, 200, user, nil)
}

func (s *Server) auth(cfg *config.Config, user, pass string) bool {
	for k, v := range cfg.Devices {
		if k == user && v.Password == pass {
			return true
		}
	}

	return false
}

func (s *Server) errorResponse(w http.ResponseWriter, r *http.Request, status int, user string, err error) {
	http.Error(w, http.StatusText(status), status)
	s.logger.LogRequest(r.Context(), r, status, user, err)
}
