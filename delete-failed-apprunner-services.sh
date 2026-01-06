#!/bin/bash

# Delete all failed App Runner services
echo "Deleting failed App Runner services..."

# Failed services to delete
FAILED_SERVICES=(
    "arn:aws:apprunner:ap-south-1:462977978299:service/episodd-minimal-test/50729a2b440b44d29226f1555cd9e073"
    "arn:aws:apprunner:ap-south-1:462977978299:service/episodd-app-v3/793483e2c06b44e283199c87aba11a81"
    "arn:aws:apprunner:ap-south-1:462977978299:service/episodd-app-v2/3ab575478c964f49b44e855968b4daf4"
    "arn:aws:apprunner:ap-south-1:462977978299:service/episodd-app/c199e6282e384892abd18f114211a951"
)

for service_arn in "${FAILED_SERVICES[@]}"; do
    echo "Deleting service: $service_arn"
    aws apprunner delete-service --service-arn "$service_arn" --region ap-south-1
    if [ $? -eq 0 ]; then
        echo "✓ Successfully initiated deletion for $service_arn"
    else
        echo "✗ Failed to delete $service_arn"
    fi
    echo "---"
done

echo "All failed services deletion initiated. Note: Deletion is asynchronous and may take a few minutes."
echo ""
echo "Remaining services:"
aws apprunner list-services --region ap-south-1 --query 'ServiceSummaryList[*].[ServiceName,Status]' --output table