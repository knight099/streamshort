# GitHub Repository Secrets Configuration

This document provides instructions for configuring GitHub repository secrets required for the CI/CD pipeline to deploy StreamShort to AWS.

## Prerequisites

- GitHub repository with admin access
- AWS infrastructure components created (ECR, OIDC, App Runner)
- AWS account ID and service ARNs from previous setup steps

## Required GitHub Secrets

The following secrets must be configured in your GitHub repository for the CI/CD pipeline to function:

### AWS Configuration Secrets

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `AWS_ACCOUNT_ID` | Your AWS account ID (12-digit number) | `123456789012` |
| `AWS_REGION` | Target AWS region for deployment | `us-east-1` |
| `AWS_ROLE_ARN` | ARN of the GitHub Actions IAM role | `arn:aws:iam::123456789012:role/GitHubActionsRole` |
| `ECR_REPOSITORY` | Name of the ECR repository | `streamshort` |
| `APP_RUNNER_SERVICE_ARN` | ARN of the App Runner service | `arn:aws:apprunner:us-east-1:123456789012:service/streamshort-api/...` |

## How to Add Secrets to GitHub Repository

### Method 1: Using GitHub Web Interface

1. Navigate to your GitHub repository
2. Click on **Settings** tab
3. In the left sidebar, click **Secrets and variables** → **Actions**
4. Click **New repository secret**
5. Enter the secret name and value
6. Click **Add secret**
7. Repeat for each required secret

### Method 2: Using GitHub CLI

If you have GitHub CLI installed:

```bash
# Set each secret using gh CLI
gh secret set AWS_ACCOUNT_ID --body "123456789012"
gh secret set AWS_REGION --body "us-east-1"
gh secret set AWS_ROLE_ARN --body "arn:aws:iam::123456789012:role/GitHubActionsRole"
gh secret set ECR_REPOSITORY --body "streamshort"
gh secret set APP_RUNNER_SERVICE_ARN --body "arn:aws:apprunner:us-east-1:123456789012:service/streamshort-api/..."
```

## Getting the Required Values

### AWS Account ID
```bash
aws sts get-caller-identity --query Account --output text
```

### AWS Role ARN
```bash
aws iam get-role --role-name GitHubActionsRole --query 'Role.Arn' --output text
```

### ECR Repository URI
```bash
aws ecr describe-repositories --repository-names streamshort --query 'repositories[0].repositoryUri' --output text
```

### App Runner Service ARN
```bash
aws apprunner list-services --query 'ServiceSummaryList[?ServiceName==`streamshort-api`].ServiceArn' --output text
```

## Secrets Configuration Checklist

Use this checklist to ensure all secrets are properly configured:

- [ ] `AWS_ACCOUNT_ID` - 12-digit AWS account number
- [ ] `AWS_REGION` - Target deployment region (e.g., us-east-1)
- [ ] `AWS_ROLE_ARN` - Complete ARN of GitHub Actions IAM role
- [ ] `ECR_REPOSITORY` - ECR repository name (streamshort)
- [ ] `APP_RUNNER_SERVICE_ARN` - Complete ARN of App Runner service

## Verification

### Verify Secrets Are Set

Using GitHub CLI:
```bash
gh secret list
```

Using GitHub Web Interface:
1. Go to repository **Settings** → **Secrets and variables** → **Actions**
2. Verify all required secrets are listed
3. Check that secret values are not empty (GitHub shows "Updated X time ago")

### Test Secrets in Workflow

Create a simple test workflow to verify secrets are accessible:

```yaml
# .github/workflows/test-secrets.yml
name: Test Secrets
on:
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test AWS Account ID
        run: |
          if [ -z "${{ secrets.AWS_ACCOUNT_ID }}" ]; then
            echo "❌ AWS_ACCOUNT_ID is not set"
            exit 1
          else
            echo "✅ AWS_ACCOUNT_ID is set"
          fi
      
      - name: Test AWS Region
        run: |
          if [ -z "${{ secrets.AWS_REGION }}" ]; then
            echo "❌ AWS_REGION is not set"
            exit 1
          else
            echo "✅ AWS_REGION is set"
          fi
      
      - name: Test AWS Role ARN
        run: |
          if [ -z "${{ secrets.AWS_ROLE_ARN }}" ]; then
            echo "❌ AWS_ROLE_ARN is not set"
            exit 1
          else
            echo "✅ AWS_ROLE_ARN is set"
          fi
      
      - name: Test ECR Repository
        run: |
          if [ -z "${{ secrets.ECR_REPOSITORY }}" ]; then
            echo "❌ ECR_REPOSITORY is not set"
            exit 1
          else
            echo "✅ ECR_REPOSITORY is set"
          fi
      
      - name: Test App Runner Service ARN
        run: |
          if [ -z "${{ secrets.APP_RUNNER_SERVICE_ARN }}" ]; then
            echo "❌ APP_RUNNER_SERVICE_ARN is not set"
            exit 1
          else
            echo "✅ APP_RUNNER_SERVICE_ARN is set"
          fi
```

## Security Best Practices

### Secret Management
- **Never commit secrets to code**: Always use GitHub secrets for sensitive values
- **Use least privilege**: Ensure IAM roles have minimal required permissions
- **Rotate regularly**: Consider rotating AWS credentials periodically
- **Monitor access**: Review GitHub Actions logs for unauthorized access attempts

### Environment-Specific Secrets
For multiple environments (staging, production), consider using environment-specific secrets:
- `PROD_AWS_ACCOUNT_ID`, `STAGING_AWS_ACCOUNT_ID`
- `PROD_APP_RUNNER_SERVICE_ARN`, `STAGING_APP_RUNNER_SERVICE_ARN`

### Secret Naming Convention
- Use UPPERCASE with underscores
- Prefix with service name if needed (e.g., `AWS_`, `ECR_`)
- Be descriptive but concise

## Troubleshooting

### Common Issues

**Secret not found in workflow:**
- Verify secret name matches exactly (case-sensitive)
- Check that secret is set at repository level, not organization level
- Ensure workflow has proper permissions

**Invalid AWS credentials:**
- Verify AWS_ROLE_ARN is correct and role exists
- Check that OIDC provider trust policy allows your repository
- Ensure IAM role has necessary permissions

**ECR access denied:**
- Verify ECR_REPOSITORY name matches actual repository
- Check that IAM role has ECR permissions
- Ensure ECR repository exists in the specified region

## Next Steps

1. Verify all secrets are configured correctly
2. Run the test workflow to validate secret access
3. Proceed to create the GitHub Actions workflow file
4. Test the complete CI/CD pipeline with a sample deployment