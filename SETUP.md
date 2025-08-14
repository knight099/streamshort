# Streamshort API Setup Guide

This guide will help you set up the Streamshort API with Neon database integration.

## Prerequisites

- Go 1.24.5 or later
- Neon database account (or any PostgreSQL database)

## Database Setup

### Option 1: Neon Database (Recommended)

1. Sign up for a free account at [Neon](https://neon.tech)
2. Create a new project
3. Copy the connection string from your project dashboard
4. Set the `DATABASE_URL` environment variable:

```bash
export DATABASE_URL="postgres://username:password@your-neon-host:5432/streamshort?sslmode=require"
```

### Option 2: Local PostgreSQL

If you prefer to use a local PostgreSQL database:

```bash
export DATABASE_URL="postgres://postgres:password@localhost:5432/streamshort?sslmode=disable"
```

## Environment Variables

Create a `.env` file in the project root (or set environment variables):

```bash
# Database Configuration
DATABASE_URL=your-neon-connection-string

# JWT Secret (change in production)
JWT_SECRET=your-secret-key-change-in-production

# Server Configuration
PORT=8080
```

## Installation

1. Install dependencies:
```bash
go mod tidy
```

2. Run the server:
```bash
go run main.go
```

The server will start on port 8080 and automatically create the necessary database tables.

## API Endpoints

### Public Endpoints

- `GET /` - Hello World
- `GET /health` - Health check
- `POST /auth/otp/send` - Send OTP
- `POST /auth/otp/verify` - Verify OTP and get tokens
- `POST /auth/refresh` - Refresh access token

### Protected Endpoints

- `GET /api/profile` - User profile (requires Bearer token)

## Testing the API

### 1. Send OTP
```bash
curl -X POST http://localhost:8080/auth/otp/send \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919876543210"}'
```

### 2. Verify OTP
```bash
curl -X POST http://localhost:8080/auth/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919876543210", "otp": "123456"}'
```

### 3. Access Protected Endpoint
```bash
curl -X GET http://localhost:8080/api/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Notes

- OTPs are currently logged to console for development purposes
- In production, integrate with an SMS service like Twilio or AWS SNS
- Change the JWT secret in production
- The database will be automatically migrated on startup
