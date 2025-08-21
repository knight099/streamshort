# Environment Variables Setup

This document explains how to set up environment variables for the StreamShort application.

## Quick Setup

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit the `.env` file with your actual values:
   ```bash
   # Server Configuration
   PORT=8080
   
   # Database Configuration
   DATABASE_URL=postgres://username:password@localhost:5432/streamshort?sslmode=disable
   
   # Migration Configuration
   SKIP_MIGRATIONS=false
   ```

## Environment Variables

### Required Variables

- **PORT**: The port number the server will bind to (default: 8080)
- **DATABASE_URL**: PostgreSQL connection string

### Optional Variables

- **SKIP_MIGRATIONS**: Set to "true" to skip database migrations (default: false)

## For Render Deployment

When deploying to Render, make sure to:

1. Set the `PORT` environment variable in your Render service settings
2. Set the `DATABASE_URL` environment variable with your production database connection string
3. The application will automatically bind to `0.0.0.0:PORT` for deployment compatibility

## Local Development

For local development, you can create a `.env.local` file which will override the `.env` file:

```bash
# .env.local
PORT=3000
DATABASE_URL=postgres://postgres:password@localhost:5432/streamshort_dev?sslmode=disable
```

## Security Notes

- Never commit `.env` files to version control
- Use strong, unique passwords for production databases
- Consider using environment-specific configuration files for different deployment environments
