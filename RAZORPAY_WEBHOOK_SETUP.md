# Razorpay Webhook Setup Guide

## 🎯 Webhook URL for Razorpay

Your webhook endpoint is already configured in the application:

```
POST /payments/webhook
```

### Production URL Format
```
https://your-domain.com/payments/webhook
```

### Local Testing URL (using ngrok)
```
https://your-ngrok-url.ngrok.io/payments/webhook
```

---

## 📋 Step-by-Step Razorpay Configuration

### 1. Access Razorpay Dashboard

1. Go to [https://dashboard.razorpay.com](https://dashboard.razorpay.com)
2. Log in to your account
3. Navigate to **Settings** → **Webhooks**

### 2. Create New Webhook

Click on **"+ Add New Webhook"** button

### 3. Configure Webhook URL

**Webhook URL**: Enter your endpoint URL
- **Production**: `https://your-domain.com/payments/webhook`
- **Development**: `https://your-ngrok-url.ngrok.io/payments/webhook`

**Active Events**: Select the following events that your application handles:

#### Subscription Events
- ✅ `subscription.activated` - When a subscription is activated
- ✅ `subscription.updated` - When subscription details are updated
- ✅ `subscription.cancelled` - When a subscription is cancelled
- ✅ `subscription.charged` - When subscription payment is charged
- ✅ `subscription.completed` - When subscription period completes
- ✅ `subscription.pending` - When subscription is pending
- ✅ `subscription.halted` - When subscription is halted
- ✅ `subscription.resumed` - When subscription is resumed
- ✅ `subscription.paused` - When subscription is paused

#### Payment Events
- ✅ `payment.authorized` - When payment is authorized
- ✅ `payment.captured` - When payment is captured
- ✅ `payment.failed` - When payment fails

### 4. Webhook Secret

After creating the webhook, Razorpay will provide you with a **Webhook Secret**.

**IMPORTANT**: Save this secret securely - you'll need it to verify webhook signatures.

Add it to your `.env.local` file:
```bash
RAZORPAY_WEBHOOK_SECRET=your_webhook_secret_here
```

---

## 🔧 Local Development Setup with ngrok

For local testing, you need to expose your localhost to the internet:

### Install ngrok
```bash
# macOS
brew install ngrok

# Or download from https://ngrok.com/download
```

### Start ngrok tunnel
```bash
# Expose port 8080 (your server port)
ngrok http 8080
```

### Copy the HTTPS URL
ngrok will provide a URL like:
```
https://abc123.ngrok.io
```

### Use this URL in Razorpay
```
https://abc123.ngrok.io/payments/webhook
```

---

## 🔐 Webhook Security Implementation

Your webhook handler needs to verify the signature. Here's how to enhance it:

### Current Implementation
Located in: `handlers/payment.go` (line 134)

### Enhanced Security (Recommended)

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
    // Read the raw body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    
    // Get signature from header
    signature := r.Header.Get("X-Razorpay-Signature")
    if signature == "" {
        http.Error(w, "Missing signature", http.StatusUnauthorized)
        return
    }
    
    // Verify signature
    webhookSecret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")
    if !verifySignature(body, signature, webhookSecret) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    // Parse the verified body
    var req WebhookRequest
    if err := json.Unmarshal(body, &req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Process webhook...
}

func verifySignature(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expectedSignature := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
```

---

## 📊 Webhook Events Handled

Your application currently handles these events (see `handlers/payment.go` line 148-167):

| Event | Handler | Description |
|-------|---------|-------------|
| `subscription.activated` | `handleSubscriptionActivated` | Updates subscription status to "active" |
| `subscription.updated` | `handleSubscriptionUpdated` | Handles subscription updates |
| `subscription.cancelled` | `handleSubscriptionCancelled` | Updates subscription status to "cancelled" |
| `payment.succeeded` | `handlePaymentSucceeded` | Processes successful payments |
| `payment.failed` | `handlePaymentFailed` | Handles failed payments |

---

## 🧪 Testing Webhooks

### 1. Using Razorpay Test Mode

Razorpay provides a test mode where you can simulate webhook events:

1. Go to **Dashboard** → **Webhooks**
2. Click on your webhook
3. Use **"Send Test Webhook"** button
4. Select event type and send

### 2. Using cURL (Manual Testing)

```bash
curl -X POST http://localhost:8080/payments/webhook \
  -H "Content-Type: application/json" \
  -H "X-Razorpay-Signature: test_signature" \
  -d '{
    "event_type": "subscription.activated",
    "data": {
      "subscription_id": "sub_test123",
      "status": "active"
    },
    "signature": "test_signature"
  }'
```

### 3. Monitor Webhook Logs

Check your server logs to see webhook events:
```bash
# In your terminal where the server is running
# You'll see logs like:
2025-11-21 12:23:35 Received webhook: subscription.activated
```

---

## 🚀 Production Deployment Checklist

Before going live:

- [ ] Add webhook URL in Razorpay dashboard (production mode)
- [ ] Add `RAZORPAY_WEBHOOK_SECRET` to production environment variables
- [ ] Implement signature verification (security enhancement above)
- [ ] Set up webhook retry logic for failed deliveries
- [ ] Add logging for all webhook events
- [ ] Set up monitoring/alerts for webhook failures
- [ ] Test all subscription lifecycle events
- [ ] Document webhook event handling for your team

---

## 📝 Environment Variables Needed

Add these to your `.env.local` (development) and production environment:

```bash
# Razorpay Configuration
RAZORPAY_KEY_ID=rzp_test_your_key_id
RAZORPAY_KEY_SECRET=your_key_secret
RAZORPAY_WEBHOOK_SECRET=your_webhook_secret

# For production, use live keys:
# RAZORPAY_KEY_ID=rzp_live_your_key_id
# RAZORPAY_KEY_SECRET=your_live_key_secret
```

---

## 🔍 Troubleshooting

### Webhook not receiving events

1. **Check URL is accessible**
   ```bash
   curl https://your-domain.com/payments/webhook
   ```

2. **Verify webhook is active** in Razorpay dashboard

3. **Check server logs** for any errors

4. **Verify signature** is being validated correctly

### Signature verification failing

1. Ensure `RAZORPAY_WEBHOOK_SECRET` matches the one in Razorpay dashboard
2. Check that you're reading the raw request body before parsing
3. Verify the signature header name is correct: `X-Razorpay-Signature`

### Events not processing

1. Check the `event_type` in webhook payload matches your switch cases
2. Verify database updates are working
3. Add logging to each event handler

---

## 📚 Additional Resources

- [Razorpay Webhook Documentation](https://razorpay.com/docs/webhooks/)
- [Razorpay Subscription API](https://razorpay.com/docs/api/subscriptions/)
- [Webhook Signature Verification](https://razorpay.com/docs/webhooks/validate-test/)
- [ngrok Documentation](https://ngrok.com/docs)

---

## 💡 Quick Summary

**Your Webhook URL**: `https://your-domain.com/payments/webhook`

**Steps**:
1. Go to Razorpay Dashboard → Settings → Webhooks
2. Add new webhook with your URL
3. Select subscription and payment events
4. Copy the webhook secret
5. Add secret to `.env.local`
6. For local testing, use ngrok to expose localhost
7. Test with Razorpay's test webhook feature

**Current Status**: ✅ Webhook endpoint is implemented and ready to receive events!
