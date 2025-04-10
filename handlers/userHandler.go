package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	db        *pgxpool.Pool
	logPrefix string
}

// User Response Model
type user struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// User Request Model
type userRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewUserHandler(db *pgxpool.Pool) *UserHandler {
	return &UserHandler{db: db, logPrefix: "UserHandler"}
}

// Configuration of routes
func (uh *UserHandler) UserRouter() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(logSomething)

	// Routes
	r.HandleFunc("POST /", ApiHandlerAdapter(uh.insertUser))
	r.HandleFunc("GET /", ApiHandlerAdapter(uh.getAllUsers))
	r.HandleFunc("GET /{id}", ApiHandlerAdapter(uh.getUser))
	r.HandleFunc("PUT /{id}", ApiHandlerAdapter(uh.updateUser))
	r.HandleFunc("DELETE /{id}", ApiHandlerAdapter(uh.deleteUser))
	r.HandleFunc("GET /mock", ApiHandlerAdapter(uh.getMockUser))

	return r
}

func logSomething(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("This middleware should be applied only for user routes")
		next.ServeHTTP(w, r)
	})
}

// getMockUser returns a mock user for demonstration purposes
//
//	@Summary		Get mock user
//	@Description	Returns a mock user for testing purposes
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	user
//	@Failure		404	{object}	ErrorResponse
//	@Router			/users/mock [get]
func (uh *UserHandler) getMockUser(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	shouldReturnUser := true

	if shouldReturnUser {
		return &HandlerSuccess{
			Status: http.StatusOK,
			Data:   &user{ID: 1, Name: "Yan", Email: "XO2iM@example.com"},
		}, nil
	}

	return nil, &HandlerError{
		Status:  http.StatusNotFound,
		Message: ErrorResponse{Code: "E404", Message: "User not found", Detail: ""},
	}
}

// @Summary		Insert a new user
// @Description	Inserts a new user into the database
// @Tags			users
// @Accept			json
// @Produce		json
// @Param			request	body	userRequest	true	"User request"
// @Success		201	{object}	user
// @Failure		400	{object}	ErrorResponse
// @Router			/users [post]
func (uh *UserHandler) insertUser(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[UserHandler:insertUser] start")

	defer r.Body.Close()

	// parse request to userRequest struct
	var insertUserReq userRequest
	err := json.NewDecoder(r.Body).Decode(&insertUserReq)

	// Could not parse json to request
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Invalid request body", Detail: "Not a valid JSON"},
		}
	}

	log.Printf("[UserHandler:insertUser] Request body received: %+v", insertUserReq)

	// validate request body
	reqName, reqEmail := insertUserReq.Name, insertUserReq.Email
	if reqName == "" || reqEmail == "" {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Invalid request body", Detail: "name and email are required"},
		}
	}

	log.Printf("[UserHandler:insertUser] Inserting user with {name: %s} and {email: %s}", reqName, reqEmail)

	// insert user
	query := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email;`
	insertedUser := &user{}
	err = uh.db.QueryRow(context.Background(), query, reqName, reqEmail).Scan(&insertedUser.ID, &insertedUser.Name, &insertedUser.Email)
	if err != nil {
		log.Printf("[UserHandler:insertUser] Error inserting user: %v", err)
		// Check if the error is a PostgreSQL unique constraint violation
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

	log.Printf("[UserHandler:insertUser] Inserted user: %+v", insertedUser)
	log.Printf("[UserHandler:insertUser] end. Took %v", time.Since(start))
	return &HandlerSuccess{
		Status: http.StatusCreated,
		Data:   insertedUser,
	}, nil
}

// @Summary		Get all users
// @Description	Gets all users from the database
// @Tags			users
// @Produce		json
// @Success		200	{array}	user
// @Failure		500	{object}	ErrorResponse
// @Router			/users [get]
func (uh *UserHandler) getAllUsers(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[UserHandler:getAllUsers] start")

	// Query all users
	log.Printf("[UserHandler:getAllUsers] Querying all users")
	rows, err := uh.db.Query(context.Background(), `SELECT id, name, email FROM users;`)
	if err != nil {
		log.Printf("[UserHandler:getAllUsers] Error querying all users: %v", err)
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}
	defer rows.Close()

	// Scan all users
	log.Printf("[UserHandler:getAllUsers] Creating users slice from rows")
	var allUsers []user
	for rows.Next() {
		var u user
		err = rows.Scan(&u.ID, &u.Name, &u.Email)
		if err != nil {
			log.Printf("[UserHandler:getAllUsers] Error scanning user row: %v. Parsing error.", err)
			return nil, &HandlerError{
				Status:  http.StatusInternalServerError,
				Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
			}
		}
		allUsers = append(allUsers, u)
	}

	// Return all users
	log.Printf("[UserHandler:getAllUsers] end. Took %v", time.Since(start))
	return &HandlerSuccess{
		Status: http.StatusOK,
		Data:   allUsers,
	}, nil
}

// @Summary		Get user by id
// @Description	Gets a user from the database by id
// @Tags			users
// @Produce		json
// @Param			id	path	int	true	"User ID"
// @Success		200	{object}	user
// @Failure		400	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router			/users/{id} [get]
func (uh *UserHandler) getUser(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[UserHandler:getUser] start")

	// Parsing path parameter
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Not a valid id", Detail: "Path parameter 'id' must be an integer"},
		}
	}

	log.Printf("[UserHandler:getUser] Querying user with id %d", id)
	var user user
	err = uh.db.QueryRow(context.Background(), `SELECT id, name, email FROM users WHERE id = $1;`, id).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &HandlerError{
				Status:  http.StatusNotFound,
				Message: ErrorResponse{Code: "E404", Message: "Not found", Detail: "User with id " + idStr + " not found"},
			}
		}
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[UserHandler:getUser] end. Took %v", time.Since(start))
	return &HandlerSuccess{
		Status: http.StatusOK,
		Data:   user,
	}, nil
}

// @Summary		Update user by id
// @Description	Updates a user in the database by id
// @Tags			users
// @Produce		json
// @Param			id	path	int	true	"User ID"
// @Param			user	body	user	true	"User object"
// @Success		200	{object}	user
// @Failure		400	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router			/users/{id} [put]
func (uh *UserHandler) updateUser(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[UserHandler:updateUser] start")

	// Parsing path parameter
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Not a valid id", Detail: "Path parameter 'id' must be an integer"},
		}
	}

	defer r.Body.Close()

	// parse request to userRequest struct
	var updateUserReq userRequest
	err = json.NewDecoder(r.Body).Decode(&updateUserReq)
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Bad request", Detail: "Invalid request body"},
		}
	}

	log.Printf("[UserHandler:updateUser] Request body received: %+v", updateUserReq)

	// validate request
	if updateUserReq.Name == "" || updateUserReq.Email == "" {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Bad request", Detail: "name and email are required"},
		}
	}

	// update user
	log.Printf("[UserHandler:updateUser] Updating user with id %d with {name: %s} and {email: %s}", id, updateUserReq.Name, updateUserReq.Email)
	updatedUser := &user{}
	query := `UPDATE users SET name = $1, email = $2 WHERE id = $3 RETURNING id, name, email;`
	err = uh.db.QueryRow(context.Background(), query, updateUserReq.Name, updateUserReq.Email, id).Scan(&updatedUser.ID, &updatedUser.Name, &updatedUser.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &HandlerError{
				Status:  http.StatusNotFound,
				Message: ErrorResponse{Code: "E404", Message: "Not found", Detail: "User with id " + idStr + " not found"},
			}
		}
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[UserHandler:updateUser] User updated: %+v", updatedUser)
	log.Printf("[UserHandler:updateUser] end. Took %v", time.Since(start))
	return &HandlerSuccess{
		Status: http.StatusOK,
		Data:   updatedUser,
	}, nil
}

// @Summary		Delete user by id
// @Description	Deletes a user from the database by id
// @Tags			users
// @Produce		json
// @Param			id	path	int	true	"User ID"
// @Success		204
// @Failure		400	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
func (uh *UserHandler) deleteUser(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	start := time.Now()
	log.Printf("[UserHandler:deleteUser] start")

	// Parsing path parameter
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, &HandlerError{
			Status:  http.StatusBadRequest,
			Message: ErrorResponse{Code: "E400", Message: "Not a valid id", Detail: "Path parameter 'id' must be an integer"},
		}
	}

	// delete user
	log.Printf("[UserHandler:deleteUser] Deleting user with id %d", id)
	query := `DELETE FROM users WHERE id = $1;`
	_, err = uh.db.Exec(context.Background(), query, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &HandlerError{
				Status:  http.StatusNotFound,
				Message: ErrorResponse{Code: "E404", Message: "Not found", Detail: "User with id " + idStr + " not found"},
			}
		}
		return nil, &HandlerError{
			Status:  http.StatusInternalServerError,
			Message: ErrorResponse{Code: "E500", Message: "Internal Server Error", Detail: "Something went wrong. Contact support or try again later"},
		}
	}

	log.Printf("[UserHandler:deleteUser] User deleted with id %d", id)
	log.Printf("[UserHandler:deleteUser] end. Took %v", time.Since(start))
	return &HandlerSuccess{
		Status: http.StatusNoContent,
		Data:   nil,
	}, nil
}
