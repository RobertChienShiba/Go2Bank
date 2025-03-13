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

## DB Diagram
![db](https://imgur.com/55u6nUY.png)

## Tech Stack
- Gin
- Sqlc
- PostgreSQL
- Redis
- Paseto Tokens
- Github Actions
- Docker
- CronJob
- Time-based OTP
- API rate limited (Load testing through [vegeta](https://github.com/tsenart/vegeta))

## TODO
- [x] Secure `Transfers` endpoints with time-based OTP token
- [x] Implement an asynchronous worker to deliver emails
- [x] Build a robust API rate limiting middleware with a sliding window logging algorithm
- [x] Store Refresh Tokens in **HttpOnly cookies** and **Redis** for better user experience and instant revocation
- [ ] Improve Testing coverage (up to at least 80%)
- [ ] Intergate with gRPC API
- [ ] Use SQS as the message queue for OTP requests
- [ ] Create a Lambda function to listen to SQS and trigger AWS SES to complete OTP email delivery

## Reference
- [TechSchool](https://www.youtube.com/playlist?list=PLy_6D98if3ULEtXtNSY_2qN21VCKgoQAE)
- [golang-migrate](https://github.com/golang-migrate/migrate/tree/master?tab=readme-ov-file)
- [Gin](https://github.com/gin-gonic/gin/blob/master/docs/doc.md#model-binding-and-validation)
- [Requests Validator](https://github.com/go-playground/validator?tab=readme-ov-file)
- [Sqlc](https://docs.sqlc.dev/en/latest/reference/config.html#gen)
- [golang-mock](https://github.com/golang/mock)
- [go-redis](https://github.com/redis/go-redis)
- [cronjob in alpine image](https://stackoverflow.com/questions/37458287/how-to-run-a-cron-job-inside-a-docker-container)
- [Golang One-Time Password](https://github.com/xlzd/gotp)
- [Rate limiting algorithm](https://medium.com/@m-elbably/rate-limiting-the-sliding-window-algorithm-daa1d91e6196)