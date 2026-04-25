import type { Model } from './types';

export interface MutationResult {
  model: Model;
  changed: boolean;
}

export function moveEntity(model: Model, entityID: string, x: number, y: number): MutationResult {
  const next = cloneModel(model);
  for (const collection of [next.frontends, next.backends, next.rules]) {
    const entity = collection.find((item) => item.id === entityID);
    if (entity) {
      entity.view = { x, y };
      return { model: next, changed: true };
    }
  }
  return { model, changed: false };
}

export function connectEntities(model: Model, sourceID: string, targetID: string): MutationResult {
  const next = cloneModel(model);
  const sourceFrontend = next.frontends.find((item) => item.id === sourceID);
  const targetBackend = next.backends.find((item) => item.id === targetID);
  if (sourceFrontend && targetBackend) {
    sourceFrontend.default_backend = targetBackend.id;
    return { model: next, changed: true };
  }

  const sourceRule = next.rules.find((item) => item.id === sourceID);
  if (sourceRule && targetBackend) {
    sourceRule.action = { ...sourceRule.action, type: 'use_backend', backend_id: targetBackend.id };
    return { model: next, changed: true };
  }

  const targetRule = next.rules.find((item) => item.id === targetID);
  if (sourceFrontend && targetRule) {
    sourceFrontend.rules = appendUnique(sourceFrontend.rules ?? [], targetRule.id);
    return { model: next, changed: true };
  }

  const sourceBackend = next.backends.find((item) => item.id === sourceID);
  const targetServer = next.servers.find((item) => item.id === targetID);
  if (sourceBackend && targetServer) {
    sourceBackend.servers = appendUnique(sourceBackend.servers, targetServer.id);
    return { model: next, changed: true };
  }

  return { model, changed: false };
}

function cloneModel(model: Model): Model {
  return JSON.parse(JSON.stringify(model)) as Model;
}

function appendUnique(items: string[], item: string): string[] {
  return items.includes(item) ? items : [...items, item];
}

