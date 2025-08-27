# Subscription API Implementation Guide

## 🎯 Overview

The Subscription API provides comprehensive functionality for managing user subscriptions to content (series and creators) in the Streamshort platform. This implementation covers subscription creation, management, and access control for paid content.

## 🏗️ Architecture

### Models
- **Subscription**: User's subscription to specific content
- **SubscriptionPlan**: Available subscription plans with pricing and duration

### Key Features
- **Content Access Control**: Check subscription status before allowing access to paid content
- **Subscription Management**: Create, cancel, renew, and check subscription status
- **Plan Management**: Multiple subscription plans with different durations and pricing
- **Automatic Expiry**: Handle subscription expiration and status updates

## 📋 API Endpoints

### 1. Create Subscription
```
POST /api/payments/create-subscription
```

**Request Body:**
```json
{
  "target_type": "series",
  "target_id": "series_123",
  "plan_id": "plan_basic_monthly",
  "auto_renew": true
}
```

**Response:**
```json
{
  "subscription_id": "sub_9f35a4e6",
  "status": "active",
  "plan_id": "plan_basic_monthly",
  "start_date": "2025-01-20T10:00:00Z",
  "end_date": "2025-02-19T10:00:00Z",
  "next_billing": "2025-02-19T10:00:00Z",
  "checkout_url": "https://checkout.razorpay.com/v1/checkout.js?subscription_id=sub_9f35a4e6"
}
```

### 2. Get User Subscriptions
```
GET /api/subscriptions?page=1&per_page=20
```

**Response:**
```json
{
  "total": 2,
  "page": 1,
  "per_page": 20,
  "total_pages": 1,
  "subscriptions": [
    {
      "id": "sub_9f35a4e6",
      "user_id": "user_123",
      "target_type": "series",
      "target_id": "series_123",
      "status": "active",
      "start_date": "2025-01-20T10:00:00Z",
      "end_date": "2025-02-19T10:00:00Z",
      "auto_renew": true,
      "plan_id": "plan_basic_monthly",
      "amount": 99.0,
      "currency": "INR"
    }
  ]
}
```

### 3. Get Specific Subscription
```
GET /api/subscriptions/{id}
```

**Response:**
```json
{
  "id": "sub_9f35a4e6",
  "user_id": "user_123",
  "target_type": "series",
  "target_id": "series_123",
  "status": "active",
  "start_date": "2025-01-20T10:00:00Z",
  "end_date": "2025-02-19T10:00:00Z",
  "auto_renew": true,
  "plan_id": "plan_basic_monthly",
  "amount": 99.0,
  "currency": "INR"
}
```

### 4. Cancel Subscription
```
POST /api/subscriptions/{id}/cancel
```

**Response:**
```json
{
  "message": "Subscription cancelled successfully",
  "subscription_id": "sub_9f35a4e6",
  "status": "cancelled"
}
```

### 5. Renew Subscription
```
POST /api/subscriptions/{id}/renew
```

**Response:**
```json
{
  "message": "Subscription renewed successfully",
  "subscription_id": "sub_9f35a4e6",
  "status": "active",
  "new_end_date": "2025-03-21T10:00:00Z"
}
```

### 6. Check Subscription Status
```
GET /api/subscriptions/check?target_type=series&target_id=series_123
```

**Response:**
```json
{
  "has_access": true,
  "target_type": "series",
  "target_id": "series_123",
  "subscription_details": {
    "id": "sub_9f35a4e6",
    "status": "active",
    "end_date": "2025-02-19T10:00:00Z"
  }
}
```

### 7. Get Available Plans
```
GET /api/subscriptions/plans
```

**Response:**
```json
[
  {
    "id": "plan_basic_monthly",
    "name": "Basic Monthly",
    "description": "Access to one series for 30 days",
    "duration": 30,
    "amount": 99.0,
    "currency": "INR",
    "is_active": true
  },
  {
    "id": "plan_premium_monthly",
    "name": "Premium Monthly",
    "description": "Access to all series from one creator for 30 days",
    "duration": 30,
    "amount": 299.0,
    "currency": "INR",
    "is_active": true
  }
]
```

## 🗄️ Database Schema

### Subscriptions Table
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('series', 'creator')),
    target_id UUID NOT NULL,
    razorpay_subscription_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'expired')),
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    auto_renew BOOLEAN DEFAULT true,
    plan_id VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'INR',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_target ON subscriptions(target_type, target_id);
```

### Subscription Plans Table
```sql
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    duration INTEGER NOT NULL, -- Duration in days
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'INR',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## 🔐 Access Control

### Content Access Logic
The system automatically checks subscription status when users try to access paid content:

1. **Free Content**: Always accessible to authenticated users
2. **Paid Content**: Requires active subscription
3. **Subscription Check**: Validates subscription status and expiration

### Episode Manifest Access
```go
// Check if user has access to this content based on series pricing
if episode.Series.PriceType != "free" {
    // Check if user has an active subscription to this series
    var subscription models.Subscription
    err := h.db.Where("user_id = ? AND target_type = ? AND target_id = ? AND status = ?", 
        userID, "series", episode.Series.ID, "active").
        First(&subscription).Error
    
    if err != nil {
        http.Error(w, "Subscription required to access this content", http.StatusForbidden)
        return
    }
    
    // Check if subscription is still active (not expired)
    if subscription.IsExpired() {
        h.db.Model(&subscription).Update("status", "expired")
        http.Error(w, "Subscription has expired", http.StatusForbidden)
        return
    }
}
```

## 🔄 Workflow

### 1. Subscription Creation Flow
```
User Request → Validate Target → Check Existing → Create Record → Return Checkout URL
```

### 2. Content Access Flow
```
Content Request → Check Pricing → Validate Subscription → Check Expiry → Grant Access
```

### 3. Subscription Management Flow
```
User Action → Verify Ownership → Update Status → Return Confirmation
```

## 🚀 Testing

### Manual Testing
Use the provided test script:
```bash
./test_subscription_api.sh
```

### API Testing with curl

#### Create Subscription
```bash
curl -X POST http://localhost:8080/api/payments/create-subscription \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "series",
    "target_id": "series_123",
    "plan_id": "plan_basic_monthly",
    "auto_renew": true
  }'
```

#### Check Subscription Status
```bash
curl -X GET "http://localhost:8080/api/subscriptions/check?target_type=series&target_id=series_123" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### Get User Subscriptions
```bash
curl -X GET "http://localhost:8080/api/subscriptions?page=1&per_page=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 🔧 Configuration

### Environment Variables
- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret for JWT token generation

### Database Migration
The API automatically creates subscription tables using GORM's `AutoMigrate`:
```go
db.AutoMigrate(
    &models.Subscription{},
    &models.SubscriptionPlan{},
    // ... other models
)
```

## 🌱 Seeding

### Subscription Plans
Default subscription plans are automatically created:
- Basic Monthly (30 days, ₹99)
- Premium Monthly (30 days, ₹299)
- Basic Yearly (365 days, ₹999)
- Premium Yearly (365 days, ₹2999)

### Test Subscriptions
Test subscriptions are created for existing users and series for development purposes.

## 🚧 Future Enhancements

### 1. Payment Integration
- Integrate with Razorpay for actual payment processing
- Handle payment failures and retries
- Implement webhook processing for payment events

### 2. Advanced Subscription Features
- Family plans and group subscriptions
- Promotional codes and discounts
- Subscription tiers and upgrades

### 3. Analytics and Reporting
- Subscription metrics and analytics
- Revenue reporting for creators
- Churn analysis and retention metrics

### 4. Notification System
- Subscription expiry reminders
- Payment failure notifications
- Renewal confirmations

## 📝 Notes

- **Mock Responses**: Payment integration returns mock responses for development
- **Validation**: Comprehensive validation for subscription requests
- **Error Handling**: Standard HTTP status codes with JSON error responses
- **Security**: JWT-based authentication with user ownership verification
- **Expiry Handling**: Automatic status updates for expired subscriptions

## 🐛 Troubleshooting

### Common Issues

1. **"Subscription required to access this content"**
   - User needs to subscribe to the series/creator
   - Check if the content is marked as paid

2. **"Subscription has expired"**
   - Subscription end date has passed
   - User needs to renew or create a new subscription

3. **"User already has an active subscription"**
   - Cannot create duplicate subscriptions to the same content
   - Check existing subscription status

4. **Database connection issues**
   - Check `DATABASE_URL` environment variable
   - Ensure PostgreSQL is running

### Debug Mode
Enable detailed logging by setting GORM log level in `config/database.go`:
```go
Logger: logger.Default.LogMode(logger.Info)
```
