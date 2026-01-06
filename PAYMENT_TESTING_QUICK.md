# 🧪 Quick Payment Testing Guide

## ✅ Your Setup is Ready!

- Server: Running on `localhost:8080`
- ngrok: Active (check terminal for URL)
- Subscription APIs: Working perfectly
- Test user: `+919650690943`

---

## 🚀 3 Ways to Test Payments

### Method 1: Quick API Test (Easiest)

```bash
# 1. Create OTP
go run cmd/create_test_otp/main.go

# 2. Get token and create subscription (one command)
TOKEN=$(curl -s -X POST http://localhost:8080/auth/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919650690943", "otp": "069009"}' | \
  jq -r '.access_token')

# 3. Create subscription
curl -X POST http://localhost:8080/api/payments/create-subscription \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "series",
    "target_id": "49c9cd71-e83d-4863-8757-f8a58e4b96b8",
    "plan_id": "plan_basic_monthly",
    "auto_renew": true
  }' | jq '.'

# 4. Check subscription
curl -X GET "http://localhost:8080/api/subscriptions/check?target_type=series&target_id=49c9cd71-e83d-4863-8757-f8a58e4b96b8" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

**Expected Result**: Subscription created with `status: "active"` and `has_access: true`

---

### Method 2: Razorpay Test Cards (Real Payment Flow)

#### A. Get Razorpay Test Keys

1. Go to https://dashboard.razorpay.com
2. Switch to **Test Mode**
3. Settings → API Keys → Generate Test Key
4. Copy Key ID and Secret

#### B. Add to `.env.local`

```bash
RAZORPAY_KEY_ID=rzp_test_XXXXXXXXXXXX
RAZORPAY_KEY_SECRET=XXXXXXXXXXXX
```

#### C. Use Test Cards

When Razorpay checkout opens:

**Success Card:**
- Number: `4111 1111 1111 1111`
- CVV: `123`
- Expiry: `12/25`

**Failure Card:**
- Number: `4000 0000 0000 0002`
- CVV: `123`
- Expiry: `12/25`

More cards: https://razorpay.com/docs/payments/payments/test-card-upi-details/

---

### Method 3: Webhook Testing

#### A. Configure Webhook in Razorpay

1. Get your ngrok URL from terminal
2. Go to Razorpay Dashboard → Settings → Webhooks
3. Add webhook: `https://your-ngrok-url.ngrok.io/payments/webhook`
4. Select events:
   - `subscription.activated`
   - `subscription.cancelled`
   - `payment.captured`
   - `payment.failed`

#### B. Test Webhook

```bash
# Manual webhook test
curl -X POST http://localhost:8080/payments/webhook \
  -H "Content-Type: application/json" \
  -H "X-Razorpay-Signature: test_sig" \
  -d '{
    "event": "subscription.activated",
    "payload": {
      "subscription": {
        "entity": {
          "id": "sub_test123",
          "status": "active"
        }
      }
    }
  }'
```

Or use **"Send Test Webhook"** button in Razorpay Dashboard.

---

## 📊 Verify Everything Works

```bash
# Check all subscriptions
curl -X GET http://localhost:8080/api/subscriptions \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Check subscription plans
curl -X GET http://localhost:8080/api/subscriptions/plans \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

---

## 🎯 Current Status

✅ **Subscription created successfully!**

You already have an active subscription to the test series. To test again:

```bash
# Option 1: Use a different series ID
# Get series list:
go run cmd/list_series/main.go

# Option 2: Cancel existing subscription first
curl -X POST http://localhost:8080/api/subscriptions/{subscription_id}/cancel \
  -H "Authorization: Bearer $TOKEN"
```

---

## 📚 Full Documentation

- **Complete Guide**: [`PAYMENT_TESTING_GUIDE.md`](file:///Users/vaibhaw/Developer/episodd/PAYMENT_TESTING_GUIDE.md)
- **Webhook Setup**: [`RAZORPAY_WEBHOOK_SETUP.md`](file:///Users/vaibhaw/Developer/episodd/RAZORPAY_WEBHOOK_SETUP.md)
- **API Guide**: [`SUBSCRIPTION_API_GUIDE.md`](file:///Users/vaibhaw/Developer/episodd/SUBSCRIPTION_API_GUIDE.md)

---

## 💡 Quick Tips

1. **Test Mode**: Always use Razorpay test keys for development
2. **ngrok**: Keep it running for webhook testing
3. **Test Cards**: Use Razorpay's test card numbers
4. **Logs**: Check server logs to see webhook events
5. **Database**: Query subscriptions table to verify data

**You're all set to test payments!** 🎉
