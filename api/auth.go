package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// User data return to client
type UserData struct {
	ID       uint   `json:"id"` // Account ID
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Struct holds both access token and refresh token
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Response struct after login
type AuthResponse struct {
	UserData UserData `json:"user"`
	Tokens   Tokens   `json:"tokens"`
}

// OAuth interface
type OAuth interface {
	HandleOAuth(ctx *gin.Context)
	HandleCallback(ctx *gin.Context)
}

// OAuth implementation of Google
type GoogleOAuth struct {
	OAuthConfig *oauth2.Config
	queries     *db.Queries
	jwtService  *security.JWTService
	config      *util.Config
	logger      *slog.Logger
}

// This is the response from OAuth provider, not data return to client
type UserDataResp struct {
	ID       string `json:"id"`
	Username string `json:"name"`
	Email    string `json:"email"`
}

func NewGoogleAuth(
	queries *db.Queries,
	jwtService *security.JWTService,
	config *util.Config,
	logger *slog.Logger,
) OAuth {
	googleConfig := &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/oauth2/callback", config.BaseURL),
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	return &GoogleOAuth{
		OAuthConfig: googleConfig,
		queries:     queries,
		jwtService:  jwtService,
		config:      config,
		logger:      logger,
	}
}

func (auth *GoogleOAuth) HandleOAuth(ctx *gin.Context) {
	url := auth.OAuthConfig.AuthCodeURL("")
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (auth *GoogleOAuth) HandleCallback(ctx *gin.Context) {
	// Get the code return by OAuth provider
	code := ctx.Query("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Internal server error"})
		return
	}

	token, err := auth.OAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		auth.logger.Error("GET /oauth2/callback: failed to exchange code for token", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Fetch user data
	client := auth.OAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		auth.logger.Error("GET /oauth2/callback: failed to fetch user data from OAuth provider")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}
	defer resp.Body.Close()

	// Get user data from response
	var userData UserDataResp
	if err = json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		auth.logger.Error("GET /oauth2/callback: failed to decode user data fetch from OAuth provider", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Fetch user from database to check if they exists first
	var account = db.Account{}
	result := auth.queries.DB.
		Where("oauth_provider = ? AND oauth_provider_id = ?", db.Google, userData.ID).
		First(&account)
	if result.Error != nil {
		// If not found any user with this oauth_id -> create account
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			account.Username = userData.Username
			account.Email = userData.Email
			account.OauthProvider = string(db.Google)
			account.OauthProviderID = userData.ID
			account.TokenVersion = 1
			result = auth.queries.DB.Create(&account)
			if result.Error != nil {
				auth.logger.Error("GET /oauth2/callback: failed to inset user data into database")
				ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
				return
			}
		} else {
			// Other database errors
			auth.logger.Error("GET /oauth2/callback: failed to fetch user data from database")
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
			return
		}
	}

	// Create JWT tokens and return it back to client
	accessToken, err := auth.jwtService.CreateToken(
		account.ID, security.AccessToken, int(account.TokenVersion),
	)
	if err != nil {
		auth.logger.Error("GET /oauth2/callback: failed to create JWT access token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}
	refreshToken, err := auth.jwtService.CreateToken(
		account.ID, security.RefreshToken, int(account.TokenVersion),
	)
	if err != nil {
		auth.logger.Error("GET /oauth2/callback: failed to create JWT refresh token")
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
