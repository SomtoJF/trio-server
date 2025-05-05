package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/somtojf/trio-server/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Endpoint struct {
	DB     *gorm.DB
	Domain string
}

func NewEndpoint(db *gorm.DB, domain string) *Endpoint {
	return &Endpoint{DB: db, Domain: domain}
}

type signUpInput struct {
	Username string `json:"userName" binding:"required,max=20"`
	FullName string `json:"fullName" binding:"required,max=50"`
	Password string `json:"password" binding:"required,max=20,min=8"`
}

type loginInput struct {
	Username string `json:"userName" binding:"required,max=20"`
	Password string `json:"password" binding:"required,max=20,min=8"`
}

type passwordResetRequest struct {
	Password    string `json:"password" binding:"required,max=20"`
	NewPassword string `json:"newPassword" binding:"required,max=20"`
}

// Login godoc
//
//	@Summary		Login user
//	@Description	Logs in a user and returns an access token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			loginInput	body		loginInput				true	"Login credentials"
//	@Success		200			{object}	map[string]interface{}	"success message"
//	@Failure		400			{object}	map[string]interface{}	"error message"
//	@Failure		500			{object}	map[string]interface{}	"internal server error"
//	@Router			/login [post]
func (e *Endpoint) Login(c *gin.Context) {
	var body loginInput

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userFound models.User
	e.DB.Where("username=?", body.Username).Find(&userFound)

	if userFound.IdUser == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username or password is incorrect"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userFound.PasswordHash), []byte(body.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username or password is incorrect"})
		return
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userFound.ExternalID.String(),
		"username": userFound.Username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occured"})
	}

	var secure bool
	if strings.Contains(e.Domain, "http://localhost") {
		secure = true
	}

	cookie := http.Cookie{
		Name:     "Access_Token",
		Value:    token,
		Path:     "/",
		Domain:   e.Domain,
		MaxAge:   604800,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(c.Writer, &cookie)

	c.JSON(200, gin.H{
		"message": "success",
	})
}

// GuestLogin godoc
//
//	@Summary		Create guest account
//	@Description	Creates a temporary guest account with a random username and returns an access token
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}	"Guest account created successfully"
//	@Failure		500	{object}	map[string]interface{}	"Internal server error"
//	@Router			/guest-login [post]
func (e *Endpoint) GuestLogin(c *gin.Context) {
	guestId, err := nanoid.Generate(nanoid.DefaultAlphabet, 6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occured"})
		return
	}

	userName := fmt.Sprintf("guest-%s", guestId)

	user := models.User{
		Username:     userName,
		FullName:     fmt.Sprintf("Guest-%s", guestId),
		PasswordHash: "",
		IsGuest:      true,
	}

	if err := e.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occured"})
		return
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       user.ExternalID.String(),
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occured"})
		return
	}

	var secure bool
	if strings.Contains(e.Domain, "http://localhost") {
		secure = true
	}

	cookie := http.Cookie{
		Name:     "Access_Token",
		Value:    token,
		Path:     "/",
		Domain:   e.Domain,
		MaxAge:   604800,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(c.Writer, &cookie)

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// Signup godoc
//
//	@Summary		Signup a new user
//	@Description	Creates a new user account
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			userInput	body		signUpInput				true	"User details"
//	@Success		201			{object}	map[string]interface{}	"Account created successfully"
//	@Failure		400			{object}	map[string]interface{}	"Bad request"
//	@Failure		500			{object}	map[string]interface{}	"Internal server error"
//	@Router			/signup [post]
func (e *Endpoint) Signup(c *gin.Context) {
	var body signUpInput

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userFound models.User
	e.DB.Where("username=?", body.Username).Find(&userFound)

	if userFound.IdUser != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username taken"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Username:     body.Username,
		FullName:     body.FullName,
		PasswordHash: string(passwordHash),
	}

	if err := e.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error creating user: %s", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "account created successfully",
		"data":    body,
	})
}

// Logout godoc
//
//	@Summary		Logout user
//	@Description	Logs out the user by clearing the access token
//	@Tags			auth
//	@Success		200	{object}	map[string]interface{}	"Logout successful"
//	@Router			/logout [post]
func (e *Endpoint) Logout(c *gin.Context) {
	c.SetCookie("Access_Token", "", -1, "/", e.Domain, false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logout successful"})
}

// ResetPassword godoc
//
//	@Summary		Reset user password
//	@Description	Resets the password for the authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			passwordResetRequest	body		passwordResetRequest	true	"Password reset details"
//	@Success		200						{object}	map[string]interface{}	"Password updated successfully"
//	@Failure		400						{object}	map[string]interface{}	"Bad request"
//	@Failure		401						{object}	map[string]interface{}	"Unauthorized"
//	@Failure		500						{object}	map[string]interface{}	"Internal server error"
//	@Router			/reset-password [post]
func (e *Endpoint) ResetPassword(c *gin.Context) {
	var body passwordResetRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the current user from the context
	user, ok := c.Value("currentUser").(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Verify the current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// Update the password in the database
	user.PasswordHash = string(hashedPassword)
	if err := e.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.SetCookie("Access_Token", "", -1, "/", e.Domain, false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully. Please login with new password"})
}

// GetCurrentUser godoc
//
//	@Summary		Get current user
//	@Description	Retrieves the current authenticated user's information
//	@Tags			users
//	@Success		200	{object}	models.User				"Current user data"
//	@Failure		500	{object}	map[string]interface{}	"Internal server error"
//	@Router			/me [get]
func (e *Endpoint) GetCurrentUser(c *gin.Context) {
	user := c.Value("currentUser")
	if user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "We couldn't retrieve your data"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}
