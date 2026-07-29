# Mintlify Redirects Guide

A comprehensive guide for managing redirects in Mintlify documentation (migrated from Nextra).

---

## Table of Contents

- [Overview](#overview)
- [How to add redirects](#how-to-add-redirects)
- [Examples](#examples)
- [Limitations](#limitations)

---

## Overview

Redirects are essential when restructuring documentation (IA refactor) to ensure:

- Old links don't break
- Search engine rankings are preserved
- User bookmarks continue to work
- External links remain valid

**Location**: All Mintlify redirects are configured in `/docs.json`

---

## How to add redirects

### Step 1: Open `docs.json`

Navigate to the root of the project and open `/docs.json`.

### Step 2: Find the `redirects` array

Look for the `redirects` section (around line 33):

```json
{
  "$schema": "https://mintlify.com/docs.json",
  "theme": "mint",
  "name": "Optimism Documentation",
  ...
  "redirects": [
    // Redirects go here
  ],
  ...
}
```

### Step 3: Add your redirect

Add a new object to the array:

```json
{
  "redirects": [
    {
      "source": "/old-page-path",
      "destination": "/new-page-path"
    }
  ]
}
```

### Step 4: Test locally

```bash
# Run the dev server
mint dev

# Test the redirect
# Navigate to http://localhost:3000/old-page-path
# Should redirect to /new-page-path
```

---

## Examples

### Example 1: Simple page redirect

**Scenario**: You renamed `getting-started.mdx` to `quickstart.mdx`

```json
{
  "source": "/getting-started",
  "destination": "/quickstart"
}
```

### Example 2: Section restructure

**Scenario**: Moved all tutorials from `/docs/` to `/tutorials/`

```json
{
  "source": "/docs/deploy-contract",
  "destination": "/tutorials/deploy-contract"
},
{
  "source": "/docs/setup-wallet",
  "destination": "/tutorials/setup-wallet"
},
{
  "source": "/docs/bridge-tokens",
  "destination": "/tutorials/bridge-tokens"
}
```

**Note**: You need one redirect per page (no wildcards!)

### Example 3: Deep restructure

**Scenario**: Moved `app-developers/tools/supersim.mdx` to `interop/tools/supersim.mdx`

```json
{
  "source": "/app-developers/tools/supersim",
  "destination": "/interop/tools/supersim"
}
```

### Example 4: Deleted page

**Scenario**: Deleted a page and want to redirect to a related page

```json
{
  "source": "/deprecated-feature",
  "destination": "/new-feature-overview"
}
```

### Example 5: External redirect

**Scenario**: Moved content to external documentation

```json
{
  "source": "/old-specs",
  "destination": "https://specs.optimism.io"
}
```

---

## Limitations

### No Wildcard support

**Nextra** (supported):

```
/docs/* /tutorials/:splat 301
```

**Mintlify** (NOT supported):

```json
{
  "source": "/docs/*",
  "destination": "/tutorials/*"
}
```

**Workaround**: Create individual redirects for each page:

```json
{
  "redirects": [
    { "source": "/docs/page1", "destination": "/tutorials/page1" },
    { "source": "/docs/page2", "destination": "/tutorials/page2" },
    { "source": "/docs/page3", "destination": "/tutorials/page3" }
  ]
}
```
