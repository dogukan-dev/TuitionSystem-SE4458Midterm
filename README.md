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

## Authentication

V2 endpoints use JWT Bearer tokens. Include the token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

Get a token by logging in at `/api/v2/login` or register at  `/api/v2/register` (you need to be a student in the system beforehand).
