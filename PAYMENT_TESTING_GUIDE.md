# Payment Testing Guide for Razorpay Integration

## 🎯 Overview

This guide shows you how to test the complete payment flow for subscriptions using Razorpay's test mode.

## 📋 Prerequisites

✅ Server running on `localhost:8080`  
✅ ngrok tunnel active (exposing localhost to internet)  
✅ Razorpay account in **Test Mode**  
✅ Webhook configured in Razorpay dashboard

---

## 🔧 Step 1: Get Your ngrok URL

Your ngrok is already running! Check the terminal output for your URL:

```bash
# Look for a line like:
Forwarding: https://abc123.ngrok.io -> http://localhost:8080
```

**Your ngrok URL**: `https://[your-id].ngrok.io`

---

## 🔑 Step 2: Configure Razorpay Test Keys

### Get Test API Keys

1. Go to [Razorpay Dashboard](https://dashboard.razorpay.com)
2. Switch to **Test Mode** (toggle in top-left)
3. Navigate to **Settings** → **API Keys**
4. Click **Generate Test Key** if you don't have one
5. Copy both:
   - **Key ID**: `rzp_test_XXXXXXXXXXXX`
   - **Key Secret**: `XXXXXXXXXXXX`

### Add to Environment

Update your `.env.local`:

```bash
# Razorpay Test Keys
RAZORPAY_KEY_ID=rzp_test_your_key_id_here
RAZORPAY_KEY_SECRET=your_key_secret_here
RAZORPAY_WEBHOOK_SECRET=your_webhook_secret_here
```

**Restart your server** after adding these:
```bash
# Stop current server (Ctrl+C)
# Then restart:
go run main.go
```

---

## 🧪 Step 3: Test Payment Flow

### Method 1: Using Razorpay Checkout (Recommended)

This simulates the real user experience.

#### A. Create a Test HTML Page

Create `test_payment.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Test Razorpay Payment</title>
    <script src="https://checkout.razorpay.com/v1/checkout.js"></script>
</head>
<body>
    <h1>Test Subscription Payment</h1>
    <button id="payButton">Subscribe Now - ₹99</button>

    <script>
        const API_BASE = 'http://localhost:8080';
        const AUTH_TOKEN = 'YOUR_AUTH_TOKEN_HERE'; // Get from login

        document.getElementById('payButton').onclick = async function() {
            try {
                // Step 1: Create subscription via your API
                const response = await fetch(`${API_BASE}/api/payments/create-subscription`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${AUTH_TOKEN}`
                    },
                    body: JSON.stringify({
                        target_type: 'series',
                        target_id: '49c9cd71-e83d-4863-8757-f8a58e4b96b8',
                        plan_id: 'plan_basic_monthly',
                        auto_renew: true
                    })
                });

                const data = await response.json();
                console.log('Subscription created:', data);

                // Step 2: Open Razorpay Checkout
                const options = {
                    key: 'rzp_test_your_key_id', // Your Razorpay Test Key ID
                    subscription_id: data.subscription_id,
                    name: 'StreamShort',
                    description: 'Monthly Subscription',
                    image: 'https://your-logo-url.com/logo.png',
                    handler: function(response) {
                        alert('Payment successful!');
                        console.log('Payment ID:', response.razorpay_payment_id);
                        console.log('Subscription ID:', response.razorpay_subscription_id);
                        console.log('Signature:', response.razorpay_signature);
                    },
                    prefill: {
                        name: 'Test User',
                        email: 'test@example.com',
                        contact: '+919650690943'
                    },
                    theme: {
                        color: '#3399cc'
                    }
                };

                const rzp = new Razorpay(options);
                rzp.open();
            } catch (error) {
                console.error('Error:', error);
                alert('Failed to create subscription');
            }
        };
    </script>
</body>
</html>
```

#### B. Get Your Auth Token

```bash
# 1. Create OTP
go run cmd/create_test_otp/main.go

# 2. Login and get token
curl -X POST http://localhost:8080/auth/otp/verify \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+919650690943",
    "otp": "069009"
  }'

# Copy the access_token from response
```

#### C. Update HTML and Open

1. Replace `YOUR_AUTH_TOKEN_HERE` with your actual token
2. Replace `rzp_test_your_key_id` with your Razorpay Key ID
3. Open `test_payment.html` in your browser
4. Click "Subscribe Now"

#### D. Use Test Card Details

When Razorpay checkout opens, use these **test card details**:

**Successful Payment:**
- Card Number: `4111 1111 1111 1111`
- CVV: Any 3 digits (e.g., `123`)
- Expiry: Any future date (e.g., `12/25`)
- Name: Any name

**Failed Payment:**
- Card Number: `4000 0000 0000 0002`
- CVV: Any 3 digits
- Expiry: Any future date

**More Test Cards**: [Razorpay Test Cards](https://razorpay.com/docs/payments/payments/test-card-upi-details/)

---

### Method 2: Direct API Testing (Quick Test)

Test the subscription creation endpoint directly:

```bash
# 1. Get auth token first
TOKEN=$(curl -s -X POST http://localhost:8080/auth/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919650690943", "otp": "069009"}' | \
  grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# 2. Create subscription
curl -X POST http://localhost:8080/api/payments/create-subscription \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "series",
    "target_id": "49c9cd71-e83d-4863-8757-f8a58e4b96b8",
    "plan_id": "plan_basic_monthly",
    "auto_renew": true
  }' | jq '.'

# 3. Check subscription status
curl -X GET "http://localhost:8080/api/subscriptions/check?target_type=series&target_id=49c9cd71-e83d-4863-8757-f8a58e4b96b8" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

---

### Method 3: Test Webhook Events

#### A. Manual Webhook Test

```bash
# Test subscription activation webhook
curl -X POST http://localhost:8080/payments/webhook \
  -H "Content-Type: application/json" \
  -H "X-Razorpay-Signature: test_signature" \
  -d '{
    "event": "subscription.activated",
    "payload": {
      "subscription": {
        "entity": {
          "id": "sub_test123",
          "status": "active",
          "customer_id": "cust_test123"
        }
      }
    }
  }'
```

#### B. Using Razorpay Dashboard

1. Go to **Dashboard** → **Webhooks**
2. Click on your webhook
3. Click **"Send Test Webhook"**
4. Select event type (e.g., `subscription.activated`)
5. Click **Send**
6. Check your server logs

---

## 🧪 Step 4: Complete Test Scenario

### Scenario: User Subscribes to a Series

```bash
#!/bin/bash

echo "🧪 Complete Payment Flow Test"
echo "=============================="

# 1. Authenticate
echo "1️⃣ Authenticating user..."
go run cmd/create_test_otp/main.go

TOKEN=$(curl -s -X POST http://localhost:8080/auth/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919650690943", "otp": "069009"}' | \
  grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

echo "✅ Got token: ${TOKEN:0:20}..."
echo ""

# 2. Check available plans
echo "2️⃣ Getting subscription plans..."
curl -s -X GET http://localhost:8080/api/subscriptions/plans \
  -H "Authorization: Bearer $TOKEN" | jq '.[0:2]'
echo ""

# 3. Create subscription
echo "3️⃣ Creating subscription..."
SUB_RESPONSE=$(curl -s -X POST http://localhost:8080/api/payments/create-subscription \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "series",
    "target_id": "49c9cd71-e83d-4863-8757-f8a58e4b96b8",
    "plan_id": "plan_basic_monthly",
    "auto_renew": true
  }')

echo "$SUB_RESPONSE" | jq '.'
SUB_ID=$(echo "$SUB_RESPONSE" | jq -r '.subscription_id')
echo ""

# 4. Verify subscription
echo "4️⃣ Verifying subscription..."
curl -s -X GET "http://localhost:8080/api/subscriptions/check?target_type=series&target_id=49c9cd71-e83d-4863-8757-f8a58e4b96b8" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# 5. Get subscription details
echo "5️⃣ Getting subscription details..."
curl -s -X GET "http://localhost:8080/api/subscriptions/$SUB_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

echo "✅ Test completed!"
```

Save as `test_payment_flow.sh` and run:
```bash
chmod +x test_payment_flow.sh
./test_payment_flow.sh
```

---

## 📊 Monitor Payment Events

### Check Server Logs

Your server logs will show:
```
2025-11-21 12:49:12 [INFO] Subscription created: sub_abc123
2025-11-21 12:49:15 [INFO] Webhook received: subscription.activated
2025-11-21 12:49:15 [INFO] Updated subscription status: active
```

### Check Database

```bash
# Connect to your database and check subscriptions
psql $DATABASE_URL -c "SELECT id, user_id, status, plan_id, amount FROM subscriptions ORDER BY created_at DESC LIMIT 5;"
```

---

## 🎯 Test Checklist

- [ ] ✅ Server running on localhost:8080
- [ ] ✅ ngrok tunnel active
- [ ] ✅ Razorpay test keys configured
- [ ] ✅ Webhook URL added to Razorpay
- [ ] ✅ Can authenticate and get token
- [ ] ✅ Can fetch subscription plans
- [ ] ✅ Can create subscription
- [ ] ✅ Can check subscription status
- [ ] ✅ Webhook receives events
- [ ] ✅ Test card payment works
- [ ] ✅ Subscription appears in database

---

## 🐛 Troubleshooting

### Payment fails immediately
- Check Razorpay test mode is enabled
- Verify API keys are correct
- Check card details are from test card list

### Webhook not receiving events
- Verify ngrok URL is correct in Razorpay
- Check webhook secret matches
- Look for signature verification errors in logs

### "Subscription already exists" error
- User already has active subscription to that series
- Cancel existing subscription first or use different series

---

## 📚 Razorpay Test Resources

- **Test Cards**: https://razorpay.com/docs/payments/payments/test-card-upi-details/
- **Test UPI**: Use VPA `success@razorpay`
- **Test Wallets**: All test wallets auto-succeed
- **Webhook Testing**: https://razorpay.com/docs/webhooks/test/

---

## 🚀 Next Steps

After testing successfully:

1. **Implement Real Payment Integration**
   - Replace mock responses with actual Razorpay API calls
   - Add proper error handling
   - Implement retry logic

2. **Add Payment Verification**
   - Verify payment signature
   - Update subscription status based on payment

3. **Handle Edge Cases**
   - Payment failures
   - Subscription renewals
   - Cancellations and refunds

4. **Go Live**
   - Switch to Razorpay Live mode
   - Update API keys to production keys
   - Update webhook URL to production domain
   - Test with real small amount first

---

## 💡 Quick Test Command

```bash
# One-liner to test the complete flow
go run cmd/create_test_otp/main.go && \
TOKEN=$(curl -s -X POST http://localhost:8080/auth/otp/verify -H "Content-Type: application/json" -d '{"phone": "+919650690943", "otp": "069009"}' | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4) && \
curl -X POST http://localhost:8080/api/payments/create-subscription -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"target_type": "series", "target_id": "49c9cd71-e83d-4863-8757-f8a58e4b96b8", "plan_id": "plan_basic_monthly", "auto_renew": true}' | jq '.'
```

This creates OTP, authenticates, and creates a subscription in one command!
