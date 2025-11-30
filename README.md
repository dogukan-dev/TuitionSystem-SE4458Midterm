# University Tuition Payment System API

## Running the Server

```bash
go run .
```

The server will start on port 8080.

## API Documentation

### Swagger UI (Interactive)
- **Main Swagger UI**: http://localhost:8080/swagger/index.html
- **Alternative Swagger UI**: http://localhost:8080/swagger-ui.html

### API Specification
- **OpenAPI JSON**: http://localhost:8080/api/swagger.json

## API Versions

The API supports two versions:

### V1 (No Authentication Required for most endpoints)
- `/api/v1/health` - Health check
- `/api/v1/register` - Student registration
- `/api/v1/login` - Student login
- `/api/v1/mobile/tuition` - Query tuition (mobile app)
- `/api/v1/banking/tuition` - Query tuition (banking app)
- `/api/v1/banking/pay` - Pay tuition
- `/api/v1/admin/*` - Administrative functions

### V2 (Authentication Required for protected endpoints)
- Same endpoints as V1, but with JWT authentication for admin and some user endpoints
- `/api/v2/health` - Health check

## Authentication

V2 endpoints use JWT Bearer tokens. Include the token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

Get a token by logging in at `/api/v2/login`.