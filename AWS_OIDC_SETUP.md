# GitHub OIDC Provider Setup for AWS

This document provides instructions for setting up GitHub OIDC (OpenID Connect) authentication with AWS, allowing GitHub Actions to securely access AWS services without storing long-lived credentials.

## Prerequisites

- AWS CLI installed and configured with administrative permissions
- GitHub repository with Actions enabled
- AWS account ID and target region

## Step 1: Create GitHub OIDC Identity Provider

Create the OIDC identity provider in AWS IAM:

```bash
aws iam create-open-id-connect-provider \
    --url https://token.actions.githubusercontent.com \
    --client-id-list sts.amazonaws.com \
    --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1 \
    --thumbprint-list 1c58a3a8518e8759bf075b76b750d4f2df264fcd
```

## Step 2: Create IAM Trust Policy

Create the trust policy that allows GitHub Actions to assume the IAM role:

```bash
# Replace YOUR_ACCOUNT_ID, YOUR_GITHUB_USERNAME, and YOUR_REPO_NAME
cat > github-oidc-trust-policy.json << 'EOF'
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::YOUR_ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
                },
                "StringLike": {
                    "token.actions.githubusercontent.com:sub": "repo:YOUR_GITHUB_USERNAME/YOUR_REPO_NAME:ref:refs/heads/main"
                }
            }
        }
    ]
}
EOF
```

## Step 3: Create IAM Permissions Policy

Create the permissions policy that grants necessary AWS service access:

```bash
cat > github-actions-permissions-policy.json << 'EOF'
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
            "Resource": "arn:aws:apprunner:*:YOUR_ACCOUNT_ID:service/streamshort-api/*"
        }
    ]
}
EOF
```

## Step 4: Create IAM Role

Create the IAM role that GitHub Actions will assume:

```bash
# Create the role with trust policy
aws iam create-role \
    --role-name GitHubActionsRole \
    --assume-role-policy-document file://github-oidc-trust-policy.json \
    --description "Role for GitHub Actions to deploy StreamShort"

# Create the permissions policy
aws iam create-policy \
    --policy-name GitHubActionsPolicy \
    --policy-document file://github-actions-permissions-policy.json \
    --description "Permissions for GitHub Actions deployment"

# Attach the policy to the role
aws iam attach-role-policy \
    --role-name GitHubActionsRole \
    --policy-arn arn:aws:iam::YOUR_ACCOUNT_ID:policy/GitHubActionsPolicy
```

## Step 5: Get Role ARN

Retrieve the role ARN for use in GitHub Actions:

```bash
aws iam get-role \
    --role-name GitHubActionsRole \
    --query 'Role.Arn' \
    --output text
```

## Configuration Variables to Replace

Before running the commands, replace these placeholders:

- `YOUR_ACCOUNT_ID`: Your AWS account ID (12-digit number)
- `YOUR_GITHUB_USERNAME`: Your GitHub username or organization
- `YOUR_REPO_NAME`: Your repository name (e.g., "streamshort-backend")

## Verification

Verify the OIDC provider was created:

```bash
aws iam list-open-id-connect-providers
```

Verify the role was created:

```bash
aws iam get-role --role-name GitHubActionsRole
```

## Security Notes

- The trust policy restricts access to only the main branch of your specific repository
- The permissions policy follows the principle of least privilege
- No long-lived AWS credentials are stored in GitHub
- Temporary credentials are automatically rotated by AWS STS

## Next Steps

1. Save the Role ARN for GitHub repository secrets configuration
2. Proceed to App Runner service creation
3. Configure GitHub repository secrets with the Role ARN