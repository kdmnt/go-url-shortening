#!/bin/sh

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

# Copy pre-push hook
cp pre-push .git/hooks/pre-push

# Make the hook executable
chmod +x .git/hooks/pre-push

echo "Git hooks have been set up successfully."
