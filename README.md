# Go2Bank Bankend 

## API Endpoints
- **POST** `/api/users` : Register a new user
- **POST** `/api/users/login` : Login from your username and password
- **GET** `/api/tokens/renew_access` : Renew your access token
- **GET** `/api/users/logout` : Logout and redirect to home page 

> [!NOTE] 
> Following API Endpoints will be passed through Paseto  and CSRF Authentication Middleware
- **GET** `/api/auth/users/me` : Get a user information
- **POST** `/api/auth/accounts` : create a new account by a user
- **GET** `/api/auth/accounts/:id` : Get a account information
- **GET** `/api/auth/accounts` : List all accounts

> [!NOTE]
>  Add RBAC Authorization, This layer is applied in addition to above middleware protections.
- **PATCH** `/api/auth/users/update` : Update user information

>[!NOTE]
> Add Rate Limiting and OTP Verified Middleware, This layer is applied in addition to above middleware protections.
- **GET** `/api/auth/transfers/sendOTP`: Enqueue OTP verification task into message queue
- **POST** `/api/auth/transfers` : create a new transfer between two accounts

## DB Diagram
![db](https://github.com/RobertChienShiba/Go2Bank/blob/main/DB.png)

## A rate limit of 1000 requests per 30 seconds, with a load test configured to simulate 50 requests per second
![load](https://github.com/RobertChienShiba/Go2Bank/blob/main/load-testing.png)

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
- Google OAuth2 
- CSRF Protection 
- CORS

## TODO
- [x] Secure `Transfers` endpoints with time-based OTP
- [x] Implement an asynchronous worker to deliver emails
- [x] Build a robust API rate limiting middleware with a sliding window logging algorithm
- [x] Store Refresh Tokens in **HttpOnly cookies** and **Redis** for better user experience and instant revocation
- [x] Implemented CSRF protection and CORS policies 
- [x] Google OAuth2 Integration
- [ ] Improve Testing coverage (up to at least 80%)
- [ ] Intergate with gRPC API
- [ ] Use SQS as the message queue for OTP requests
- [ ] Create a Lambda function to listen to SQS and trigger SES to complete OTP email delivery

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
- [go-querystring](https://github.com/google/go-querystring)
- [CSRF Protection](https://studygolang.com/articles/35927?fr=sidebar)
- [Google OAuth2](https://codevoweb.com/how-to-implement-google-oauth2-in-golang/)

