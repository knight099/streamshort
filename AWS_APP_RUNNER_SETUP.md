# AWS App Runner Service Setup

This document provides instructions for creating and configuring an AWS App Runner service to host the Episodd application.

## Prerequisites

- AWS CLI installed and configured
- ECR repository created with Episodd image
- IAM role for App Runner service access
- Environment variables and secrets configured

## Step 1: Create App Runner Service Access Role

First, create an IAM role that App Runner can use to access ECR:

```bash
# Create trust policy for App Runner
cat > apprunner-trust-policy.json << 'EOF'
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "build.apprunner.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
EOF

# Create the role
aws iam create-role \
    --role-name AppRunnerECRAccessRole \
    --assume-role-policy-document file://apprunner-trust-policy.json

# Attach the managed policy for ECR access
aws iam attach-role-policy \
    --role-name AppRunnerECRAccessRole \
    --policy-arn arn:aws:iam::aws:policy/service-role/AWSAppRunnerServicePolicyForECRAccess
```

## Step 2: Create App Runner Service Configuration

Create the service configuration file:

```bash
cat > apprunner-service-config.json << 'EOF'
{
    "ServiceName": "episodd-api",
    "SourceConfiguration": {
        "ImageRepository": {
            "ImageIdentifier": "YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/episodd:latest",
            "ImageConfiguration": {
                "Port": "8080",
                "RuntimeEnvironmentVariables": {
                    "PORT": "8080",
                    "ENV": "production"
                }
            },
            "ImageRepositoryType": "ECR"
        },
        "AccessRoleArn": "arn:aws:iam::YOUR_ACCOUNT_ID:role/AppRunnerECRAccessRole"
    },
    "InstanceConfiguration": {
        "Cpu": "1 vCPU",
        "Memory": "2 GB"
    },
    "HealthCheckConfiguration": {
        "Protocol": "HTTP",
        "Path": "/health",
        "Interval": 10,
        "Timeout": 5,
        "HealthyThreshold": 1,
        "UnhealthyThreshold": 5
    },
    "AutoScalingConfigurationArn": "arn:aws:apprunner:YOUR_REGION:YOUR_ACCOUNT_ID:autoscalingconfiguration/DefaultConfiguration/1/00000000000000000000000000000001"
}
EOF
```

## Step 3: Create Auto Scaling Configuration

Create a custom auto-scaling configuration:

```bash
aws apprunner create-auto-scaling-configuration \
    --auto-scaling-configuration-name "episodd-autoscaling" \
    --max-concurrency 100 \
    --min-size 1 \
    --max-size 10
```

## Step 4: Create the App Runner Service

Create the App Runner service:

```bash
# Update the service config with the auto-scaling ARN from step 3
# Then create the service
aws apprunner create-service \
    --cli-input-json file://apprunner-service-config.json
```

## Step 5: Alternative - Direct CLI Command

Alternatively, create the service directly with CLI parameters:

```bash
aws apprunner create-service \
    --service-name "episodd-api" \
    --source-configuration '{
        "ImageRepository": {
            "ImageIdentifier": "YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/episodd:latest",
            "ImageConfiguration": {
                "Port": "8080",
                "RuntimeEnvironmentVariables": {
                    "PORT": "8080",
                    "ENV": "production"
                }
            },
            "ImageRepositoryType": "ECR"
        },
        "AccessRoleArn": "arn:aws:iam::YOUR_ACCOUNT_ID:role/AppRunnerECRAccessRole"
    }' \
    --instance-configuration '{
        "Cpu": "1 vCPU",
        "Memory": "2 GB"
    }' \
    --health-check-configuration '{
        "Protocol": "HTTP",
        "Path": "/health",
        "Interval": 10,
        "Timeout": 5,
        "HealthyThreshold": 1,
        "UnhealthyThreshold": 5
    }'
```

## Step 6: Configure Environment Variables (Optional)

If you need to add environment variables after service creation:

```bash
# Get current service configuration
aws apprunner describe-service \
    --service-arn "YOUR_SERVICE_ARN" \
    --query 'Service.SourceConfiguration.ImageRepository.ImageConfiguration'

# Update service with new environment variables
aws apprunner update-service \
    --service-arn "YOUR_SERVICE_ARN" \
    --source-configuration '{
        "ImageRepository": {
            "ImageIdentifier": "YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/episodd:latest",
            "ImageConfiguration": {
                "Port": "8080",
                "RuntimeEnvironmentVariables": {
                    "PORT": "8080",
                    "ENV": "production",
                    "DATABASE_URL": "your-database-url"
                }
            },
            "ImageRepositoryType": "ECR"
        }
    }'
```

## Service Configuration Summary

- **Service Name**: episodd-api
- **Port**: 8080
- **CPU**: 1 vCPU
- **Memory**: 2 GB
- **Health Check**: HTTP GET /health every 10 seconds
- **Auto Scaling**: 1-10 instances, max 100 concurrent requests per instance
- **Deployment**: Automatic when new ECR image is pushed

## Get Service Information

Retrieve service details:

```bash
# Get service ARN and URL
aws apprunner list-services --query 'ServiceSummaryList[?ServiceName==`episodd-api`]'

# Get detailed service information
aws apprunner describe-service --service-arn "YOUR_SERVICE_ARN"

# Get service URL
aws apprunner describe-service \
    --service-arn "YOUR_SERVICE_ARN" \
    --query 'Service.ServiceUrl' \
    --output text
```

## Configuration Variables to Replace

Before running the commands, replace these placeholders:

- `YOUR_ACCOUNT_ID`: Your AWS account ID
- `YOUR_REGION`: Your target AWS region (e.g., us-east-1)
- `YOUR_SERVICE_ARN`: The ARN of your App Runner service (obtained after creation)

## Monitoring and Logs

App Runner automatically provides:
- CloudWatch logs for application output
- CloudWatch metrics for requests, response times, CPU, memory
- Health check monitoring
- Automatic scaling based on traffic

Access logs via:
```bash
aws logs describe-log-groups --log-group-name-prefix "/aws/apprunner/episodd-api"
```

## Next Steps

1. Note the service ARN for GitHub Actions configuration
2. Test the service URL to ensure it's responding
3. Configure GitHub repository secrets with service details
4. Set up the GitHub Actions workflow to deploy to this service