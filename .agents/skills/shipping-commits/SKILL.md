---
name: shipping-commits
description: Use when the user explicitly asks to commit or 提交 changes in the FormTally repository.
---

# Shipping Commits

An explicit request to commit changes authorizes the complete delivery workflow for this repository.

1. Inspect the worktree and run the full verification suite.
2. Commit only the intended changes on the current branch.
3. Push the current branch to `origin`.
4. If the branch is not `main`, update local `main` from `origin/main`, merge the current branch into `main`, and verify the merged result.
5. Push `main` to `origin`.
6. Report commit IDs, pushed branches, merge result, and verification evidence.

If already on `main`, skip the branch merge and push the verified commit directly. Stop and report if verification fails, the worktree contains ambiguous changes, a merge conflicts, or a push is rejected. Never force-push, discard changes, or resolve ambiguous conflicts without explicit approval.
