-- Quick diagnostic query to check if razorpay_plan_id column exists and has values
-- Run this in your database to diagnose the issue

-- Check if column exists
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'subscription_plans' 
AND column_name = 'razorpay_plan_id';

-- Check plan data
SELECT id, name, razorpay_plan_id, is_active 
FROM subscription_plans 
WHERE id = 'plan_basic_monthly';

-- If razorpay_plan_id is NULL, update it:
-- UPDATE subscription_plans 
-- SET razorpay_plan_id = 'plan_Rpb0jZR1OcTK8A' 
-- WHERE id = 'plan_basic_monthly';

