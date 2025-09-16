package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/service/worker"
	"github.com/danglnh07/zola/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// Interface for OAuth
type OAuth interface {
	HandleOAuth(ctx *gin.Context)
	HandleCallback(ctx *gin.Context)
}

// Google OAuth struct, which holds specific OAuth configs, and dependencies for login action
type GoogleOAuth struct {
	OAuthConfig *oauth2.Config
	queries     *db.Queries
	distributor worker.TaskDistributor
	jwtService  *security.JWTService
	config      *util.Config
	logger      *slog.Logger
}

// This is the response from OAuth provider when fetching data, not data return to client
type UserDataResp struct {
	ID       string `json:"id"`
	Username string `json:"name"`
	Email    string `json:"email"`
}

// Constructor method for GoogleAuth
func NewGoogleOAuth(
	queries *db.Queries,
	distributor worker.TaskDistributor,
	jwtService *security.JWTService,
	config *util.Config,
	logger *slog.Logger,
) OAuth {
	// Config Google OAuth2
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
		distributor: distributor,
		jwtService:  jwtService,
		config:      config,
		logger:      logger,
	}
}

// Handler for OAuth login. It will redirect to the login page of the OAuth provider
func (auth *GoogleOAuth) HandleOAuth(ctx *gin.Context) {
	// Get the role from query parameter and building the state
	role := ctx.Query("role")
	if role == "" {
		role = string(db.User)
	}

	state := base64.URLEncoding.EncodeToString([]byte(role))
	url := auth.OAuthConfig.AuthCodeURL(state)
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

// Handler for processing callback
func (auth *GoogleOAuth) HandleCallback(ctx *gin.Context) {
	// Get the code return by OAuth provider
	code := ctx.Query("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Internal server error"})
		return
	}

	// Exchange code to get access token
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

	// Get the state and role
	state, err := base64.URLEncoding.DecodeString(ctx.Query("state"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid state"})
		return
	}
	role := db.Role(string(state))

	if role != db.User && role != db.Admin {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid state"})
		return
	}

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
			account.Password = sql.NullString{Valid: false}
			account.OauthProvider = sql.NullString{String: string(db.Google), Valid: true}
			account.OauthProviderID = sql.NullString{String: userData.ID, Valid: true}
			account.TokenVersion = 1
			result = auth.queries.DB.Create(&account)
			if result.Error != nil {
				auth.logger.Error("GET /oauth2/callback: failed to inset user data into database")
				ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
				return
			}

			// Create background tasks: send verification email
			err = auth.distributor.DistributeTaskSendEmail(context.Background(), worker.EmailPayload{
				Email:    account.Email,
				Username: account.Username,
			})
			if err != nil {
				auth.logger.Error("GET /oauth2/callback: failed to distribute \"send welcome email\" task",
					"error", err)
				// Should NOT return here
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
		account.ID, role, security.AccessToken, int(account.TokenVersion),
	)
	if err != nil {
		auth.logger.Error("GET /oauth2/callback: failed to create JWT access token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}
	refreshToken, err := auth.jwtService.CreateToken(
		account.ID, role, security.RefreshToken, int(account.TokenVersion),
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
