# AWS Deployment Guide

This guide provides step-by-step instructions for setting up the complete CI/CD pipeline to deploy Episodd to AWS using GitHub Actions, Amazon ECR, and AWS App Runner.

## Prerequisites

- AWS CLI installed and configured with appropriate permissions
- GitHub repository with admin access
- Docker installed locally (for testing)

## Step 1: Create Amazon ECR Repository

First, create a private ECR repository to store your Docker images:

```bash
# Set your variables
export AWS_REGION="us-east-1"  # Change to your preferred region
export ECR_REPOSITORY="episodd"

# Create ECR repository
aws ecr create-repository \
    --repository-name $ECR_REPOSITORY \
    --region $AWS_REGION \
    --image-scanning-configuration scanOnPush=true \
    --encryption-configuration encryptionType=AES256

# Create lifecycle policy to keep only the last 10 images
aws ecr put-lifecycle-policy \
    --repository-name $ECR_REPOSITORY \
    --region $AWS_REGION \
    --lifecycle-policy-text '{
        "rules": [
            {
                "rulePriority": 1,
                "description": "Keep last 10 images",
                "selection": {
                    "tagStatus": "any",
                    "countType": "imageCountMoreThan",
                    "countNumber": 10
                },
                "action": {
                    "type": "expire"
                }
            }
        ]
    }'
```

## Step 2: Set Up GitHub OIDC Provider

Create an OIDC identity provider to allow GitHub Actions to authenticate with AWS without storing credentials:

```bash
# Create OIDC identity provider
aws iam create-open-id-connect-provider \
    --url https://token.actions.githubusercontent.com \
    --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1 \
    --client-id-list sts.amazonaws.com
```

## Step 3: Create IAM Role for GitHub Actions

Create an IAM role that GitHub Actions will assume:

```bash
# Set your variables
export GITHUB_ORG="your-github-username"  # Replace with your GitHub username/org
export GITHUB_REPO="episodd"          # Replace with your repository name
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Create trust policy file
cat > github-actions-trust-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
                },
                "StringLike": {
                    "token.actions.githubusercontent.com:sub": "repo:${GITHUB_ORG}/${GITHUB_REPO}:ref:refs/heads/main"
                }
            }
        }
    ]
}
EOF

# Create IAM role
aws iam create-role \
    --role-name GitHubActionsRole \
    --assume-role-policy-document file://github-actions-trust-policy.json

# Create permissions policy file
cat > github-actions-permissions-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecr:GetAuthorizationToken",
                "ecr:BatchCheckLayerAvailability",
                "ecr:GetDownloadUrlForLayer",
                "ecr:BatchGetImage",
                "ecr:InitiateLayerUpload",
                "ecr:UploadLayerPart",
                "ecr:CompleteLayerUpload",
                "ecr:PutImage"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "apprunner:UpdateService",
                "apprunner:DescribeService"
            ],
            "Resource": "*"
        }
    ]
}
EOF

# Attach permissions policy to role
aws iam put-role-policy \
    --role-name GitHubActionsRole \
    --policy-name GitHubActionsPermissions \
    --policy-document file://github-actions-permissions-policy.json

# Clean up policy files
rm github-actions-trust-policy.json github-actions-permissions-policy.json
```

## Step 4: Create AWS App Runner Service

Create the App Runner service that will host your application:

```bash
# Create App Runner service configuration file
cat > apprunner-service.json << EOF
{
    "ServiceName": "episodd-api",
    "SourceConfiguration": {
        "ImageRepository": {
            "ImageIdentifier": "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:latest",
            "ImageConfiguration": {
                "Port": "8080",
                "RuntimeEnvironmentVariables": {
                    "PORT": "8080"
                }
            },
            "ImageRepositoryType": "ECR"
        },
        "AutoDeploymentsEnabled": false
    },
    "InstanceConfiguration": {
        "Cpu": "1 vCPU",
        "Memory": "2 GB"
    },
    "AutoScalingConfigurationArn": "",
    "HealthCheckConfiguration": {
        "Protocol": "HTTP",
        "Path": "/health",
        "Interval": 10,
        "Timeout": 5,
        "HealthyThreshold": 1,
        "UnhealthyThreshold": 5
    }
}
EOF

# Create App Runner service
aws apprunner create-service \
    --cli-input-json file://apprunner-service.json \
    --region $AWS_REGION

# Clean up configuration file
rm apprunner-service.json

# Get the service ARN (you'll need this for GitHub secrets)
aws apprunner list-services \
    --region $AWS_REGION \
    --query 'ServiceSummaryList[?ServiceName==`episodd-api`].ServiceArn' \
    --output text
```

## Step 5: Configure GitHub Repository Secrets

Add the following secrets to your GitHub repository (Settings → Secrets and variables → Actions):

1. **AWS_ACCOUNT_ID**: Your AWS account ID
   ```bash
   echo $AWS_ACCOUNT_ID
   ```

2. **AWS_REGION**: Your AWS region (e.g., `us-east-1`)
   ```bash
   echo $AWS_REGION
   ```

3. **ECR_REPOSITORY**: Your ECR repository name
   ```bash
   echo $ECR_REPOSITORY
   ```

4. **APP_RUNNER_SERVICE_ARN**: The ARN of your App Runner service
   ```bash
   aws apprunner list-services \
       --region $AWS_REGION \
       --query 'ServiceSummaryList[?ServiceName==`episodd-api`].ServiceArn' \
       --output text
   ```

5. **IAM_ROLE_ARN**: The ARN of the IAM role for GitHub Actions
   ```bash
   aws iam get-role \
       --role-name GitHubActionsRole \
       --query 'Role.Arn' \
       --output text
   ```

## Step 6: Test the Setup

1. **Test Docker build locally:**
   ```bash
   docker build -t episodd-test .
   docker run -p 8080:8080 episodd-test
   ```

2. **Test health endpoint:**
   ```bash
   curl http://localhost:8080/health
   ```

3. **Push to main branch** to trigger the GitHub Actions workflow

## Troubleshooting

### Common Issues and Solutions

#### 1. OIDC Authentication Failures

**Error:** `Error: Could not assume role with OIDC`

**Solutions:**
- Verify the OIDC provider exists:
  ```bash
  aws iam list-open-id-connect-providers
  ```
- Check the trust policy allows your repository:
  ```bash
  aws iam get-role --role-name GitHubActionsRole --query 'Role.AssumeRolePolicyDocument'
  ```
- Ensure the repository name and organization match exactly in the trust policy

#### 2. ECR Push Permission Denied

**Error:** `denied: User: arn:aws:sts::ACCOUNT:assumed-role/GitHubActionsRole is not authorized to perform: ecr:InitiateLayerUpload`

**Solutions:**
- Verify ECR permissions in the IAM role:
  ```bash
  aws iam get-role-policy --role-name GitHubActionsRole --policy-name GitHubActionsPermissions
  ```
- Ensure the ECR repository exists in the correct region
- Check that `ecr:GetAuthorizationToken` permission is included

#### 3. App Runner Deployment Failures

**Error:** `Service update failed`

**Solutions:**
- Check App Runner service logs in AWS Console
- Verify the Docker image runs locally
- Ensure health check endpoint `/health` is accessible
- Check environment variables are set correctly

#### 4. Docker Build Failures

**Error:** `failed to solve: process "/bin/sh -c go build" didn't complete successfully`

**Solutions:**
- Verify `go.mod` and `go.sum` are committed to the repository
- Check for missing dependencies:
  ```bash
  go mod tidy
  ```
- Ensure the build context includes all necessary files (check `.dockerignore`)

#### 5. Health Check Failures

**Error:** `Health check failed`

**Solutions:**
- Verify your application exposes a `/health` endpoint
- Check the endpoint returns HTTP 200 status
- Ensure the application binds to `0.0.0.0:8080`, not `localhost:8080`
- Test locally:
  ```bash
  curl -v http://localhost:8080/health
  ```

#### 6. GitHub Actions Workflow Syntax Errors

**Error:** `Invalid workflow file`

**Solutions:**
- Validate YAML syntax using online validators
- Check indentation (use spaces, not tabs)
- Verify job dependencies are correct
- Use `actionlint` for GitHub Actions-specific validation:
  ```bash
  # Install actionlint
  go install github.com/rhymond/actionlint/cmd/actionlint@latest
  
  # Validate workflow
  actionlint .github/workflows/deploy.yml
  ```

### Debugging Commands

**Check ECR repository:**
```bash
aws ecr describe-repositories --repository-names $ECR_REPOSITORY --region $AWS_REGION
```

**Check App Runner service status:**
```bash
aws apprunner describe-service --service-arn $APP_RUNNER_SERVICE_ARN --region $AWS_REGION
```

**List recent App Runner operations:**
```bash
aws apprunner list-operations --service-arn $APP_RUNNER_SERVICE_ARN --region $AWS_REGION
```

**Check IAM role:**
```bash
aws iam get-role --role-name GitHubActionsRole
```

### Getting Help

- **AWS Documentation:** [App Runner Developer Guide](https://docs.aws.amazon.com/apprunner/)
- **GitHub Actions:** [GitHub Actions Documentation](https://docs.github.com/en/actions)
- **Docker:** [Docker Documentation](https://docs.docker.com/)

For additional support, check the GitHub Actions workflow logs and AWS CloudTrail for detailed error messages.