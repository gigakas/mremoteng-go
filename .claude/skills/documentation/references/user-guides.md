# User Guides

A user guide is **task-oriented**, not feature-oriented. It starts from what the user wants to accomplish, not from the menu structure.

## When to write one

- A user-facing feature needs onboarding.
- Support keeps answering the same question.
- A workflow has prerequisites or pitfalls.
- A release changes how users do something.

## Structure

| Section | Content |
|---------|---------|
| **Title** | The task: "Export invoices to CSV" |
| **Who this is for** | Role, plan, permissions |
| **Before you start** | Prerequisites, data, access |
| **Steps** | Numbered, one action per step |
| **Result** | What success looks like |
| **Troubleshooting** | Common errors and fixes |
| **Related** | Links to concepts/reference |

## Template

```markdown
# Export invoices to CSV

Applies to: Finance users with the `Reports` role.

## Before you start
- At least one invoice in the selected period.
- Access to **Reports → Invoices**.

## Steps
1. Open **Reports → Invoices**.
2. Select the date range.
3. Click **Export → CSV**.
4. Choose a destination and confirm.

## Result
A `.csv` file downloads with one row per invoice.

## Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| Export button disabled | No invoices in range | Change the date range |
| File is empty | Filters exclude all rows | Clear filters and retry |

## Related
- [Invoice concepts](../concepts/invoices.md)
```

## Writing rules

- **One task per guide.** Split multi-task pages.
- **Use the user's vocabulary**, not internal names.
- **Number the steps**; one action per step.
- **Show the expected result** after the steps.
- **Put warnings before the action**, not after.
- **Use real screenshots** with annotations; update them when the UI changes.
- **Make it accessible**: alt text, not color-only cues, readable contrast.
- **Version it** if the product has multiple supported versions.

## Troubleshooting section

List the errors users actually hit:

| Problem | Cause | Fix |
|---------|-------|-----|
| ... | ... | ... |

Link to support/escalation for issues the guide cannot solve.

## Checklist

- [ ] Title is a task, not a feature name.
- [ ] Audience and prerequisites stated.
- [ ] Steps numbered and copy-pasteable where applicable.
- [ ] Expected result shown.
- [ ] Troubleshooting covers real errors.
- [ ] Screenshots current and accessible.
- [ ] Links resolve.
- [ ] Reviewed by someone who did not write the feature.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| Feature tour | Users cannot find their task | Rewrite around tasks |
| Wall of text | Unscannable | Steps, headings, tables |
| No prerequisites | User gets stuck | State access/data first |
| Outdated screenshots | Misleading | Regenerate in the same release |
| No troubleshooting | Support burden | Add common errors |
| Jargon | Users do not understand | Use domain language |
