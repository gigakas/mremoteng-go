# Runbooks

A runbook is an operational procedure for a person under pressure. It must be specific enough to execute at 3 a.m., by someone who did not write the system.

## When to write one

- Deploying or rolling back a service.
- Running a migration or backfill.
- Responding to a recurring incident (queue backlog, failed scheduler, disk full).
- Rotating secrets or certificates.
- Restoring from backup.
- On-call escalation.

If you have explained the same procedure twice in chat, write a runbook.

## Structure

| Section | Content |
|---------|---------|
| **Purpose** | What this runbook does and when to use it |
| **Scope** | Services/sites/environments covered |
| **Prerequisites** | Access, tools, approvals, maintenance window |
| **Health checks** | How to confirm the system is healthy before/after |
| **Procedure** | Numbered commands with expected output |
| **Rollback** | Exact steps to restore the previous state |
| **Common incidents** | Symptom → diagnosis → fix |
| **Escalation** | Who to contact, with what information |
| **Post-incident** | Logs/metrics to capture, postmortem link |

## Template

```markdown
# Runbook: Deploy <service>

## Purpose
Deploy version X to production safely.

## Scope
Production site `<site>`, app `<app>`.

## Prerequisites
- Access to the bastion / bench host.
- Approval from the release owner.
- Backup completed in the last 24h.

## Health checks (before)
```bash
curl -fsS https://<host>/api/method/ping
bench --site <site> doctor
```

## Procedure
1. Announce the maintenance window.
2. Take a backup:
   ```bash
   bench --site <site> backup --with-files
   ```
3. Enable maintenance mode (optional):
   ```bash
   bench --site <site> set-maintenance-mode on
   ```
4. Deploy:
   ```bash
   bench --site <site> migrate
   bench build
   bench restart
   ```
5. Verify:
   ```bash
   curl -fsS https://<host>/api/method/ping
   bench --site <site> doctor
   ```
6. Disable maintenance mode.

## Rollback
1. Restore the pre-deploy backup:
   ```bash
   bench --site <site> restore <backup-file> --force
   ```
2. Re-deploy the previous commit/tag.
3. Run the health checks.

## Common incidents

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| 502 after deploy | Workers not restarted | `bench restart` |
| Migration error | Patch failure | Read traceback, fix patch, re-run migrate |
| Stale UI | Cache | `bench --site <site> clear-cache`, hard reload |

## Escalation
- Primary: <name/contact>
- Secondary: <name/contact>
- Include: site, version, command, full error, timestamps.

## Post-incident
- Capture logs and metrics.
- Open a postmortem if user impact > 15 min.
```

## Rules

- **Commands must be copy-pasteable**; no placeholders unless clearly marked.
- **Every destructive step has a backup/rollback** before it.
- **State the expected output** so the operator knows it worked.
- **Do not assume the operator knows the system**; link to architecture/ADRs.
- **Keep it current**: update it in the same PR that changes the procedure.
- **Test the runbook** in staging at least once; an untested runbook is a guess.

## Checklist

- [ ] Purpose, scope, prerequisites.
- [ ] Health checks before and after.
- [ ] Numbered procedure with expected output.
- [ ] Rollback with exact commands.
- [ ] Common incidents and fixes.
- [ ] Escalation path.
- [ ] Post-incident capture.
- [ ] Tested in a non-production environment.
- [ ] Linked from the README.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| "Run the deploy script" | No detail under pressure | List the exact commands |
| No rollback | Incident gets worse | Add restore/deploy-previous steps |
| No health checks | Cannot tell if it worked | Add before/after checks |
| Secrets in the runbook | Leak risk | Reference a secret manager |
| Never tested | Fails when needed | Drill in staging |
| Outdated screenshots | Misleading | Regenerate or remove |
