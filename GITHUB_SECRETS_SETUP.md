# GitHub Secrets Setup for Automated Deployment

## Required Secrets

Add these secrets to your GitHub repository:
**Settings → Secrets and variables → Actions → New repository secret**

### AWS Configuration
```
AWS_ACCOUNT_ID = 
AWS_REGION = -1
AWS_ROLE_ARN = 
ECR_REPOSITORY = 
```

### Application Environment Variables
```
DATABASE_URL = 
JWT_SECRET = 
FIREBASE_API_KEY = 
FIREBASE_PROJECT_ID = -49ffe
RAZORPAY_KEY_ID = 
RAZORPAY_KEY_SECRET = 
AWS_ACCESS_KEY_ID = 
AWS_SECRET_ACCESS_KEY = 
AWS_S3_BUCKET = 
CORS_ALLOWED_ORIGINS = 
```

## App Runner Service ARN (Add after service is created successfully)
```
APPRUNNER_SERVICE_ARN = 
```

## How to Add Secrets:
1. Go to your GitHub repository
2. Click Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Add each secret with the exact name and value above
5. Click "Add secret"

## Deployment Trigger:
- Push to `main` branch will trigger automatic deployment
- The workflow will build, push to ECR, and update App Runner service