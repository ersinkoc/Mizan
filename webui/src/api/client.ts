import type { AuditEvent, Engine, GenerateResult, IRResponse, Model, ProjectMeta, SnapshotTag, ValidateResult } from '../lib/types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {})
    }
  });
  if (!res.ok) {
    const problem = await res.json().catch(() => ({ detail: res.statusText }));
    throw new Error(problem.detail ?? res.statusText);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export const api = {
  listProjects: () => request<ProjectMeta[]>('/api/v1/projects'),
  createProject: (body: { name: string; description: string; engines: Engine[] }) =>
    request<{ project: ProjectMeta; ir: Model; version: string }>('/api/v1/projects', {
      method: 'POST',
      body: JSON.stringify(body)
    }),
  importProject: (body: { name: string; description: string; filename: string; config: string }) =>
    request<{ project: ProjectMeta; ir: Model; version: string; issues: IRResponse['issues'] }>('/api/v1/projects/import', {
      method: 'POST',
      body: JSON.stringify(body)
    }),
  getIR: (projectID: string) => request<IRResponse>(`/api/v1/projects/${projectID}/ir`),
  saveIR: (projectID: string, ir: Model, version: string) =>
    request<IRResponse>(`/api/v1/projects/${projectID}/ir`, {
      method: 'PATCH',
      headers: { 'If-Match': version },
      body: JSON.stringify({ ir })
    }),
  generate: (projectID: string, target: Engine) =>
    request<GenerateResult>(`/api/v1/projects/${projectID}/generate`, {
      method: 'POST',
      body: JSON.stringify({ target })
    }),
  validate: (projectID: string, target: Engine) =>
    request<ValidateResult>(`/api/v1/projects/${projectID}/validate`, {
      method: 'POST',
      body: JSON.stringify({ target })
    }),
  listSnapshots: (projectID: string) => request<string[]>(`/api/v1/projects/${projectID}/ir/snapshots`),
  listTags: (projectID: string) => request<SnapshotTag[]>(`/api/v1/projects/${projectID}/ir/tags`),
  tagSnapshot: (projectID: string, snapshotRef: string, label: string) =>
    request<SnapshotTag>(`/api/v1/projects/${projectID}/ir/tag`, {
      method: 'POST',
      body: JSON.stringify({ snapshot_ref: snapshotRef, label })
    }),
  revertSnapshot: (projectID: string, snapshotRef: string, version: string) =>
    request<IRResponse>(`/api/v1/projects/${projectID}/ir/revert`, {
      method: 'POST',
      headers: { 'If-Match': version },
      body: JSON.stringify({ snapshot_ref: snapshotRef })
    }),
  listAudit: (projectID: string, limit = 100) => request<AuditEvent[]>(`/api/v1/projects/${projectID}/audit?limit=${limit}`)
};
