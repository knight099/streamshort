# Creator Payment Plan Specification

## Overview
This document outlines the payment structure, revenue sharing model, and API requirements for creator monetization on Episodd.

---

## 1. Revenue Sharing Model

### 1.1 Platform Fee Structure
- **Platform Fee**: 30% of gross revenue
- **Creator Share**: 70% of gross revenue
- **Minimum Payout Threshold**: ₹500 (INR)

### 1.2 Revenue Sources

#### A. Subscription Revenue (Series-specific)
- **Source**: Users subscribing to a creator's premium series
- **Calculation**: 
  - Monthly subscription amount × Creator share (70%)
  - Example: ₹100/month subscription → Creator earns ₹70/month

#### B. One-time Purchase Revenue
- **Source**: Users purchasing individual series/episodes
- **Calculation**: 
  - Purchase amount × Creator share (70%)
  - Example: ₹500 purchase → Creator earns ₹350

#### C. Ad Revenue (Future)
- **Source**: In-stream advertisements
- **Calculation**: 
  - CPM (Cost Per Mille) × Views × Creator share (70%)
  - To be implemented in Phase 2

---

## 2. Earnings Calculation

### 2.1 Real-time Earnings Tracking
Earnings should be calculated and updated in real-time based on:
- Active subscriptions (recurring monthly)
- One-time purchases
- View-based metrics (for future ad revenue)

### 2.2 Earnings Components
```json
{
  "total_earnings": 5000.00,
  "pending_earnings": 1200.00,
  "paid_earnings": 3800.00,
  "breakdown": {
    "subscription_revenue": 3500.00,
    "one_time_purchases": 1500.00,
    "ad_revenue": 0.00
  },
  "currency": "INR"
}
```

---

## 3. Payout System

### 3.1 Payout Methods
- **Primary**: Razorpay Payouts API (Bank Transfer)
- **Alternative**: UPI (to be added)
- **Future**: Razorpay X (for international creators)

### 3.2 Payout Schedule
- **Frequency**: Monthly (1st of each month)
- **Processing Time**: 3-5 business days
- **Minimum Threshold**: ₹500 must be accumulated before payout

### 3.3 Payout Status
- `pending`: Earnings accumulated but below threshold
- `processing`: Payout initiated, awaiting bank transfer
- `completed`: Successfully transferred to creator's account
- `failed`: Transfer failed (wrong account details, etc.)

---

## 4. Required API Endpoints

### 4.1 Creator Earnings Endpoint

#### GET `/api/creators/{creator_id}/earnings`
**Description**: Get creator's earnings summary

**Response**:
```json
{
  "creator_id": "uuid",
  "total_earnings": 5000.00,
  "pending_earnings": 1200.00,
  "paid_earnings": 3800.00,
  "available_for_payout": 1200.00,
  "minimum_threshold": 500.00,
  "can_request_payout": true,
  "breakdown": {
    "subscription_revenue": {
      "total": 3500.00,
      "active_subscriptions": 50,
      "monthly_recurring": 350.00
    },
    "one_time_purchases": {
      "total": 1500.00,
      "count": 15
    },
    "ad_revenue": {
      "total": 0.00
    }
  },
  "currency": "INR",
  "last_payout_date": "2025-11-01T00:00:00Z",
  "next_payout_date": "2025-12-01T00:00:00Z"
}
```

---

### 4.2 Payout Request Endpoint

#### POST `/api/creators/{creator_id}/payouts/request`
**Description**: Request a payout (if above threshold)

**Request**:
```json
{
  "amount": 1200.00,
  "payout_method": "bank_transfer"
}
```

**Response**:
```json
{
  "payout_id": "uuid",
  "creator_id": "uuid",
  "amount": 1200.00,
  "status": "processing",
  "requested_at": "2025-12-11T17:00:00Z",
  "estimated_completion": "2025-12-16T17:00:00Z",
  "razorpay_payout_id": "pout_xxxxx"
}
```

---

### 4.3 Payout History Endpoint

#### GET `/api/creators/{creator_id}/payouts`
**Description**: Get payout history

**Query Parameters**:
- `page` (optional): Page number (default: 1)
- `per_page` (optional): Items per page (default: 20)

**Response**:
```json
{
  "payouts": [
    {
      "payout_id": "uuid",
      "amount": 2500.00,
      "status": "completed",
      "requested_at": "2025-11-01T00:00:00Z",
      "completed_at": "2025-11-04T10:30:00Z",
      "razorpay_payout_id": "pout_xxxxx",
      "transaction_reference": "TXN123456789"
    }
  ],
  "total": 5,
  "page": 1,
  "per_page": 20
}
```

---

### 4.4 Update Payout Details Endpoint

#### PUT `/api/creators/{creator_id}/payout-details`
**Description**: Update creator's payout/bank account details

**Request**:
```json
{
  "account_holder_name": "John Doe",
  "account_number": "1234567890",
  "ifsc_code": "HDFC0001234",
  "bank_name": "HDFC Bank",
  "account_type": "savings", // savings, current
  "upi_id": "john@upi" // optional
}
```

**Response**:
```json
{
  "creator_id": "uuid",
  "payout_details": {
    "account_holder_name": "John Doe",
    "account_number": "1234567890",
    "ifsc_code": "HDFC0001234",
    "bank_name": "HDFC Bank",
    "account_type": "savings",
    "upi_id": "john@upi",
    "verified": true,
    "updated_at": "2025-12-11T17:00:00Z"
  }
}
```

---

### 4.5 Earnings Breakdown Endpoint

#### GET `/api/creators/{creator_id}/earnings/breakdown`
**Description**: Get detailed earnings breakdown by series/episode

**Query Parameters**:
- `start_date` (optional): Start date (ISO 8601)
- `end_date` (optional): End date (ISO 8601)
- `series_id` (optional): Filter by specific series

**Response**:
```json
{
  "period": {
    "start_date": "2025-11-01T00:00:00Z",
    "end_date": "2025-12-01T00:00:00Z"
  },
  "total_earnings": 3500.00,
  "series_breakdown": [
    {
      "series_id": "uuid",
      "series_title": "Didi Ke Sapne",
      "earnings": 2500.00,
      "subscription_revenue": 2000.00,
      "one_time_purchases": 500.00,
      "active_subscriptions": 20,
      "total_views": 5000
    }
  ],
  "episode_breakdown": [
    {
      "episode_id": "uuid",
      "episode_title": "Episode 1",
      "series_id": "uuid",
      "earnings": 100.00,
      "views": 1000,
      "watch_time_seconds": 50000
    }
  ]
}
```

---

## 5. Database Schema Requirements

### 5.1 Earnings Table
```sql
CREATE TABLE creator_earnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES creators(id),
    amount DECIMAL(10, 2) NOT NULL,
    earnings_type VARCHAR(50) NOT NULL, -- subscription, one_time, ad_revenue
    series_id UUID REFERENCES series(id),
    episode_id UUID REFERENCES episodes(id),
    subscription_id UUID REFERENCES subscriptions(id),
    status VARCHAR(20) NOT NULL, -- pending, paid, cancelled
    created_at TIMESTAMP DEFAULT NOW(),
    paid_at TIMESTAMP,
    payout_id UUID REFERENCES payouts(id)
);

CREATE INDEX idx_creator_earnings_creator_id ON creator_earnings(creator_id);
CREATE INDEX idx_creator_earnings_status ON creator_earnings(status);
CREATE INDEX idx_creator_earnings_created_at ON creator_earnings(created_at);
```

### 5.2 Payouts Table
```sql
CREATE TABLE payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES creators(id),
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL, -- pending, processing, completed, failed
    razorpay_payout_id VARCHAR(255),
    transaction_reference VARCHAR(255),
    requested_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    failure_reason TEXT,
    payout_method VARCHAR(50) NOT NULL, -- bank_transfer, upi
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payouts_creator_id ON payouts(creator_id);
CREATE INDEX idx_payouts_status ON payouts(status);
```

### 5.3 Creator Payout Details Table
```sql
CREATE TABLE creator_payout_details (
    creator_id UUID PRIMARY KEY REFERENCES creators(id),
    account_holder_name VARCHAR(255) NOT NULL,
    account_number VARCHAR(50) NOT NULL,
    ifsc_code VARCHAR(20) NOT NULL,
    bank_name VARCHAR(255) NOT NULL,
    account_type VARCHAR(20) NOT NULL, -- savings, current
    upi_id VARCHAR(255),
    verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## 6. Integration with Razorpay Payouts

### 6.1 Razorpay Payouts API Integration
Use Razorpay Payouts API for bank transfers:

**Endpoint**: `https://api.razorpay.com/v1/payouts`

**Request**:
```json
{
  "account_number": "2323230000000000", // Razorpay Fund Account ID
  "fund_account": {
    "account_type": "bank_account",
    "bank_account": {
      "name": "John Doe",
      "ifsc": "HDFC0001234",
      "account_number": "1234567890"
    }
  },
  "amount": 120000, // Amount in paise (₹1200.00)
  "currency": "INR",
  "mode": "NEFT",
  "purpose": "payout",
  "queue_if_low_balance": true,
  "reference_id": "payout_uuid",
  "narration": "Episodd Creator Payout"
}
```

### 6.2 Webhook Handling
Handle Razorpay payout webhooks:
- `payout.processed`: Payout completed successfully
- `payout.failed`: Payout failed (update status, notify creator)

---

## 7. View Tracking API (Required for Earnings)

### 7.1 Series View Tracking

#### POST `/api/series/{series_id}/view`
**Description**: Track a view for a series (currently returns 404 - needs implementation)

**Request**:
```json
{
  "series_id": "uuid",
  "action": "view"
}
```

**Response**:
```json
{
  "success": true,
  "series_id": "uuid",
  "view_count": 5001
}
```

**Implementation Notes**:
- Increment `view_count` in `series` table
- Create entry in `series_views` table for analytics
- Update creator earnings if applicable (for future ad revenue)

---

## 8. Earnings Calculation Logic

### 8.1 Subscription Revenue
```python
# When a user subscribes to a creator's series
subscription_amount = plan.amount  # e.g., ₹100
platform_fee = subscription_amount * 0.30  # ₹30
creator_earnings = subscription_amount * 0.70  # ₹70

# Create earnings record
create_earnings_record(
    creator_id=series.creator_id,
    amount=creator_earnings,
    earnings_type="subscription",
    series_id=series.id,
    subscription_id=subscription.id
)
```

### 8.2 Monthly Recurring Revenue
```python
# Run monthly cron job (1st of each month)
for subscription in active_subscriptions:
    if subscription.status == "active":
        create_earnings_record(
            creator_id=subscription.series.creator_id,
            amount=subscription.plan.amount * 0.70,
            earnings_type="subscription",
            series_id=subscription.series.id,
            subscription_id=subscription.id
        )
```

---

## 9. Frontend Integration Points

### 9.1 Creator Dashboard
- Display total earnings, pending earnings, paid earnings
- Show earnings breakdown by series
- Display payout history
- Show "Request Payout" button (if above threshold)

### 9.2 Payout Settings
- Form to update bank account details
- Display current payout details
- Show verification status

---

## 10. Security & Compliance

### 10.1 KYC Requirements
- Creators must complete KYC before receiving payouts
- Verify bank account details before first payout
- Store sensitive data encrypted

### 10.2 Tax Compliance
- Generate TDS certificates (if applicable)
- Maintain records for tax purposes
- Issue Form 16A for TDS deductions

---

## 11. Testing Checklist

- [ ] Earnings calculation for subscriptions
- [ ] Earnings calculation for one-time purchases
- [ ] Payout request (above threshold)
- [ ] Payout request (below threshold) - should fail
- [ ] Razorpay payout integration
- [ ] Webhook handling for payout status
- [ ] View tracking increments series view count
- [ ] Earnings breakdown by series/episode
- [ ] Payout history pagination
- [ ] Bank account verification

---

## 12. Priority Implementation Order

1. **Phase 1 (Critical)**:
   - Series view tracking API (`POST /api/series/{series_id}/view`)
   - Earnings calculation for subscriptions
   - GET `/api/creators/{creator_id}/earnings`
   - GET `/api/creators/{creator_id}/payouts`

2. **Phase 2 (High Priority)**:
   - POST `/api/creators/{creator_id}/payouts/request`
   - PUT `/api/creators/{creator_id}/payout-details`
   - Razorpay Payouts integration
   - Webhook handling

3. **Phase 3 (Medium Priority)**:
   - GET `/api/creators/{creator_id}/earnings/breakdown`
   - Earnings for one-time purchases
   - Payout history filters

4. **Phase 4 (Future)**:
   - Ad revenue integration
   - UPI payouts
   - Tax certificate generation

---

## 13. API Response Examples

### Success Response Format
```json
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}
```

### Error Response Format
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Earnings below minimum payout threshold of ₹500",
    "details": {
      "current_balance": 300.00,
      "minimum_threshold": 500.00
    }
  }
}
```

---

## 14. Notes for Backend Team

1. **View Tracking**: The endpoint `/api/series/{series_id}/view` currently returns 404. This needs to be implemented to track views and update view counts.

2. **Earnings Calculation**: Should be real-time or near-real-time. Consider using database triggers or background jobs.

3. **Payout Processing**: Use Razorpay Payouts API. Handle webhooks for status updates.

4. **Currency**: Currently INR only. Consider multi-currency support in future.

5. **Platform Fee**: Currently 30%. Make this configurable in admin settings.

6. **Minimum Threshold**: Currently ₹500. Make this configurable.

---

## Contact
For questions or clarifications, please refer to this document or contact the development team.

