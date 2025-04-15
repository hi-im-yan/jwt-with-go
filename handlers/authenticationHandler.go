package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthenticationHandler struct {
	DB *pgxpool.Pool
}

func NewAuthenticationHandler(db *pgxpool.Pool) *AuthenticationHandler {
	return &AuthenticationHandler{DB: db}
}

type newAccountRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

func (ah *AuthenticationHandler) AuthRouter() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("POST /register", ApiHandlerAdapter(ah.RegisterNewAccount))
	r.HandleFunc("POST /login", ApiHandlerAdapter(ah.Login))
	return r
}

// This function creates a JWT token with the given username and role
func (ah *AuthenticationHandler) CreateJwtToken(username string, role string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Minute * 15).Unix(),
	}
	log.Printf("[APIHandler:CreateJwtToken] Creating JWT token with claims %v", claims)
	// Create a new token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with a secret key
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		log.Printf("[APIHandler:CreateJwtToken] Error creating JWT token: %v", err)
		return "", err
	}

	log.Printf("[APIHandler:CreateJwtToken] Successfully created JWT token")
	return tokenString, nil
}

// RegisterNewAccount godoc
// @Summary      Register a new account
// @Description  Creates a new user account with name, email, and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        user  body      newAccountRequest  true  "New Account Info"
// @Success      201   {object}  authResponse
// @Failure      400   {object}  ErrorResponse "Invalid request body"
// @Failure      409   {object}  ErrorResponse "Email already in use"
// @Failure      500   {object}  ErrorResponse "Internal server error"
// @Router       /register [post]
func (ah *AuthenticationHandler) RegisterNewAccount(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[AuthenticationHandler:registerNewAccount] start")

	defer r.Body.Close()

	// parse request to userRequest struct
	var newAccountReq newAccountRequest
	err := json.NewDecoder(r.Body).Decode(&newAccountReq)

	// Could not parse json to request
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Invalid request body", Detail: "Not a valid JSON"},
		}
	}

	log.Printf("[AuthenticationHandler:registerNewAccount] Request body received with {name: %s, email: %s}", newAccountReq.Name, newAccountReq.Email)

	// validate request body
	if newAccountReq.Email == "" || newAccountReq.Password == "" || newAccountReq.Name == "" {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Invalid request body", Detail: "email, name and password are required"},
		}
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(newAccountReq.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[AuthenticationHandler:login] Error hashing password: %v", err)
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[AuthenticationHandler:registerNewAccount] Inserting new user with {name: %s} and {email: %s}", newAccountReq.Name, newAccountReq.Email)

	// insert user
	query := `INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, 'user') RETURNING id, name, email, role;`
	insertedAccount := &user{}
	err = ah.DB.QueryRow(r.Context(), query, newAccountReq.Name, newAccountReq.Email, encryptedPassword).Scan(&insertedAccount.ID, &insertedAccount.Name, &insertedAccount.Email, &insertedAccount.Role)
	if err != nil {
		log.Printf("[AuthenticationHandler:registerNewAccount] Error inserting user: %v", err)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // Unique constraint violation (email already exists)
				return nil, &HandlerError{
					Status: http.StatusConflict,
					Message: ErrorResponse{
						Code:    "E409",
						Message: "Conflict",
						Detail:  "Email is already in use. Please use a different email.",
					},
				}
			}
		}
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[AuthenticationHandler:registerNewAccount] User inserted: %+v", insertedAccount)

	token, err := ah.CreateJwtToken(insertedAccount.Name, insertedAccount.Role)

	if err != nil {
		log.Printf("[AuthenticationHandler:registerNewAccount] Error creating JWT token: %v", err)
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[AuthenticationHandler:registerNewAccount] end in %s", time.Since(start))

	return &HandlerSuccess{
		Status: http.StatusCreated,
		Data:   &authResponse{Message: "Account created successfully", Token: token},
	}, nil
}

// Login godoc
// @Summary      Login with credentials
// @Description  Authenticates a user using email and password, returns a JWT. If trying to login as admin, check credentials in the .env file.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      loginRequest  true  "User Credentials"
// @Success      200          {object}  authResponse
// @Failure      400          {object}  ErrorResponse "Invalid request body"
// @Failure      401          {object}  ErrorResponse "Invalid email or password"
// @Failure      500          {object}  ErrorResponse "Internal server error"
// @Router       /login [post]
func (ah *AuthenticationHandler) Login(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[AuthenticationHandler:login] start")

	defer r.Body.Close()

	// parse request to userRequest struct
	var loginReq loginRequest
	err := json.NewDecoder(r.Body).Decode(&loginReq)

	// Could not parse json to request
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Invalid request body", Detail: "Not a valid JSON"},
		}
	}

	log.Printf("[AuthenticationHandler:login] Request body received for login: %s", loginReq.Email)

	// validate request body
	if loginReq.Email == "" || loginReq.Password == "" {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Invalid request body", Detail: "email and password are required"},
		}
	}

	log.Printf("[AuthenticationHandler:login] Validating user with {email: %s}", loginReq.Email)

	// validate user
	query := `SELECT id, name, email, role, password FROM users WHERE email = $1`
	user := &user{}
	var hashedPassword string
	err = ah.DB.QueryRow(r.Context(), query, loginReq.Email).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &hashedPassword)
	if err != nil {
		log.Printf("[AuthenticationHandler:login] Error validating user: %v", err)
		if err == pgx.ErrNoRows {
			return nil, &HandlerError{
				Status: http.StatusUnauthorized,
				Message: ErrorResponse{
					Code:    "E401",
					Message: "Unauthorized",
					Detail:  "Invalid email or password",
				},
			}
		}
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginReq.Password))
	if err != nil {
		log.Printf("[AuthenticationHandler:login] Error validating user: %v", err)
		return nil, &HandlerError{
			Status: http.StatusUnauthorized,
			Message: ErrorResponse{
				Code:    "E401",
				Message: "Unauthorized",
				Detail:  "Invalid email or password",
			},
		}
	}

	log.Printf("[AuthenticationHandler:login] User validated: %+v", user)

	token, err := ah.CreateJwtToken(user.Name, user.Role)

	if err != nil {
		log.Printf("[AuthenticationHandler:login] Error creating JWT token: %v", err)
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[AuthenticationHandler:login] end in %s", time.Since(start))

	return &HandlerSuccess{
		Status: http.StatusOK,
		Data:   &authResponse{Message: "Login successful", Token: token},
	}, nil
}
