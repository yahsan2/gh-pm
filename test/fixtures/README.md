# Test Fixtures

This directory contains test data and sample files for testing gh-pm commands.

## Files

### split_command_sample_tasks.md
Sample task list for testing the `gh pm split` command. Contains GitHub-style checkboxes that can be extracted and converted into sub-issues.

**Usage example:**
```bash
# Test split command with this file
gh pm split 123 --from=test/fixtures/split_command_sample_tasks.md

# Test dry-run mode
gh pm split 123 --from=test/fixtures/split_command_sample_tasks.md --dry-run
```