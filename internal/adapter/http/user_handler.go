package http

import (
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

// Listar todos os usuários
func (h *UserHandler) GetUsers(c *gin.Context) {
	// Verifica se o usuário atual é admin
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
		return
	}

	var users []model.UserEntity
	result := h.db.Order("created_at desc").Find(&users)
	if result.Error != nil {
		h.logger.Error("Erro ao listar usuários", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao listar usuários"})
		return
	}

	// Converter para resposta (sem expor senhas)
	var response []gin.H
	for _, user := range users {
		response = append(response, gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

// Obter usuário por ID
func (h *UserHandler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuário não fornecido"})
		return
	}

	// Verificar se o usuário atual é admin ou o proprietário desta conta
	currentUser, exists := getCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não autenticado"})
		return
	}

	if currentUser.ID != id && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
		return
	}

	var user model.UserEntity
	result := h.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
			return
		}
		h.logger.Error("Erro ao buscar usuário", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar usuário"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"username":  user.Username,
		"email":     user.Email,
		"role":      user.Role,
		"createdAt": user.CreatedAt,
		"updatedAt": user.UpdatedAt,
	})
}

// Atualizar usuário
type UpdateUserRequest struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	Email    *string `json:"email"`
	Role     *string `json:"role"`
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuário não fornecido"})
		return
	}

	// Verificar se o usuário atual é admin ou o proprietário desta conta
	currentUser, exists := getCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não autenticado"})
		return
	}

	// Apenas admin pode atualizar outros usuários ou alterar roles
	isCurrentUserAdmin := currentUser.Role == "admin"
	isSelfUpdate := currentUser.ID == id

	if !isCurrentUserAdmin && !isSelfUpdate {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	// Buscar o usuário existente
	var user model.UserEntity
	result := h.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
			return
		}
		h.logger.Error("Erro ao buscar usuário para atualização", zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar usuário"})
		return
	}

	// Atualizar campos, apenas se fornecidos
	updates := make(map[string]interface{})
	updates["updated_at"] = time.Now()

	if req.Username != nil {
		// Verificar se o novo username já existe
		if *req.Username != user.Username {
			var existingUser model.UserEntity
			checkResult := h.db.Where("username = ? AND id != ?", *req.Username, id).First(&existingUser)
			if checkResult.Error == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Nome de usuário já está em uso"})
				return
			} else if checkResult.Error != gorm.ErrRecordNotFound {
				h.logger.Error("Erro ao verificar username existente", zap.Error(checkResult.Error))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar nome de usuário"})
				return
			}
			updates["username"] = *req.Username
		}
	}

	if req.Email != nil {
		// Verificar se o novo email já existe
		if *req.Email != user.Email {
			var existingUser model.UserEntity
			checkResult := h.db.Where("email = ? AND id != ?", *req.Email, id).First(&existingUser)
			if checkResult.Error == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email já está em uso"})
				return
			} else if checkResult.Error != gorm.ErrRecordNotFound {
				h.logger.Error("Erro ao verificar email existente", zap.Error(checkResult.Error))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar email"})
				return
			}
			updates["email"] = *req.Email
		}
	}

	if req.Password != nil {
		// Hash da nova senha
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			h.logger.Error("Erro ao gerar hash da senha", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar senha"})
			return
		}
		updates["password"] = string(hashedPassword)
	}

	if req.Role != nil {
		// Apenas admin pode alterar a role
		if !isCurrentUserAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Apenas administradores podem alterar roles"})
			return
		}
		updates["role"] = *req.Role
	}

	// Se não há nada para atualizar
	if len(updates) <= 1 { // apenas updated_at
		c.JSON(http.StatusOK, gin.H{
			"message": "Nenhuma alteração necessária",
			"id":      id,
		})
		return
	}

	// Aplicar atualizações
	updateResult := h.db.Model(&user).Updates(updates)
	if updateResult.Error != nil {
		h.logger.Error("Erro ao atualizar usuário", zap.Error(updateResult.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar usuário"})
		return
	}

	h.logger.Info("Usuário atualizado com sucesso",
		zap.String("id", user.ID),
		zap.Int64("alterações", updateResult.RowsAffected))

	c.JSON(http.StatusOK, gin.H{
		"message": "Usuário atualizado com sucesso",
		"id":      id,
	})
}

// Excluir usuário
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuário não fornecido"})
		return
	}

	// Apenas admin pode excluir usuários
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
		return
	}

	// Verificar se o usuário existe
	var user model.UserEntity
	checkResult := h.db.Where("id = ?", id).First(&user)
	if checkResult.Error != nil {
		if checkResult.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
			return
		}
		h.logger.Error("Erro ao verificar existência do usuário", zap.Error(checkResult.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar usuário"})
		return
	}

	// Não permitir excluir o último admin
	if user.Role == "admin" {
		var adminCount int64
		h.db.Model(&model.UserEntity{}).Where("role = ?", "admin").Count(&adminCount)
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Não é possível excluir o último administrador do sistema",
			})
			return
		}
	}

	// Excluir o usuário
	deleteResult := h.db.Delete(&user)
	if deleteResult.Error != nil {
		h.logger.Error("Erro ao excluir usuário", zap.Error(deleteResult.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir usuário"})
		return
	}

	h.logger.Info("Usuário excluído com sucesso", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{
		"message": "Usuário excluído com sucesso",
		"id":      id,
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	// Obter o contexto atual
	ctx := c.Request.Context()

	// Criar span para a operação do handler
	ctx, span := otel.Tracer("api-gateway.handler").Start(
		ctx,
		"UserHandler.Login",
		trace.WithAttributes(
			attribute.String("handler", "login"),
			attribute.String("path", c.FullPath()),
		),
	)
	defer span.End()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetStatus(codes.Error, "invalid request")
		span.SetAttributes(attribute.Bool("error", true))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Buscar usuário pelo username
	var user model.UserEntity
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(attribute.Bool("error", true))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciais inválidas"})
		return
	}

	// Verificar senha
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		span.SetStatus(codes.Error, "invalid credentials")
		span.SetAttributes(attribute.Bool("error", true))
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

// Funções auxiliares

// isAdmin verifica se o usuário é administrador
func isAdmin(c *gin.Context) bool {
	userValue, exists := c.Get("user")
	if !exists {
		return false
	}

	user, ok := userValue.(*model.User)
	return ok && user.Role == "admin"
}

// getCurrentUser obtém o usuário atual do contexto
func getCurrentUser(c *gin.Context) (*model.User, bool) {
	userValue, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	user, ok := userValue.(*model.User)
	if !ok {
		return nil, false
	}

	return user, true
}
