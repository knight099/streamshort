#!/bin/bash

# StreamShort Environment Setup Script

echo "🚀 Setting up StreamShort environment variables..."

# Check if .env already exists
if [ -f ".env" ]; then
    echo "⚠️  .env file already exists. Backing up to .env.backup"
    cp .env .env.backup
fi

# Create .env file
cat > .env << EOF
# Server Configuration
PORT=8080

# Database Configuration
DATABASE_URL=postgres://postgres:password@localhost:5432/streamshort?sslmode=disable

# Migration Configuration
SKIP_MIGRATIONS=false

# Add other environment variables as needed
# JWT_SECRET=your_jwt_secret_here
# AWS_ACCESS_KEY_ID=your_aws_access_key
# AWS_SECRET_ACCESS_KEY=your_aws_secret_key
# AWS_REGION=us-east-1
# AWS_S3_BUCKET=your_s3_bucket_name
EOF

echo "✅ .env file created successfully!"
echo ""
echo "📝 Please edit the .env file with your actual values:"
echo "   - Update DATABASE_URL with your database connection string"
echo "   - Set PORT if you want to use a different port"
echo "   - Configure other variables as needed"
echo ""
echo "🔒 Remember: Never commit .env files to version control!"
echo ""
echo "📖 See ENV_SETUP.md for more details"
