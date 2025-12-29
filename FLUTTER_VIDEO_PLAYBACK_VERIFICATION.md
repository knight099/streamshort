# Flutter Video Playback Verification Guide

## Overview
This guide helps you verify that your Flutter app correctly fetches and plays videos from CloudFront signed URLs.

## Current Flow

1. **Fetch Episode Manifest**:
   ```
   GET /api/episodes/{episode_id}/manifest
   Authorization: Bearer {token}
   ```

2. **Response**:
   ```json
   {
     "manifest_url": "https://djoxr6aauqbf6.cloudfront.net/...",
     "expires_at": "2025-12-29T12:00:00Z"
   }
   ```

3. **Play Video**: Use the `manifest_url` in your video player

## Verification Steps for Flutter

### Step 1: Add Detailed Logging

Add comprehensive logging to track the entire video loading process:

```dart
// In your video player service or widget

Future<void> loadVideo(String episodeId) async {
  try {
    print('🎬 [Video] Starting video load for episode: $episodeId');
    
    // Step 1: Fetch manifest URL
    print('📡 [Video] Fetching manifest URL from API...');
    final response = await dio.get(
      '/api/episodes/$episodeId/manifest',
      options: Options(
        headers: {'Authorization': 'Bearer $token'},
      ),
    );
    
    print('✅ [Video] Manifest response received');
    print('📋 [Video] Status: ${response.statusCode}');
    print('📋 [Video] Response: ${response.data}');
    
    final manifestUrl = response.data['manifest_url'] as String;
    final expiresAt = response.data['expires_at'] as String;
    
    print('🔗 [Video] Manifest URL: $manifestUrl');
    print('⏰ [Video] Expires at: $expiresAt');
    
    // Step 2: Verify URL format
    if (!manifestUrl.contains('Signature=') || !manifestUrl.contains('Key-Pair-Id=')) {
      print('⚠️ [Video] WARNING: URL doesn\'t look like a signed URL!');
    } else {
      print('✅ [Video] URL appears to be properly signed');
    }
    
    // Step 3: Test URL accessibility (optional but recommended)
    print('🔍 [Video] Testing URL accessibility...');
    final testResponse = await dio.head(
      manifestUrl,
      options: Options(
        validateStatus: (status) => true, // Accept any status
        followRedirects: true,
      ),
    );
    
    print('📊 [Video] URL test status: ${testResponse.statusCode}');
    
    if (testResponse.statusCode == 200) {
      print('✅ [Video] URL is accessible!');
    } else if (testResponse.statusCode == 404) {
      print('⚠️ [Video] File not found (might not be transcoded yet)');
    } else if (testResponse.statusCode == 403) {
      print('❌ [Video] Access denied! CloudFront configuration issue.');
      print('🔧 [Video] Check: CLOUDFRONT_403_TROUBLESHOOTING.md');
    } else {
      print('⚠️ [Video] Unexpected status: ${testResponse.statusCode}');
    }
    
    // Step 4: Initialize video player
    print('🎥 [Video] Initializing video player...');
    await _initializePlayer(manifestUrl);
    
    print('✅ [Video] Video player initialized successfully');
    
  } catch (e, stackTrace) {
    print('❌ [Video] Error loading video: $e');
    print('📚 [Video] Stack trace: $stackTrace');
    
    if (e is DioException) {
      print('🔍 [Video] DioException details:');
      print('   - Status code: ${e.response?.statusCode}');
      print('   - Response data: ${e.response?.data}');
      print('   - Headers: ${e.response?.headers}');
    }
    
    rethrow;
  }
}
```

### Step 2: Add URL Validation

Create a helper function to validate signed URLs:

```dart
class VideoUrlValidator {
  static bool isValidSignedUrl(String url) {
    // Check if URL contains required CloudFront signed URL parameters
    final hasSignature = url.contains('Signature=');
    final hasKeyPairId = url.contains('Key-Pair-Id=');
    final hasExpires = url.contains('Expires=');
    
    print('🔍 [Validation] URL validation:');
    print('   - Has Signature: $hasSignature');
    print('   - Has Key-Pair-Id: $hasKeyPairId');
    print('   - Has Expires: $hasExpires');
    
    return hasSignature && hasKeyPairId && hasExpires;
  }
  
  static bool isExpired(String expiresAt) {
    try {
      final expiryTime = DateTime.parse(expiresAt);
      final now = DateTime.now();
      final isExpired = now.isAfter(expiryTime);
      
      print('⏰ [Validation] Expiry check:');
      print('   - Expires at: $expiryTime');
      print('   - Current time: $now');
      print('   - Is expired: $isExpired');
      
      return isExpired;
    } catch (e) {
      print('❌ [Validation] Error parsing expiry time: $e');
      return false;
    }
  }
  
  static Duration getTimeUntilExpiry(String expiresAt) {
    try {
      final expiryTime = DateTime.parse(expiresAt);
      final now = DateTime.now();
      final duration = expiryTime.difference(now);
      
      print('⏱️ [Validation] Time until expiry: ${duration.inMinutes} minutes');
      
      return duration;
    } catch (e) {
      print('❌ [Validation] Error calculating expiry: $e');
      return Duration.zero;
    }
  }
}
```

### Step 3: Add Error Handling UI

Show user-friendly error messages:

```dart
class VideoErrorHandler {
  static String getErrorMessage(int? statusCode, dynamic error) {
    if (statusCode == 403) {
      return 'Video access denied. Please try again in a few minutes.';
    } else if (statusCode == 404) {
      return 'Video is still being processed. Please try again shortly.';
    } else if (statusCode == null && error.toString().contains('SocketException')) {
      return 'Network error. Please check your internet connection.';
    } else {
      return 'Failed to load video. Please try again.';
    }
  }
  
  static void showErrorDialog(BuildContext context, int? statusCode, dynamic error) {
    final message = getErrorMessage(statusCode, error);
    
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Video Error'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(message),
            if (statusCode != null) ...[
              const SizedBox(height: 8),
              Text(
                'Error code: $statusCode',
                style: TextStyle(
                  fontSize: 12,
                  color: Colors.grey[600],
                ),
              ),
            ],
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('OK'),
          ),
          if (statusCode == 403 || statusCode == 404)
            TextButton(
              onPressed: () {
                Navigator.pop(context);
                // Retry loading video
              },
              child: const Text('Retry'),
            ),
        ],
      ),
    );
  }
}
```

### Step 4: Add Retry Logic

Implement automatic retry for transient errors:

```dart
class VideoLoader {
  static const maxRetries = 3;
  static const retryDelay = Duration(seconds: 2);
  
  Future<String> fetchManifestUrlWithRetry(String episodeId) async {
    int attempts = 0;
    
    while (attempts < maxRetries) {
      try {
        attempts++;
        print('🔄 [Video] Attempt $attempts of $maxRetries');
        
        final response = await dio.get('/api/episodes/$episodeId/manifest');
        final manifestUrl = response.data['manifest_url'] as String;
        
        // Verify URL before returning
        if (!VideoUrlValidator.isValidSignedUrl(manifestUrl)) {
          throw Exception('Invalid signed URL format');
        }
        
        print('✅ [Video] Successfully fetched manifest URL');
        return manifestUrl;
        
      } catch (e) {
        print('❌ [Video] Attempt $attempts failed: $e');
        
        if (attempts >= maxRetries) {
          print('💥 [Video] All retry attempts exhausted');
          rethrow;
        }
        
        if (e is DioException && e.response?.statusCode == 403) {
          print('⏳ [Video] 403 error - waiting before retry...');
          await Future.delayed(retryDelay * attempts); // Exponential backoff
        } else {
          await Future.delayed(retryDelay);
        }
      }
    }
    
    throw Exception('Failed to fetch manifest URL after $maxRetries attempts');
  }
}
```

### Step 5: Add Network Diagnostics

Create a diagnostic tool to test connectivity:

```dart
class VideoDiagnostics {
  static Future<Map<String, dynamic>> runDiagnostics(String episodeId) async {
    final results = <String, dynamic>{};
    
    print('🔬 [Diagnostics] Starting video diagnostics...');
    
    // Test 1: API connectivity
    try {
      print('🔬 [Diagnostics] Test 1: API connectivity');
      final response = await dio.get('/health');
      results['api_connectivity'] = response.statusCode == 200;
      print('✅ [Diagnostics] API is reachable');
    } catch (e) {
      results['api_connectivity'] = false;
      print('❌ [Diagnostics] API is not reachable: $e');
    }
    
    // Test 2: Authentication
    try {
      print('🔬 [Diagnostics] Test 2: Authentication');
      final response = await dio.get('/api/profile');
      results['authentication'] = response.statusCode == 200;
      print('✅ [Diagnostics] Authentication is valid');
    } catch (e) {
      results['authentication'] = false;
      print('❌ [Diagnostics] Authentication failed: $e');
    }
    
    // Test 3: Manifest endpoint
    try {
      print('🔬 [Diagnostics] Test 3: Manifest endpoint');
      final response = await dio.get('/api/episodes/$episodeId/manifest');
      results['manifest_endpoint'] = response.statusCode == 200;
      results['manifest_url'] = response.data['manifest_url'];
      print('✅ [Diagnostics] Manifest endpoint is working');
    } catch (e) {
      results['manifest_endpoint'] = false;
      print('❌ [Diagnostics] Manifest endpoint failed: $e');
    }
    
    // Test 4: CloudFront URL accessibility
    if (results['manifest_url'] != null) {
      try {
        print('🔬 [Diagnostics] Test 4: CloudFront URL accessibility');
        final response = await dio.head(
          results['manifest_url'],
          options: Options(validateStatus: (status) => true),
        );
        results['cloudfront_status'] = response.statusCode;
        results['cloudfront_accessible'] = response.statusCode == 200 || response.statusCode == 404;
        print('✅ [Diagnostics] CloudFront returned: ${response.statusCode}');
      } catch (e) {
        results['cloudfront_accessible'] = false;
        print('❌ [Diagnostics] CloudFront test failed: $e');
      }
    }
    
    print('🔬 [Diagnostics] Diagnostics complete');
    print('📊 [Diagnostics] Results: $results');
    
    return results;
  }
  
  static void printDiagnosticReport(Map<String, dynamic> results) {
    print('');
    print('═══════════════════════════════════════');
    print('📊 VIDEO DIAGNOSTICS REPORT');
    print('═══════════════════════════════════════');
    print('API Connectivity:     ${results['api_connectivity'] ? '✅' : '❌'}');
    print('Authentication:       ${results['authentication'] ? '✅' : '❌'}');
    print('Manifest Endpoint:    ${results['manifest_endpoint'] ? '✅' : '❌'}');
    print('CloudFront Status:    ${results['cloudfront_status'] ?? 'N/A'}');
    print('CloudFront Access:    ${results['cloudfront_accessible'] ? '✅' : '❌'}');
    print('═══════════════════════════════════════');
    print('');
    
    if (results['cloudfront_status'] == 403) {
      print('⚠️ CloudFront is returning 403 Forbidden');
      print('💡 This means:');
      print('   1. CloudFront configuration is still propagating (wait 10-20 min)');
      print('   2. OR signed URL configuration needs to be fixed');
      print('   3. Check: CLOUDFRONT_403_TROUBLESHOOTING.md');
    } else if (results['cloudfront_status'] == 404) {
      print('⚠️ CloudFront is returning 404 Not Found');
      print('💡 This means:');
      print('   1. Video hasn\'t been transcoded yet');
      print('   2. OR the file path is incorrect');
      print('   3. Wait for transcoding to complete');
    } else if (results['cloudfront_status'] == 200) {
      print('✅ Everything looks good!');
      print('💡 Video should play successfully');
    }
  }
}
```

### Step 6: Add Debug Screen (Optional)

Create a debug screen to test video loading:

```dart
class VideoDebugScreen extends StatefulWidget {
  final String episodeId;
  
  const VideoDebugScreen({required this.episodeId});
  
  @override
  State<VideoDebugScreen> createState() => _VideoDebugScreenState();
}

class _VideoDebugScreenState extends State<VideoDebugScreen> {
  Map<String, dynamic>? diagnosticResults;
  bool isRunning = false;
  
  Future<void> runDiagnostics() async {
    setState(() => isRunning = true);
    
    try {
      final results = await VideoDiagnostics.runDiagnostics(widget.episodeId);
      setState(() {
        diagnosticResults = results;
        isRunning = false;
      });
      
      VideoDiagnostics.printDiagnosticReport(results);
    } catch (e) {
      print('❌ Diagnostics failed: $e');
      setState(() => isRunning = false);
    }
  }
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Video Debug')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Episode ID: ${widget.episodeId}'),
            const SizedBox(height: 16),
            
            ElevatedButton(
              onPressed: isRunning ? null : runDiagnostics,
              child: Text(isRunning ? 'Running...' : 'Run Diagnostics'),
            ),
            
            const SizedBox(height: 24),
            
            if (diagnosticResults != null) ...[
              const Text('Results:', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              
              _buildResultRow('API Connectivity', diagnosticResults!['api_connectivity']),
              _buildResultRow('Authentication', diagnosticResults!['authentication']),
              _buildResultRow('Manifest Endpoint', diagnosticResults!['manifest_endpoint']),
              _buildResultRow('CloudFront Access', diagnosticResults!['cloudfront_accessible']),
              
              if (diagnosticResults!['cloudfront_status'] != null) ...[
                const SizedBox(height: 8),
                Text('CloudFront Status: ${diagnosticResults!['cloudfront_status']}'),
              ],
              
              if (diagnosticResults!['manifest_url'] != null) ...[
                const SizedBox(height: 16),
                const Text('Manifest URL:', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 4),
                SelectableText(
                  diagnosticResults!['manifest_url'],
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }
  
  Widget _buildResultRow(String label, bool? value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Text('$label: '),
          Icon(
            value == true ? Icons.check_circle : Icons.error,
            color: value == true ? Colors.green : Colors.red,
            size: 20,
          ),
        ],
      ),
    );
  }
}
```

## Testing Checklist

Use this checklist to verify video playback:

```dart
// Add this to your test suite or run manually

void testVideoPlayback() async {
  print('🧪 Starting video playback test...');
  
  // Test 1: Fetch manifest URL
  print('\n📋 Test 1: Fetch manifest URL');
  final manifestUrl = await fetchManifestUrl('test-episode-id');
  assert(manifestUrl.isNotEmpty, 'Manifest URL should not be empty');
  print('✅ Test 1 passed');
  
  // Test 2: Validate URL format
  print('\n📋 Test 2: Validate URL format');
  assert(VideoUrlValidator.isValidSignedUrl(manifestUrl), 'URL should be a valid signed URL');
  print('✅ Test 2 passed');
  
  // Test 3: Check URL accessibility
  print('\n📋 Test 3: Check URL accessibility');
  final response = await dio.head(manifestUrl);
  assert(response.statusCode == 200 || response.statusCode == 404, 'URL should be accessible');
  print('✅ Test 3 passed');
  
  // Test 4: Initialize player
  print('\n📋 Test 4: Initialize player');
  final player = VideoPlayerController.network(manifestUrl);
  await player.initialize();
  assert(player.value.isInitialized, 'Player should initialize');
  print('✅ Test 4 passed');
  
  print('\n🎉 All tests passed!');
}
```

## Common Issues and Solutions

### Issue 1: 403 Forbidden
**Symptoms**: Video fails to load, logs show 403 status
**Solution**: 
- CloudFront configuration is still propagating (wait 10-20 minutes)
- Check `CLOUDFRONT_403_TROUBLESHOOTING.md`

### Issue 2: 404 Not Found
**Symptoms**: Video fails to load, logs show 404 status
**Solution**:
- Video hasn't been transcoded yet
- Wait for transcoding to complete
- Check if `hls_manifest_url` is populated in database

### Issue 3: URL Expired
**Symptoms**: Video plays initially but fails after some time
**Solution**:
- Implement URL refresh logic
- Fetch new manifest URL when current one is about to expire

### Issue 4: Network Timeout
**Symptoms**: Request hangs or times out
**Solution**:
- Check internet connectivity
- Increase timeout duration
- Implement retry logic

## Production Recommendations

1. **Remove verbose logging** in production builds
2. **Implement analytics** to track video load success/failure rates
3. **Add user feedback** for long loading times
4. **Cache manifest URLs** (but respect expiry times)
5. **Preload next video** for better UX
6. **Monitor error rates** and set up alerts

## Quick Test Command

Add this button to your debug menu:

```dart
ElevatedButton(
  onPressed: () async {
    final results = await VideoDiagnostics.runDiagnostics(episodeId);
    VideoDiagnostics.printDiagnosticReport(results);
  },
  child: const Text('Test Video Loading'),
)
```

This will help you quickly identify if the issue is with:
- API connectivity
- Authentication
- Manifest endpoint
- CloudFront configuration
- Network issues

## Next Steps

1. Add the logging code to your video player
2. Test with a known episode ID
3. Check the logs for any errors
4. Use the diagnostic tool to identify issues
5. Fix any problems found
6. Test video playback in the app

Once CloudFront propagation is complete (10-20 minutes), videos should play successfully!
