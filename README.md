# Go2Bank Bankend 

## API Endpoints
- **POST** `/users` : Register a new user
- **POST** `/users/login` : Login from your username and password
- **GET** `/users/renew_access` : Renew your access token
- **GET** `/users/logout` : Logout and redirect to home page 

⚠️ Following API Endpoints will go through Paseto Authentication Middleware and some will go through RBAC Authorization
- **PATCH** `/users/update` : Update user information
- **GET** `/users/me` : Get a user information
- **GET** `/users/renew_access` : Renew your access token
- **POST** `/accounts` : create a new account by a user
- **GET** `/accounts/:id` : Get a account information
- **GET** `/accounts/` : List all accounts
- **POST** `/transfers` : create a new transfer between two accounts

## Tech Stack
- Gin
- Sqlc
- Postgres
- Redis
- Paseto token
- Github action
- Docker
- Crontab

## TODO
- [ ] Improve Testing coverage
- [ ] Build deposit and withdraw endpoints with OTP token
- [ ] Intergate with GRPC


