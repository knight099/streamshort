-- Add razorpay_plan_id column to subscription_plans table
ALTER TABLE subscription_plans 
ADD COLUMN IF NOT EXISTS razorpay_plan_id VARCHAR(255);

-- Update existing plans with placeholder (you need to replace these with actual Razorpay plan IDs)
-- IMPORTANT: Replace 'plan_xxxxx' with your actual Razorpay plan IDs from Razorpay Dashboard
UPDATE subscription_plans 
SET razorpay_plan_id = CASE 
    WHEN id = 'plan_basic_monthly' THEN 'plan_Rpb0jZR1OcTK8A'  -- Replace with actual Razorpay plan ID
    WHEN id = 'plan_premium_monthly' THEN 'plan_xxxxx'  -- Replace with actual Razorpay plan ID
    WHEN id = 'plan_basic_yearly' THEN 'plan_xxxxx'  -- Replace with actual Razorpay plan ID
    WHEN id = 'plan_premium_yearly' THEN 'plan_Rpb14tjeOmdtQN'  -- Replace with actual Razorpay plan ID
    WHEN id = 'all_access_30d' THEN 'plan_xxxxx'  -- Replace with actual Razorpay plan ID
    WHEN id = 'all_access_365d' THEN 'plan_xxxxx'  -- Replace with actual Razorpay plan ID
    ELSE NULL
END
WHERE razorpay_plan_id IS NULL;

