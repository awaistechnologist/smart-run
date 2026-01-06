#!/bin/bash
set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Navigate to the project root (one level up from scripts/)
cd "$SCRIPT_DIR/.."

echo "🚀 SmartRun Demo"
echo "================"
echo

echo "✅ Step 1: Build the binary"
go build -o smart-run ./cmd/smart-run
echo

echo "✅ Step 2: Initialize household"
./smart-run init
echo

echo "✅ Step 3: Add appliances"
./smart-run appliance add --name "Dishwasher" --cycle 60 --kwh 1.2 --noise 3
./smart-run appliance add --name "Washing Machine" --cycle 75 --kwh 0.9 --noise 4
./smart-run appliance add --name "EV Charger" --cycle 240 --kwh 10 --noise 1
echo

echo "✅ Step 4: List appliances"
./smart-run appliance list
echo

echo "✅ Step 5: Fetch Octopus Agile prices (Region C = London)"
echo "Fetching prices for today and tomorrow..."
./smart-run fetch --region C --date today | head -10
echo "... (showing first 10 slots)"
echo

echo "✅ Step 6: Generate optimal schedule"
echo "Finding cheapest times to run each appliance..."
./smart-run plan --region C | jq '.'
echo

echo "🎉 Demo complete!"
echo
echo "Key findings:"
echo "- Negative prices mean you GET PAID to use electricity!"
echo "- Cheapest times are usually overnight (high wind generation)"
echo "- All recommendations respect quiet hours (22:00-07:00)"
echo
echo "Try it yourself:"
echo "  ./smart-run plan --region C"
