package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"subscription-manager/cache"
	"subscription-manager/config"
	"subscription-manager/middleware"
)

type AuthHandler struct {
	db    *sql.DB
	cache *cache.Cache
	cfg   *config.Config
}

func NewAuthHandler(db *sql.DB, c *cache.Cache, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cache: c, cfg: cfg}
}

func (h *AuthHandler) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func (h *AuthHandler) Register(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"Error": "Email and password are required."})
		return
	}
	if len(password) < 8 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"Error": "Password must be at least 8 characters.", "Email": email})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"Error": "Internal error, please try again."})
		return
	}

	var id string
	err = h.db.QueryRowContext(c.Request.Context(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, string(hash),
	).Scan(&id)
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"Error": "That email is already registered.", "Email": email})
		return
	}

	h.issueTokenCookie(c, id, email)
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	var id, hash string
	err := h.db.QueryRowContext(c.Request.Context(),
		`SELECT id, password_hash FROM users WHERE email = $1`, email,
	).Scan(&id, &hash)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid email or password.", "Email": email})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid email or password.", "Email": email})
		return
	}

	h.issueTokenCookie(c, id, email)
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if tokenStr, ok := c.Get("token"); ok {
		if s, ok := tokenStr.(string); ok {
			h.cache.BlacklistToken(c.Request.Context(), s, 25*time.Hour)
		}
	}
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) issueTokenCookie(c *gin.Context, userID, email string) {
	exp := time.Now().Add(24 * time.Hour)
	claims := &middleware.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(h.cfg.JWTSecret))
	c.SetCookie("token", signed, int(time.Until(exp).Seconds()), "/", "", false, true)
}
