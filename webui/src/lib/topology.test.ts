import { describe, expect, it } from 'vitest';
import { buildTopology } from './topology';
import type { Model } from './types';

const model: Model = {
  version: 1,
  id: 'p_1',
  name: 'edge',
  engines: ['haproxy'],
  frontends: [
    {
      id: 'fe_web',
      name: 'web',
      bind: ':443',
      protocol: 'http',
      tls_id: 'tls_default',
      rules: ['r_api'],
      default_backend: 'be_app',
      view: { x: 0, y: 0 }
    }
  ],
  backends: [
    { id: 'be_app', name: 'app', algorithm: 'leastconn', servers: ['s1'], view: { x: 0, y: 0 } },
    { id: 'be_api', name: 'api', algorithm: 'roundrobin', servers: [], view: { x: 0, y: 0 } }
  ],
  servers: [{ id: 's1', address: '10.0.0.1', port: 8080, weight: 100 }],
  rules: [
    {
      id: 'r_api',
      predicate: { type: 'path_prefix', value: '/api/' },
      action: { type: 'use_backend', backend_id: 'be_api' },
      view: { x: 0, y: 0 }
    }
  ],
  tls_profiles: [{ id: 'tls_default', cert_path: '/etc/ssl/edge.pem' }],
  health_checks: [],
  rate_limits: [],
  caches: [],
  loggers: [],
  view: { zoom: 1, pan: { x: 0, y: 0 } }
};

describe('buildTopology', () => {
  it('creates nodes and route edges from IR references', () => {
    const graph = buildTopology(model, [{ severity: 'error', entity_id: 'fe_web', message: 'bad' }]);

    expect(graph.nodes.map((node) => node.id)).toEqual(
      expect.arrayContaining(['fe_web', 'tls_default', 'r_api', 'be_app', 'be_api', 's1'])
    );
    expect(graph.edges.map((edge) => `${edge.source}->${edge.target}`)).toEqual(
      expect.arrayContaining(['fe_web->r_api', 'r_api->be_api', 'fe_web->be_app', 'be_app->s1'])
    );
    expect(graph.nodes.find((node) => node.id === 'fe_web')?.data.issueCount).toBe(1);
  });

  it('uses persisted positions and health check side nodes', () => {
    const withHealth = {
      ...model,
      backends: [{ ...model.backends[0], health_check_id: 'hc_default', view: { x: 12, y: 34 } }],
      health_checks: [{ id: 'hc_default', type: 'http', path: '/healthz', interval_ms: 1, timeout_ms: 1, rise: 1, fall: 1 }]
    };
    const graph = buildTopology(withHealth);
    expect(graph.nodes.find((node) => node.id === 'be_app')?.position).toEqual({ x: 12, y: 34 });
    expect(graph.nodes.map((node) => node.id)).toContain('hc_default');
    expect(graph.edges.map((edge) => `${edge.source}->${edge.target}`)).toContain('hc_default->be_app');
  });

  it('handles missing optional references with fallback labels', () => {
    const sparse: Model = {
      ...model,
      frontends: [
        {
          id: 'fe_sparse',
          name: '',
          bind: '',
          protocol: '',
          tls_id: 'tls_missing',
          rules: ['missing_rule'],
          view: { x: 0, y: 0 }
        }
      ],
      backends: [
        {
          id: 'be_sparse',
          name: '',
          algorithm: '',
          health_check_id: 'hc_missing',
          servers: ['missing_server'],
          view: { x: 0, y: 0 }
        }
      ],
      tls_profiles: [],
      health_checks: []
    };
    const graph = buildTopology(sparse, [{ severity: 'warning', message: 'global' }]);
    expect(graph.nodes.find((node) => node.id === 'fe_sparse')?.data.subtitle).toBe('unbound - http');
    expect(graph.nodes.find((node) => node.id === 'tls_missing')?.data.subtitle).toBe('certificate pending');
    expect(graph.nodes.find((node) => node.id === 'hc_missing')?.data.subtitle).toBe('check');
    expect(graph.edges.map((edge) => edge.id)).toContain('tls_missing->fe_sparse:tls');
    expect(graph.edges.map((edge) => edge.id)).toContain('hc_missing->be_sparse:checks');
  });
});
