# Flutter Frontend: Razorpay Subscription Integration Guide

## 🎯 What Changed

The subscription payment system now integrates with **Razorpay**. The API behavior has changed:

### Key Changes:
1. **Subscription ID Format**: Now returns Razorpay format (`sub_ABC123xyz`) instead of UUID
2. **Status Flow**: Subscriptions start as `"pending"` and become `"active"` after payment
3. **Checkout URL**: Returns a valid Razorpay payment link
4. **Payment Confirmation**: Subscription activates via webhook (not immediately)

---

## 📱 Implementation Guide

### Step 1: Create Subscription API Call

```dart
Future<Map<String, dynamic>> createSubscription({
  required String planId,
  required String authToken,
  bool autoRenew = true,
}) async {
  final dio = Dio();
  
  try {
    final response = await dio.post(
      'https://api.episodd.com/api/payments/create-subscription',
      options: Options(
        headers: {
          'Authorization': 'Bearer $authToken',
          'Content-Type': 'application/json',
        },
      ),
      data: {
        'plan_id': planId,  // e.g., 'plan_basic_monthly'
        'auto_renew': autoRenew,
        'target_type': 'global',  // Optional, defaults to 'global'
      },
    );

    if (response.statusCode == 201) {
      return response.data;
    }
    throw Exception('Failed to create subscription');
  } on DioException catch (e) {
    if (e.response != null) {
      throw Exception(e.response?.data['error'] ?? 'Payment error');
    }
    throw Exception('Network error');
  }
}
```

### Step 2: Handle Response

**Response Structure:**
```json
{
  "subscription_id": "sub_ABC123xyz",  // ⚠️ Razorpay ID, not UUID
  "status": "pending",                 // ⚠️ Starts as "pending"
  "plan_id": "plan_basic_monthly",
  "start_date": "2024-01-15T10:00:00Z",
  "end_date": "2024-02-15T10:00:00Z",
  "next_billing": "2024-02-15T10:00:00Z",
  "checkout_url": "https://rzp.io/l/xxxxx"  // ✅ Valid payment link
}
```

### Step 3: Open Payment Link

**Option A: Using url_launcher (Recommended)**
```dart
import 'package:url_launcher/url_launcher.dart';

Future<void> openPaymentLink(String checkoutUrl) async {
  final uri = Uri.parse(checkoutUrl);
  if (await canLaunchUrl(uri)) {
    await launchUrl(
      uri,
      mode: LaunchMode.externalApplication, // Opens in browser
    );
  } else {
    throw Exception('Could not launch payment URL');
  }
}
```

**Option B: Using Razorpay Flutter SDK**
```dart
import 'package:razorpay_flutter/razorpay_flutter.dart';

final _razorpay = Razorpay();

void initRazorpay() {
  _razorpay.on(Razorpay.EVENT_PAYMENT_SUCCESS, _handlePaymentSuccess);
  _razorpay.on(Razorpay.EVENT_PAYMENT_ERROR, _handlePaymentError);
}

void openRazorpayCheckout({
  required String subscriptionId,
  required double amount,
}) {
  var options = {
    'key': 'rzp_test_xxxxx', // Your Razorpay Key ID from backend config
    'subscription_id': subscriptionId, // From API response
    'amount': (amount * 100).toInt(), // Amount in paise
    'name': 'Episodd',
    'description': 'Subscription Payment',
    'prefill': {
      'contact': userPhone,
      'email': userEmail,
    },
    'external': {
      'wallets': ['paytm']
    }
  };
  
  _razorpay.open(options);
}

void _handlePaymentSuccess(PaymentSuccessResponse response) {
  // Payment successful - check subscription status
  checkSubscriptionStatus();
}

void _handlePaymentError(PaymentFailureResponse response) {
  // Handle payment failure
  print('Payment failed: ${response.message}');
}
```

### Step 4: Verify Subscription Activation

**Poll for Status Update:**
```dart
Future<bool> checkSubscriptionStatus(String authToken) async {
  final dio = Dio();
  
  try {
    final response = await dio.get(
      'https://api.episodd.com/api/subscriptions/check',
      options: Options(
        headers: {
          'Authorization': 'Bearer $authToken',
        },
      ),
    );

    if (response.statusCode == 200) {
      final data = response.data;
      return data['has_access'] == true;
    }
    return false;
  } catch (e) {
    return false;
  }
}

// Poll every 3 seconds for up to 60 seconds
Future<void> waitForSubscriptionActivation(String authToken) async {
  for (int i = 0; i < 20; i++) {
    await Future.delayed(Duration(seconds: 3));
    
    final isActive = await checkSubscriptionStatus(authToken);
    if (isActive) {
      // Subscription activated!
      return;
    }
  }
  
  // Timeout - show message to user
  throw Exception('Payment verification timeout. Please refresh.');
}
```

---

## 🔄 Complete Payment Flow

```dart
Future<void> handleSubscriptionPayment({
  required String planId,
  required String authToken,
}) async {
  try {
    // Step 1: Create subscription
    final subscriptionData = await createSubscription(
      planId: planId,
      authToken: authToken,
    );
    
    final checkoutUrl = subscriptionData['checkout_url'];
    final subscriptionId = subscriptionData['subscription_id'];
    
    // Step 2: Show loading state
    showLoadingDialog('Redirecting to payment...');
    
    // Step 3: Open payment link
    await openPaymentLink(checkoutUrl);
    
    // Step 4: Wait for user to complete payment
    // (User completes payment in browser/app)
    
    // Step 5: Poll for activation
    await waitForSubscriptionActivation(authToken);
    
    // Step 6: Success!
    hideLoadingDialog();
    showSuccessMessage('Subscription activated successfully!');
    
    // Refresh subscription status in app
    refreshUserSubscription();
    
  } catch (e) {
    hideLoadingDialog();
    showErrorMessage('Payment failed: ${e.toString()}');
  }
}
```

---

## ⚠️ Important Notes

### 1. Subscription Status Values
- **`pending`**: Created but payment not completed
- **`active`**: Payment successful, subscription active ✅
- **`cancelled`**: User cancelled
- **`expired`**: Subscription period ended
- **`halted`**: Payment failed (may auto-retry)

### 2. Subscription ID Storage
**Before:**
```dart
String subscriptionId = "550e8400-e29b-41d4-a716-446655440000"; // UUID
```

**After:**
```dart
String subscriptionId = "sub_ABC123xyz"; // Razorpay format
```

**Action Required:** Update any code that stores/uses subscription IDs.

### 3. Error Handling

**Common Errors:**
```dart
// 400 - Invalid plan_id
if (response.statusCode == 400) {
  throw Exception('Invalid subscription plan');
}

// 500 - Backend/Razorpay error
if (response.statusCode == 500) {
  final error = response.data['error'] ?? 'Payment gateway error';
  throw Exception(error);
}
```

### 4. UI States

**Show appropriate UI based on status:**
```dart
Widget buildSubscriptionStatus(String status) {
  switch (status) {
    case 'pending':
      return Text('Payment in progress...');
    case 'active':
      return Text('✅ Active');
    case 'cancelled':
      return Text('❌ Cancelled');
    case 'expired':
      return Text('⏰ Expired');
    case 'halted':
      return Text('⚠️ Payment failed');
    default:
      return Text('Unknown status');
  }
}
```

---

## 🧪 Testing Checklist

- [ ] Test with valid plan IDs (`plan_basic_monthly`, `plan_premium_yearly`)
- [ ] Verify subscription ID format is `sub_xxxxx`
- [ ] Test payment link opens correctly
- [ ] Test payment success flow
- [ ] Test payment failure flow
- [ ] Verify subscription status updates after payment
- [ ] Test with test Razorpay credentials
- [ ] Handle network errors gracefully
- [ ] Show appropriate loading states
- [ ] Update UI when subscription becomes active

---

## 📋 Available Plan IDs

Based on your database:
- `plan_basic_monthly` → Razorpay Plan: `plan_Rpb0jZR1OcTK8A`
- `plan_premium_yearly` → Razorpay Plan: `plan_Rpb14tjeOmdtQN`
- `plan_premium_monthly` → (Update in database)
- `plan_basic_yearly` → (Update in database)
- `all_access_30d` → (Update in database)
- `all_access_365d` → (Update in database)

**Note:** Only plans with `razorpay_plan_id` set in database will work.

---

## 🚀 Quick Start Example

```dart
// Complete example
class SubscriptionService {
  final Dio _dio = Dio();
  final String baseUrl = 'https://api.episodd.com';
  
  Future<void> subscribeToPlan({
    required String planId,
    required String authToken,
  }) async {
    try {
      // 1. Create subscription
      final response = await _dio.post(
        '$baseUrl/api/payments/create-subscription',
        options: Options(
          headers: {'Authorization': 'Bearer $authToken'},
        ),
        data: {'plan_id': planId, 'auto_renew': true},
      );
      
      final checkoutUrl = response.data['checkout_url'];
      final subscriptionId = response.data['subscription_id'];
      
      // 2. Open payment
      await launchUrl(Uri.parse(checkoutUrl));
      
      // 3. Wait for activation
      await _pollForActivation(authToken);
      
    } catch (e) {
      throw Exception('Subscription failed: $e');
    }
  }
  
  Future<void> _pollForActivation(String token) async {
    for (int i = 0; i < 20; i++) {
      await Future.delayed(Duration(seconds: 3));
      final check = await _dio.get(
        '$baseUrl/api/subscriptions/check',
        options: Options(
          headers: {'Authorization': 'Bearer $token'},
        ),
      );
      if (check.data['has_access'] == true) return;
    }
    throw Exception('Activation timeout');
  }
}
```

---

## 🆘 Troubleshooting

**Issue:** "The id provided does not exist"
- **Solution:** Plan doesn't have `razorpay_plan_id` set in database. Contact backend team.

**Issue:** Subscription stays "pending"
- **Solution:** Payment not completed or webhook not received. Check Razorpay dashboard.

**Issue:** Payment link doesn't open
- **Solution:** Check `url_launcher` permissions in `AndroidManifest.xml` / `Info.plist`

---

## 📞 Support

If you encounter issues:
1. Check backend logs for Razorpay API errors
2. Verify plan IDs exist in database with `razorpay_plan_id` set
3. Test with Razorpay test credentials first
4. Check webhook configuration in Razorpay dashboard

