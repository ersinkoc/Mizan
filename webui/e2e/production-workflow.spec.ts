import { expect, test } from '@playwright/test';

test('operator can import, edit, validate, request approval, preview deploy, audit, and monitor', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'No project selected' })).toBeVisible();

  const importForm = page.locator('form.import-project');
  await importForm.getByLabel('Imported project name').fill('e2e-imported-edge');
  await importForm.getByLabel('Config filename').fill('haproxy.cfg');
  await importForm.getByLabel('Config text').fill([
    'frontend web',
    '  bind :80',
    '  default_backend app',
    'backend app',
    '  balance roundrobin',
    '  server app1 127.0.0.1:8080 check'
  ].join('\n'));
  await importForm.getByRole('button', { name: /Import/ }).click();

  await expect(page.getByRole('heading', { name: 'e2e-imported-edge' })).toBeVisible();
  await expect(page.locator('.editor textarea')).toHaveValue(/fe_web/);
  await expect(page.locator('.audit-list')).toContainText('project.import');

  await page.getByRole('button', { name: /Sample/ }).click();
  await expect(page.locator('.editor textarea')).toHaveValue(/app-pool/);
  await expect(page.locator('.metrics')).toContainText('Frontends');
  await expect(page.locator('.metrics')).toContainText('Backends');

  await page.getByRole('button', { name: /Validate/ }).click();
  await expect(page.locator('.config-preview')).toContainText('frontend web');
  await expect(page.locator('.audit-list')).toContainText('config.validate');

  const targetForm = page.locator('form.target-form');
  await targetForm.getByLabel('Target name').fill('prod-a');
  await targetForm.getByLabel('Target host').fill('127.0.0.1');
  await targetForm.getByLabel('SSH user').fill('root');
  await targetForm.getByLabel('SSH port').fill('22');
  await targetForm.getByLabel('Target engine').selectOption('haproxy');
  await targetForm.getByLabel('Rollback command').fill('cp /etc/haproxy/haproxy.cfg.bak /etc/haproxy/haproxy.cfg && systemctl reload haproxy');
  await targetForm.getByRole('button', { name: /Add Target/ }).click();

  const targetCard = page.locator('.target-card').filter({ hasText: 'prod-a' });
  await expect(targetCard).toContainText('root@127.0.0.1:22');
  await expect(page.locator('.monitor-panel')).toContainText('prod-a');

  const clusterForm = page.locator('form.cluster-form');
  await clusterForm.getByLabel('Cluster name').fill('prod-cluster');
  await clusterForm.getByLabel('Deployment parallelism').fill('1');
  await clusterForm.getByLabel('Required deployment approvals').fill('1');
  await clusterForm.getByLabel('prod-a').check();
  await clusterForm.getByRole('button', { name: /Add Cluster/ }).click();

  const clusterCard = page.locator('.cluster-card').filter({ hasText: 'prod-cluster' });
  await expect(clusterCard).toContainText('1 approval');
  await clusterCard.getByLabel('Rollout batch for prod-cluster').fill('1');
  await clusterCard.getByTitle('Request approval').click();
  const approvalCard = page.locator('.approval-card').filter({ hasText: 'prod-cluster' });
  await expect(approvalCard).toContainText('0/1 approval');
  await expect(approvalCard).toContainText('batch 1');
  await page.getByLabel('Approval actor').fill('alice');
  await approvalCard.getByTitle('Approve request').click();
  await expect(approvalCard).toContainText('1/1 approval');
  await approvalCard.getByTitle('Preview approved request').click();
  await expect(page.locator('.deploy-plan')).toContainText('alice');

  await targetCard.getByTitle('Preview deployment').click();
  await expect(page.locator('.deploy-plan')).toContainText('Dry-run deployment plan');
  await expect(page.locator('.deploy-plan')).toContainText('upload');
  await expect(page.locator('.deploy-plan')).toContainText('rollback');
  await expect(page.locator('.audit-list')).toContainText('deploy.run');
  await expect(page.locator('.audit-list')).toContainText('rollback 1 planned');
  await expect(page.locator('.audit-list')).toContainText('dry-run');
  await expect(page.locator('.audit-list')).toContainText('approval.request');
  await expect(page.locator('.audit-list')).toContainText('approval.approve');
  await expect(page.locator('.audit-list')).toContainText('request approved');
  await page.getByRole('button', { name: /Deploys/ }).click();
  await expect(page.locator('.audit-list')).toContainText('deploy.run');
  await expect(page.locator('.audit-list')).not.toContainText('approval.request');
  await page.getByRole('button', { name: /Approvals/ }).click();
  await expect(page.locator('.audit-list')).toContainText('approval.request');
  await expect(page.locator('.audit-list')).toContainText('approval.approve');
  await expect(page.locator('.audit-list')).not.toContainText('deploy.run');
  await page.getByRole('button', { name: /All/ }).click();

  await expect(page.locator('.monitor-panel')).toContainText('Unknown');
  await expect(page.locator('.audit-panel')).toContainText('Live');
});
