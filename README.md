# Go JWT Auth API

A simple JWT authentication API in Go.

## Table of Contents

* [Overview](#overview)
* [Features](#features)
* [Getting Started](#getting-started)
* [API Endpoints](#api-endpoints)
* [Security](#security)
* [License](#license)

## Overview

This project provides a basic JWT authentication API in Go, using the Chi router and PostgreSQL as the database. It includes features for user registration, login, and authentication.

## Features

* User registration with email, name, and password
* Login with email and password, returning a JWT token
* Authentication using JWT tokens
* Support for admin users

## Getting Started

### Prerequisites

* Go 1.17 or later
* PostgreSQL 13 or later
* A .env file with the following environment variables:
	+ DB_HOST
	+ DB_USER
	+ DB_PASSWORD
	+ DB_NAME
	+ DB_PORT
	+ JWT_SECRET_KEY
	+ ADMIN_EMAIL
	+ ADMIN_PASSWORD

### Running the Application

1. Clone the repository: `git clone [https://github.com/hi-im-yan/jwt-with-go.git`](git clone [https://github.com/hi-im-yan/jwt-with-go.git`)
2. Navigate to the project directory: `cd your-repo-name`
3. Install dependencies: `go get ./...`
4. Create a .env file with the required environment variables
5. Run the application: `go run main.go`

## API Endpoints

### Authentication

* `POST /login`: Login with email and password, returning a JWT token
* `POST /register`: Register a new user with email, name, and password

### Users

* `GET /users`: Get all users (admin only)
* `GET /users/{id}`: Get a user by ID (admin only)
* `PUT /users/{id}`: Update a user's name and email (admin only)
* `DELETE /users/{id}`: Delete a user by ID (admin only)

### Health Check

* `GET /`: Health check endpoint

## Security

* JWT tokens are used for authentication
* Passwords are hashed using bcrypt
* Environment variables are used to store sensitive data
