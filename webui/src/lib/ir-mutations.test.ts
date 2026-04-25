import { describe, expect, it } from 'vitest';
import { connectEntities, moveEntity } from './ir-mutations';
import type { Model } from './types';

const baseModel: Model = {
  version: 1,
  id: 'p_1',
  name: 'edge',
  engines: ['haproxy'],
  frontends: [{ id: 'fe_web', name: 'web', bind: ':80', protocol: 'http', rules: [], view: { x: 0, y: 0 } }],
  backends: [{ id: 'be_app', name: 'app', algorithm: 'roundrobin', servers: [], view: { x: 0, y: 0 } }],
  servers: [{ id: 's1', address: '10.0.0.1', port: 8080, weight: 100 }],
  rules: [
    {
      id: 'r_api',
      predicate: { type: 'path_prefix', value: '/api/' },
      action: { type: 'use_backend' },
      view: { x: 0, y: 0 }
    }
  ],
  tls_profiles: [],
  health_checks: [],
  rate_limits: [],
  caches: [],
  loggers: [],
  view: { zoom: 1, pan: { x: 0, y: 0 } }
};

describe('ir topology mutations', () => {
  it('moves entities with persisted view metadata', () => {
    const result = moveEntity(baseModel, 'fe_web', 120, 240);
    expect(result.changed).toBe(true);
    expect(result.model.frontends[0].view).toEqual({ x: 120, y: 240 });
    expect(baseModel.frontends[0].view).toEqual({ x: 0, y: 0 });
  });

  it('connects frontend to backend as default backend', () => {
    const result = connectEntities(baseModel, 'fe_web', 'be_app');
    expect(result.changed).toBe(true);
    expect(result.model.frontends[0].default_backend).toBe('be_app');
  });

  it('connects backend to server as pool member', () => {
    const result = connectEntities(baseModel, 'be_app', 's1');
    expect(result.changed).toBe(true);
    expect(result.model.backends[0].servers).toEqual(['s1']);
  });

  it('connects rules and frontends without duplicating references', () => {
    const frontendRule = connectEntities(baseModel, 'fe_web', 'r_api');
    expect(frontendRule.changed).toBe(true);
    expect(frontendRule.model.frontends[0].rules).toEqual(['r_api']);

    const ruleBackend = connectEntities(baseModel, 'r_api', 'be_app');
    expect(ruleBackend.changed).toBe(true);
    expect(ruleBackend.model.rules[0].action.backend_id).toBe('be_app');

    const duplicate = connectEntities(frontendRule.model, 'fe_web', 'r_api');
    expect(duplicate.changed).toBe(true);
    expect(duplicate.model.frontends[0].rules).toEqual(['r_api']);
  });

  it('returns unchanged results for unknown moves and connections', () => {
    const moved = moveEntity(baseModel, 'missing', 1, 2);
    expect(moved.changed).toBe(false);
    expect(moved.model).toBe(baseModel);

    const connected = connectEntities(baseModel, 'missing', 'be_app');
    expect(connected.changed).toBe(false);
    expect(connected.model).toBe(baseModel);
  });
});
