# Optimism Issue Templates

This directory contains the issue templates for the Optimism repository. These templates help standardize the format of issues and enable automatic labeling for better organization and triage.

## Available Templates

1. **Bug Report (`BUG-FORM.yml`)** - For reporting bugs and unexpected behavior
2. **Feature Request (`FEATURE-FORM.yml`)** - For suggesting new features or improvements
3. **Documentation Issue (`DOCUMENTATION-FORM.yml`)** - For reporting documentation problems

## Automatic Labeling

The issue templates are set up to automatically apply labels based on the selections made in the form. This helps with:

1. **Area Labeling (`A-*`)** - Identifying which part of the codebase the issue relates to
2. **Team Labeling (`T-*`)** - Routing the issue to the appropriate team
3. **Category Labeling (`C-*`)** - Classifying the type of issue
4. **Meta Labeling (`M-*`)** - Additional information about priority or impact

## How It Works

1. When a new issue is created, the user selects a template
2. The form guides them to provide all necessary information
3. Upon submission, GitHub Actions automatically applies appropriate labels based on the form responses
4. Team members can focus on issues relevant to them by filtering by labels

## Adding or Modifying Templates

When making changes to the templates:

1. Follow the YAML format for GitHub issue forms
2. Update the labeler workflow as needed to ensure labels are correctly applied
3. Test the template by creating a test issue

## Available Labels Reference

### Area Labels (A-*)
- `A-node` - Node related issues
- `A-batcher` - Batcher related issues
- `A-proposer` - Proposer related issues
- `A-challenger` - Challenger related issues
- `A-devnet` - Devnet related issues
- `A-contracts` - Smart Contracts related issues
- `A-ops` - Operations related issues
- `A-docs` - Documentation related issues

### Team Labels (T-*)
- `T-node` - Node team
- `T-protocol` - Protocol team
- `T-smart-contracts` - Smart Contracts team
- `T-platforms` - Platforms team
- `T-community` - Community team
- `T-devex` - Developer Experience team

### Category Labels (C-*)
- `C-bug` - Bug reports
- `C-feature` - Feature requests
- `C-documentation` - Documentation issues
- `C-good first issue` - Issues suitable for first-time contributors

### Meta Labels (M-*)
- `M-high-impact` - High community impact
- `M-medium-impact` - Medium community impact
- `M-low-impact` - Low community impact
- `M-community` - Community related
