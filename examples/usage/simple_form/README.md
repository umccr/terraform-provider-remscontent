# Simple Example — Creating Forms

This example demonstrates how to create **REMS Forms** using the `remscontent`
Terraform provider. It covers common field types as well as conditional field
visibility.

## What This Example Does

Two forms are created:

### `example_001` — Basic Form

A minimal form demonstrating the most common field types.

| Field ID | Title       | Type    |
|----------|-------------|---------|
| `fld1`   | Title       | `text`  |
| `fld2`   | Email       | `email` |
| `fld3`   | Description | `texta` |

### `test_form` — Advanced Form with Conditional Visibility

A more comprehensive form showcasing additional field types and a conditional
visibility rule — the **Attachment** field is only shown when the user selects
**No** for "Attach a file?".

| Field ID      | Title          | Type           | Notes |
|---------------|----------------|----------------|-------|
| `fld1`        | Title          | `text`         | |
| `fld2`        | Phone Number   | `phone-number` | |
| `fld3`        | Email          | `email`        | |
| `fld4`        | Date           | `date`         | |
| `attaching-5` | Attach a file? | `option`       | Drives visibility of `fld6` |
| `fld6`        | Attachment     | `attachment`   | Visible only when `attaching-5 = "n"` |
