#!/bin/sh
# Simple test to check task file access

echo "Creating nodejs task..."
TASK_ID=$(cat /task/new/nodejs)
echo "Got task ID: $TASK_ID"

echo "Checking if task directory exists..."
ls -la /task/ | grep "$TASK_ID" || echo "Task $TASK_ID not in listing!"

echo "Trying to list task directory..."
ls /task/$TASK_ID 2>&1 || echo "Cannot list /task/$TASK_ID"

echo "Trying to read cmd file..."
cat /task/$TASK_ID/cmd 2>&1 || echo "Cannot read cmd"

echo "Trying to write with echo..."
echo "test" > /task/$TASK_ID/cmd 2>&1 || echo "Cannot write with echo"

echo "Done"