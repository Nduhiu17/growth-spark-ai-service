# Growth Spark AI Service

A Go-based SaaS marketing service with authentication functionality.

## Project Structure

```
growth-spark-ai-service/
├── main.go                 # Application entry point
├── auth/                   # JWT authentication utilities
│   └── jwt.go
├── database/               # Database connection and configuration
│   └── connection.go
├── handlers/               # HTTP request handlers
│   ├── auth.go            # Authentication endpoints
│   └── organization.go    # Organization management
├── middleware/             # HTTP middleware
│   └── auth.go            # Authentication middleware
└── models/                 # Data models
    ├── organization.go
    └── user.go
```

## Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=saas_marketing_db
JWT_SECRET=your-secret-key-here
PORT=8080
```

## API Endpoints

### Public Endpoints (No Authentication Required)

#### Create Organization
```http
POST /api/v1/organizations
Content-Type: application/json

{
  "name": "Example Corp",
  "description": "A sample organization",
  "contactPersonName": "John Doe",
  "contactPersonPhone": "+1234567890",
  "contactPersonEmail": "john@example.com"
}
```

#### User Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### User Registration
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "organizationId": "60f7b3b3b3b3b3b3b3b3b3b3",
  "username": "johndoe",
  "email": "john@example.com",
  "password": "password123",
  "role": "user"
}
```

### Protected Endpoints (Authentication Required)

Include the JWT token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

#### Get User Profile
```http
GET /api/v1/profile
Authorization: Bearer <your-jwt-token>
```

## Authentication Flow

1. **Organization Creation**: Create an organization using the `/organizations` endpoint
2. **User Registration**: Register users with the organization ID using `/auth/register`
3. **Login**: Authenticate users with `/auth/login` to receive a JWT token
4. **Protected Access**: Use the JWT token in the Authorization header for protected endpoints

## User Roles

- `super_admin`: Full access to organization management
- `admin`: Administrative access within the organization
- `user`: Standard user access (default)

## Running the Application

1. Install dependencies:
```bash
go mod tidy
```

2. Set up environment variables in `.env` file

3. Run the application:
```bash
go run main.go
```

The server will start on the port specified in the `PORT` environment variable (default: 8080).

## Security Features

- Password hashing using bcrypt
- JWT token-based authentication
- Role-based access control middleware
- Token expiration (24 hours)
- Protected routes with authentication middleware
