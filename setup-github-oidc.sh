#!/bin/bash

# Setup GitHub OIDC for AWS
# Run this with AWS admin credentials

set -e

echo "🔧 Setting up GitHub OIDC Provider for AWS..."

# Step 1: Create OIDC Provider (if it doesn't exist)
echo "📋 Creating GitHub OIDC Provider..."
aws iam create-open-id-connect-provider \
    --url https://token.actions.githubusercontent.com \
    --client-id-list sts.amazonaws.com \
    --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1 \
    --thumbprint-list 1c58a3a8518e8759bf075b76b750d4f2df264fcd 2>/dev/null || echo "OIDC Provider already exists"

# Step 2: Update GitHubActionsRole trust policy
echo "🔐 Updating GitHubActionsRole trust policy..."
aws iam update-assume-role-policy \
    --role-name GitHubActionsRole \
    --policy-document file://github-oidc-trust-policy.json

echo "✅ GitHub OIDC setup complete!"
echo ""
echo "🔍 Verifying setup..."
aws iam get-role --role-name GitHubActionsRole --query 'Role.AssumeRolePolicyDocument'

echo ""
echo "🚀 GitHub Actions should now be able to assume the role!"
echo "   Try running the workflow again."