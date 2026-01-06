# Amazon ECR Repository Setup

This document provides step-by-step instructions for creating and configuring an Amazon ECR repository for the Episodd application.

## Prerequisites

- AWS CLI installed and configured with appropriate permissions
- AWS account with ECR service access

## Create ECR Repository

Create a private ECR repository to store the Episodd Docker images:

```bash
aws ecr create-repository \
    --repository-name episodd \
    --region us-east-1 \
    --image-scanning-configuration scanOnPush=true \
    --encryption-configuration encryptionType=AES256
```

## Configure Lifecycle Policy

Create a lifecycle policy to automatically clean up old images and manage storage costs:

```bash
# Create lifecycle policy JSON file
cat > ecr-lifecycle-policy.json << 'EOF'
{
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
}
EOF

# Apply the lifecycle policy
aws ecr put-lifecycle-policy \
    --repository-name episodd \
    --lifecycle-policy-text file://ecr-lifecycle-policy.json
```

## Verify Repository Creation

Confirm the repository was created successfully:

```bash
aws ecr describe-repositories --repository-names episodd
```

## Get Repository URI

Retrieve the repository URI for use in GitHub Actions:

```bash
aws ecr describe-repositories \
    --repository-names episodd \
    --query 'repositories[0].repositoryUri' \
    --output text
```

Save this URI as it will be needed for the GitHub Actions workflow configuration.

## Repository Configuration Summary

- **Repository Name**: episodd
- **Region**: us-east-1 (adjust as needed)
- **Image Scanning**: Enabled on push
- **Encryption**: AES-256
- **Lifecycle Policy**: Keep last 10 images
- **Tag Mutability**: MUTABLE (allows overwriting 'latest' tag)

## Next Steps

1. Note the repository URI for GitHub Actions configuration
2. Proceed to GitHub OIDC provider setup
3. Configure GitHub repository secrets with ECR details