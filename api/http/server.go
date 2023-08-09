package http

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"hanafi_fiqh_qa/internal/auth"
	"hanafi_fiqh_qa/internal/base/crypto"
	"hanafi_fiqh_qa/internal/user"
)

type Config interface {
	DetailedError() bool
	Address() string
}

type ServerOpts struct {
	UserUsecases user.UserUsecases
	AuthService  auth.AuthService
	Crypto       crypto.Crypto
	Config       Config
}

func NewServer(opts ServerOpts) *Server {
	gin.SetMode(gin.ReleaseMode)

	server := &Server{
		engine:       gin.New(),
		config:       opts.Config,
		crypto:       opts.Crypto,
		userUsecases: opts.UserUsecases,
		authService:  opts.AuthService,
	}

	initRouter(server)

	return server
}

type Server struct {
	engine       *gin.Engine
	config       Config
	crypto       crypto.Crypto
	userUsecases user.UserUsecases
	authService  auth.AuthService
}

func (s Server) Listen() error {
	fmt.Printf("API server listening at: %s\n\n", s.config.Address())
	return s.engine.Run(s.config.Address())
}
