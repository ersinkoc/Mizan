import type { Edge, Node } from '@xyflow/react';
import type { Issue, Model } from './types';

type NodeKind = 'frontend' | 'rule' | 'backend' | 'server' | 'aux';

export interface TopologyNodeData extends Record<string, unknown> {
  label: string;
  subtitle: string;
  kind: NodeKind;
  issueCount: number;
}

export function buildTopology(model: Model, issues: Issue[] = []): { nodes: Node<TopologyNodeData>[]; edges: Edge[] } {
  const nodes: Node<TopologyNodeData>[] = [];
  const edges: Edge[] = [];
  const issueCounts = new Map<string, number>();
  for (const issue of issues) {
    if (!issue.entity_id) continue;
    issueCounts.set(issue.entity_id, (issueCounts.get(issue.entity_id) ?? 0) + 1);
  }

  const addNode = (
    id: string,
    kind: NodeKind,
    label: string,
    subtitle: string,
    position: { x: number; y: number }
  ) => {
    nodes.push({
      id,
      type: 'mizan',
      position,
      data: { label, subtitle, kind, issueCount: issueCounts.get(id) ?? 0 }
    });
  };

  model.frontends.forEach((frontend, index) => {
    addNode(
      frontend.id,
      'frontend',
      frontend.name || frontend.id,
      `${frontend.bind || 'unbound'} - ${frontend.protocol || 'http'}`,
      pickPosition(frontend.view, 80, 90 + index * 150)
    );

    if (frontend.tls_id) {
      const tls = model.tls_profiles.find((item) => item.id === frontend.tls_id);
      addNode(
        frontend.tls_id,
        'aux',
        tls?.name || 'TLS',
        tls?.cert_path || 'certificate pending',
        { x: 80, y: 45 + index * 150 }
      );
      edges.push(edge(frontend.tls_id, frontend.id, 'tls'));
    }

    frontend.rules?.forEach((ruleID, ruleIndex) => {
      const rule = model.rules.find((item) => item.id === ruleID);
      if (!rule) return;
      addNode(
        rule.id,
        'rule',
        rule.name || rule.id,
        `${rule.predicate.type}: ${rule.predicate.value}`,
        pickPosition(rule.view, 310, 70 + index * 150 + ruleIndex * 72)
      );
      edges.push(edge(frontend.id, rule.id, 'routes'));
      if (rule.action.backend_id) {
        edges.push(edge(rule.id, rule.action.backend_id, 'uses'));
      }
    });

    if (frontend.default_backend) {
      edges.push(edge(frontend.id, frontend.default_backend, 'default'));
    }
  });

  model.backends.forEach((backend, index) => {
    addNode(
      backend.id,
      'backend',
      backend.name || backend.id,
      `${backend.algorithm || 'roundrobin'} - ${backend.servers.length} servers`,
      pickPosition(backend.view, 560, 90 + index * 150)
    );

    if (backend.health_check_id) {
      const hc = model.health_checks.find((item) => item.id === backend.health_check_id);
      addNode(
        backend.health_check_id,
        'aux',
        hc?.name || 'Health check',
        hc?.path || hc?.type || 'check',
        { x: 560, y: 45 + index * 150 }
      );
      edges.push(edge(backend.health_check_id, backend.id, 'checks'));
    }

    backend.servers.forEach((serverID, serverIndex) => {
      const server = model.servers.find((item) => item.id === serverID);
      if (!server) return;
      addNode(
        server.id,
        'server',
        server.name || server.id,
        `${server.address}:${server.port} - weight ${server.weight || 100}`,
        { x: 820, y: 70 + index * 150 + serverIndex * 72 }
      );
      edges.push(edge(backend.id, server.id, 'member'));
    });
  });

  return { nodes, edges };
}

function pickPosition(view: { x: number; y: number } | undefined, fallbackX: number, fallbackY: number) {
  if (view && (view.x !== 0 || view.y !== 0)) {
    return { x: view.x, y: view.y };
  }
  return { x: fallbackX, y: fallbackY };
}

function edge(source: string, target: string, label: string): Edge {
  return {
    id: `${source}->${target}:${label}`,
    source,
    target,
    label,
    type: 'smoothstep',
    animated: label === 'routes',
    style: { strokeWidth: label === 'default' ? 2.5 : 1.6 }
  };
}
