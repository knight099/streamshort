# Frontend Integration Guide: Razorpay Subscription Changes

## 🚨 Important Changes

The subscription payment flow has been updated to integrate with **Razorpay**. The API now returns **real Razorpay subscription IDs** and **valid checkout URLs** instead of mock data.

---

## 📋 API Changes

### **POST `/api/payments/create-subscription`**

#### Request (Unchanged)
```json
{
  "plan_id": "all_access_30d",
  "auto_renew": true
}
```

#### Response (Changed)

**Before:**
- `subscription_id`: UUID (e.g., `"550e8400-e29b-41d4-a716-446655440000"`)
- `status`: Always `"active"` immediately
- `checkout_url`: Invalid JS library URL

**After:**
- `subscription_id`: **Razorpay subscription ID** (e.g., `"sub_ABC123xyz"`) ⚠️
- `status`: `"pending"` initially (becomes `"active"` after payment)
- `checkout_url`: **Valid payment link** (e.g., `"https://rzp.io/l/xxxxx"`)

**Example Response:**
```json
{
  "subscription_id": "sub_ABC123xyz",
  "status": "pending",
  "plan_id": "all_access_30d",
  "start_date": "2024-01-15T10:00:00Z",
  "end_date": "2024-02-15T10:00:00Z",
  "next_billing": "2024-02-15T10:00:00Z",
  "checkout_url": "https://rzp.io/l/xxxxx"
}
```

---

## 🔄 Updated Payment Flow

### Step 1: Create Subscription
```dart
// Call the API
POST /api/payments/create-subscription
Headers: Authorization: Bearer <token>
Body: { "plan_id": "all_access_30d", "auto_renew": true }

// Response contains checkout_url
```

### Step 2: Open Payment Link
**Option A: Use Payment Link (Recommended)**
```dart
// Open the checkout_url in browser or WebView
launchUrl(Uri.parse(response.checkout_url));
```

**Option B: Use Razorpay Flutter SDK**
```dart
// If you prefer using Razorpay SDK directly
Razorpay razorpay = Razorpay();
razorpay.on(Razorpay.EVENT_PAYMENT_SUCCESS, handlePaymentSuccess);
razorpay.on(Razorpay.EVENT_PAYMENT_ERROR, handlePaymentError);

var options = {
  'key': 'rzp_test_xxxxx', // Your Razorpay Key ID
  'subscription_id': response.subscription_id, // Use the subscription_id from API
  'amount': amount * 100, // Amount in paise
  'name': 'StreamShort',
  'description': 'Subscription',
};

razorpay.open(options);
```

### Step 3: Verify Payment Status
After payment completes, **poll or check** subscription status:

```dart
// Check subscription status
GET /api/subscriptions/check
Headers: Authorization: Bearer <token>

// Response will show if subscription is now "active"
{
  "has_access": true,
  "active_subscriptions": [
    {
      "id": "sub_ABC123xyz",
      "status": "active", // ✅ Changed from "pending"
      ...
    }
  ]
}
```

---

## ⚠️ Important Notes

### 1. **Subscription Status Flow**
- **`pending`**: Created but payment not completed yet
- **`active`**: Payment successful, subscription active
- **`cancelled`**: User cancelled subscription
- **`expired`**: Subscription period ended
- **`halted`**: Payment failed (auto-retry may happen)

### 2. **Subscription ID Format**
- **Old**: UUID format (`550e8400-e29b-41d4-a716-446655440000`)
- **New**: Razorpay format (`sub_ABC123xyz`)
- **Action Required**: Update any code that stores/uses subscription IDs

### 3. **Payment Confirmation**
- Subscription is **NOT active** immediately after calling `create-subscription`
- Status starts as `"pending"`
- Backend activates subscription **only after** Razorpay webhook confirms payment
- **Recommendation**: Poll `/api/subscriptions/check` every 2-3 seconds after redirecting to payment, or use webhooks

### 4. **Error Handling**
If `create-subscription` returns an error:
- `500` with "Payment gateway not configured" → Backend missing Razorpay credentials
- `400` with "Invalid or inactive plan_id" → Plan doesn't exist in database
- `500` with "Failed to create subscription" → Razorpay API error (check plan_id matches Razorpay plan)

---

## 🔧 Frontend Implementation Checklist

- [ ] Update subscription ID storage (use `sub_xxxxx` format)
- [ ] Handle `"pending"` status in UI (show "Payment in progress" or similar)
- [ ] Implement payment link opening (browser/WebView or Razorpay SDK)
- [ ] Add polling/checking for subscription activation after payment
- [ ] Update error messages for new error cases
- [ ] Test with test Razorpay credentials (`rzp_test_...`)

---

## 📱 Example Flutter Implementation

```dart
Future<void> createSubscription(String planId) async {
  try {
    // Step 1: Create subscription
    final response = await http.post(
      Uri.parse('$baseUrl/api/payments/create-subscription'),
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: jsonEncode({
        'plan_id': planId,
        'auto_renew': true,
      }),
    );

    if (response.statusCode == 201) {
      final data = jsonDecode(response.body);
      final checkoutUrl = data['checkout_url'];
      final subscriptionId = data['subscription_id']; // sub_xxxxx format
      
      // Step 2: Open payment link
      if (await canLaunchUrl(Uri.parse(checkoutUrl))) {
        await launchUrl(Uri.parse(checkoutUrl));
        
        // Step 3: Poll for activation (optional)
        _pollSubscriptionStatus(subscriptionId);
      }
    }
  } catch (e) {
    // Handle error
  }
}

Future<void> _pollSubscriptionStatus(String subscriptionId) async {
  // Poll every 3 seconds for up to 60 seconds
  for (int i = 0; i < 20; i++) {
    await Future.delayed(Duration(seconds: 3));
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/subscriptions/check'),
      headers: {'Authorization': 'Bearer $token'},
    );
    
    final data = jsonDecode(response.body);
    if (data['has_access'] == true) {
      // Subscription activated!
      break;
    }
  }
}
```

---

## 🆘 Support

If you encounter issues:
1. Check backend logs for Razorpay API errors
2. Verify `RAZORPAY_KEY_ID` and `RAZORPAY_KEY_SECRET` are set
3. Ensure Razorpay plans exist with matching `plan_id` values
4. Check webhook URL is configured in Razorpay dashboard

