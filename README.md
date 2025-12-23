# StreamShort

[![Deploy to AWS](https://github.com/YOUR_USERNAME/streamshort/actions/workflows/deploy.yml/badge.svg)](https://github.com/YOUR_USERNAME/streamshort/actions/workflows/deploy.yml)

A Go-based backend service for short-form video streaming with creator monetization.

## Deployment

This application is automatically deployed to AWS App Runner using GitHub Actions. Every push to the `main` branch triggers a complete CI/CD pipeline that builds, tests, and deploys the application.

### Quick Start

1. **Fork this repository**
2. **Set up AWS infrastructure** - Follow the [AWS Deployment Guide](./AWS_DEPLOYMENT.md)
3. **Configure GitHub secrets** - Add required AWS credentials to your repository
4. **Push to main branch** - Deployment happens automatically

### Deployment Status

- **Production**: Deployed automatically from `main` branch
- **Health Check**: Available at `/health` endpoint
- **Monitoring**: AWS App Runner provides built-in monitoring and auto-scaling

For detailed setup instructions, troubleshooting, and configuration options, see the [AWS Deployment Guide](./AWS_DEPLOYMENT.md).


### Todo

### Authentication:

- POST /auth/signup (phone/email)

- POST /auth/login

- POST /auth/refresh

- POST /auth/logout

### User:

- GET /me

- GET /users/{id}/subscriptions

### Creator:

- POST /creators/onboard (KYC fields)

- GET /creators/{id}/dashboard (analytics)

- POST /creators/{id}/payout-request

### Content:

- POST /content/series (create)

- POST /content/series/{id}/episodes (create metadata)

- GET /content/series (list, filters: language, category)

- GET /content/series/{id}

- GET /content/episodes/{id}/manifest (returns signed HLS URL if authorized)

- POST /content/upload-url (generate signed URL for upload)


### Payment:

- POST /payments/create-subscription

- POST /payments/webhook (Razorpay callbacks)

- GET /payments/{id}/status

### Interaction:

- POST /episodes/{id}/like

-POST /episodes/{id}/rating

- GET /episodes/{id}/comments

- POST /episodes/{id}/comments

### Admin:

- GET /admin/uploads/pending

- POST /admin/approve-content
