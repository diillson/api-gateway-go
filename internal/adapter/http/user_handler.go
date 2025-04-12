package http

import (
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid" // Adicione esta dependência
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type UserHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserHandler(db *gorm.DB, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		db:     db,
		logger: logger,
	}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Role     string `json:"role"`
}

func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("JSON inválido", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	h.logger.Info("Tentativa de registro de usuário",
		zap.String("username", req.Username),
		zap.String("email", req.Email))

	// Definir role padrão se não especificada
	if req.Role == "" {
		req.Role = "user"
	}

	// Hash da senha
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Erro ao gerar hash da senha", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar senha"})
		return
	}

	// Criar usuário
	now := time.Now()
	user := model.UserEntity{
		ID:        uuid.New().String(), // Gerar ID único
		Username:  req.Username,
		Password:  string(hashedPassword),
		Email:     req.Email,
		Role:      req.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Criar a tabela se não existir
	if !h.db.Migrator().HasTable(&model.UserEntity{}) {
		h.logger.Info("Tabela de usuários não existe, criando...")
		err := h.db.AutoMigrate(&model.UserEntity{})
		if err != nil {
			h.logger.Error("Erro ao criar tabela de usuários", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno do servidor"})
			return
		}
	}

	// Verificar se o usuário já existe
	var existingUser model.UserEntity
	result := h.db.Where("username = ?", req.Username).First(&existingUser)
	if result.Error == nil {
		h.logger.Warn("Tentativa de criar usuário já existente", zap.String("username", req.Username))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nome de usuário já existe"})
		return
	} else if result.Error != gorm.ErrRecordNotFound {
		h.logger.Error("Erro ao verificar usuário existente", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar usuário"})
		return
	}

	// Verificar se o email já existe
	result = h.db.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		h.logger.Warn("Tentativa de criar usuário com email já existente", zap.String("email", req.Email))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email já está em uso"})
		return
	} else if result.Error != gorm.ErrRecordNotFound {
		h.logger.Error("Erro ao verificar email existente", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar email"})
		return
	}

	// Criar o usuário com tratamento de erro detalhado
	result = h.db.Create(&user)
	if result.Error != nil {
		h.logger.Error("Erro ao criar usuário no banco de dados",
			zap.String("username", req.Username),
			zap.String("email", req.Email),
			zap.Error(result.Error))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erro ao criar usuário: " + result.Error.Error(),
		})
		return
	}

	if result.RowsAffected == 0 {
		h.logger.Error("Nenhuma linha afetada ao criar usuário")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar usuário: nenhum registro criado"})
		return
	}

	h.logger.Info("Usuário criado com sucesso",
		zap.String("id", user.ID),
		zap.String("username", user.Username))

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Usuário criado com sucesso",
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Buscar usuário pelo username
	var user model.UserEntity
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciais inválidas"})
		return
	}

	// Verificar senha
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciais inválidas"})
		return
	}

	// Gerar token JWT
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Usar a função centralizada para obter o segredo JWT
	secretKey := security.GetJWTSecret()

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		h.logger.Error("Erro ao gerar token JWT", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
