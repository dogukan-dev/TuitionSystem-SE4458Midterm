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

## Design,Assumptions and Issues
I can say as a whole it was a beneficial project in terms of remembering the basics of api design
and combining common concepts together.I had the most issues when trying to bridge connection between
middlewares and handlers so api gateway


## ER Diagram Analysis

### 1\. Entities 

- **Student** (Attributes: `student_no` - **Primary Key**, `balance`, `daily_payment_limit`)
- **Account** (Attributes: `account_no` - **Primary Key**, `hashed_password`, `student_no` - **Foreign Key/Unique**)
- **Tuition** (Attributes: `tuition_id` - **Primary Key**, `term`, `tuition_total`, `student_no` - **Foreign Key**)

### 2\. Relationships 

- **Student** and **Account**: The `student_no` in the `account` table is both a **Foreign Key** referencing `student` and is declared as **UNIQUE**. This establishes a **one-to-one (1:1)** relationship:
	- One Student has one Account.
	- One Account belongs to one Student.
- **Student** and **Tuition**: The `student_no` in the `tuition` table is a **Foreign Key** but is **not** unique (since a student can have tuition records for multiple terms). This establishes a **one-to-many (1:N)** relationship:
	- **One** Student has many Tuition records.
	- **One** Tuition record belongs TO one Student.
