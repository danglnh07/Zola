package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/service/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserData struct {
	ID       uint   `json:"id"` // Account ID
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	UserData UserData `json:"user"`
	Tokens   Tokens   `json:"tokens"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" bidning:"required"`
	Role     string `json:"role" binding:"required"`
}

func (server *Server) HandleRegister(ctx *gin.Context) {
	// Get and validate request body
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body: " + err.Error()})
		return
	}

	// Check role
	role := db.Role(req.Role)
	if role != db.User && role != db.Admin {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid role"})
		return
	}

	// Hash password
	hashed, err := security.BcryptHash(req.Password)
	if err != nil {
		server.logger.Error("POST /api/auth/register: failed to hash password")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Insert into database
	result := server.queries.DB.Create(&db.Account{
		Model:           gorm.Model{},
		Username:        req.Username,
		Email:           req.Email,
		Password:        sql.NullString{String: hashed, Valid: true},
		Role:            role,
		OauthProvider:   sql.NullString{Valid: false},
		OauthProviderID: sql.NullString{Valid: false},
		TokenVersion:    1,
	})
	if result.Error != nil {
		// If email or username already taken
		if strings.Contains(result.Error.Error(), "accounts_email_key") {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Email already taken"})
			return
		}

		if strings.Contains(result.Error.Error(), "accounts_username_key") {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Username already taken"})
			return
		}

		// Other database error
		server.logger.Error("POST /api/auth/register: failed to create account", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Distribute background task: send welcome email
	// Create background tasks: send verification email
	err = server.distributor.DistributeTaskSendEmail(context.Background(), worker.Payload{
		Email:    req.Email,
		Username: req.Username,
	})
	if err != nil {
		server.logger.Error("POST /api/auth/register: failed to distribute \"send welcome email\" task",
			"error", err)
		// Should NOT return here
	}

	// Send message back to client (NOT generate tokens here)
	ctx.JSON(http.StatusCreated, "Register successfully")
}

type LoginRequest struct {
	// We allow login via username or email, so we will validate manually instead of using binding for these fields
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

func (server *Server) HandleLogin(ctx *gin.Context) {
	// Get and validate request
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Error("POST /api/auth/login", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if username or email exists
	if req.Username == "" && req.Email == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Missing credential, please provide at least username or email"})
		return
	}

	// Fetch data from database
	var account db.Account
	result := server.queries.DB.Where("username = ? OR email = ?", req.Username, req.Email).Find(&account)
	if result.Error != nil {
		// If login credential is incorrect
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, ErrorResponse{"Incorrect login credential"})
			return
		}

		// Other database errors
		server.logger.Error("POST /api/auth/login: failed to fetch account data from database", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// If found, compate password
	if !security.BcryptCompare(account.Password.String, req.Password) {
		ctx.JSON(http.StatusNotFound, ErrorResponse{"Incorrect login credential"})
		return
	}

	// Create JWT tokens and return it back to client
	accessToken, err := server.jwtService.CreateToken(
		account.ID, account.Role, security.AccessToken, int(account.TokenVersion),
	)
	if err != nil {
		server.logger.Error("POST /api/auth/login: failed to create JWT access token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}
	refreshToken, err := server.jwtService.CreateToken(
		account.ID, account.Role, security.RefreshToken, int(account.TokenVersion),
	)
	if err != nil {
		server.logger.Error("POST /api/auth/login: failed to create JWT refresh token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, AuthResponse{
		UserData: UserData{
			ID:       account.ID,
			Username: account.Username,
			Email:    account.Email,
		},
		Tokens: Tokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	})
}

func (server *Server) HandleRefreshToken(ctx *gin.Context) {
	// Get claims from refresh token from context
	claims, _ := ctx.Get(claimsKey)
	customClaims := claims.(*security.CustomClaims)

	// Increase the token version in database
	result := server.queries.DB.
		Table("accounts").
		Where("id", customClaims.ID).
		Update("token_version", gorm.Expr("token_version + ?", 1))
	if result.Error != nil {
		server.logger.Error("POST /auth/token/refresh: failed to update token version", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Create new access token and refresh token
	accessToken, err := server.jwtService.CreateToken(
		customClaims.ID, customClaims.Role, security.AccessToken, int(customClaims.Version+1),
	)
	if err != nil {
		server.logger.Error("POST /auth/token/refresh: failed to create JWT access token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}
	refreshToken, err := server.jwtService.CreateToken(
		customClaims.ID, customClaims.Role, security.RefreshToken, int(customClaims.Version+1),
	)
	if err != nil {
		server.logger.Error("POST /auth/token/refresh: failed to create JWT access token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return the access token back to client
	ctx.JSON(http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (server *Server) HandleLogout(ctx *gin.Context) {
	// Get claims from access token from context
	claims, _ := ctx.Get(claimsKey)
	customClaims := claims.(*security.CustomClaims)

	// Increase the token version in database
	result := server.queries.DB.
		Table("accounts").
		Where("id", customClaims.ID).
		Update("token_version", gorm.Expr("token_version + ?", 1))
	if result.Error != nil {
		server.logger.Error("POST /auth/logout: failed to update token version", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, "Logout successfully")
}
