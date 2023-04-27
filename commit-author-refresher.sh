#!/bin/bash

# Check if the required number of arguments is passed
if [ "$#" -ne 4 ]; then
  echo "Usage: $0 <path-to-original-repo> <new-repo-directory> <new-author-name> <new-author-email>"
  exit 1
fi

# Set script arguments
ORIGINAL_REPO_PATH="$1"
NEW_REPO_DIR="$2"
NEW_AUTHOR_NAME="$3"
NEW_AUTHOR_EMAIL="$4"

# Create a new directory for the new repository and navigate to it
mkdir "$NEW_REPO_DIR"
cd "$NEW_REPO_DIR"

# Initialize an empty Git repository in the new directory
git init

# Add the original repository as a remote
git remote add old-repo "$ORIGINAL_REPO_PATH"

# Fetch all branches and tags from the original repository
git fetch old-repo

# Create a local branch for each branch in the original repository
for branch in $(git branch -r | grep -v 'HEAD' | sed 's/old-repo\///'); do
  git checkout -b "$branch" "old-repo/$branch"
done

# Change the commit author for all commits across all branches
git filter-branch --env-filter '
    export GIT_COMMITTER_NAME="'"$NEW_AUTHOR_NAME"'"
    export GIT_COMMITTER_EMAIL="'"$NEW_AUTHOR_EMAIL"'"
    export GIT_AUTHOR_NAME="'"$NEW_AUTHOR_NAME"'"
    export GIT_AUTHOR_EMAIL="'"$NEW_AUTHOR_EMAIL"'"
' --tag-name-filter cat -- --all

echo "New repository with updated author information created at $NEW_REPO_DIR"
