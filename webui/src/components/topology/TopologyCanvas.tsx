import {
  Background,
  Controls,
  type Connection,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  type Node,
  type NodeProps
} from '@xyflow/react';
import { useMemo } from 'react';
import { buildTopology, type TopologyNodeData } from '../../lib/topology';
import type { Issue, Model } from '../../lib/types';

const nodeTypes = {
  mizan: MizanNode
};

type MizanFlowNode = Node<TopologyNodeData, 'mizan'>;

interface TopologyCanvasProps {
  model: Model | null;
  issues?: Issue[];
  onMoveEntity?: (entityID: string, x: number, y: number) => void;
  onConnectEntities?: (sourceID: string, targetID: string) => void;
}

export function TopologyCanvas({ model, issues = [], onMoveEntity, onConnectEntities }: TopologyCanvasProps) {
  const graph = useMemo(() => (model ? buildTopology(model, issues) : { nodes: [], edges: [] }), [issues, model]);

  const handleConnect = (connection: Connection) => {
    if (!connection.source || !connection.target) return;
    onConnectEntities?.(connection.source, connection.target);
  };

  if (!model) {
    return <div className="topology-empty">Create or import a project to render topology.</div>;
  }

  return (
    <div className="topology-canvas">
      <ReactFlow
        nodes={graph.nodes}
        edges={graph.edges}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.18 }}
        minZoom={0.35}
        maxZoom={1.7}
        nodesDraggable
        nodesConnectable
        onNodeDragStop={(_event, node) => {
          onMoveEntity?.(node.id, node.position.x, node.position.y);
        }}
        onConnect={handleConnect}
        elementsSelectable
      >
        <Background gap={20} size={1} />
        <MiniMap pannable zoomable nodeStrokeWidth={3} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}

function MizanNode({ data }: NodeProps<MizanFlowNode>) {
  return (
    <div className={`topology-node ${data.kind}`}>
      <Handle type="target" position={Position.Left} />
      <div>
        <strong>{data.label}</strong>
        <span>{data.subtitle}</span>
      </div>
      {data.issueCount > 0 && <mark>{data.issueCount}</mark>}
      <Handle type="source" position={Position.Right} />
    </div>
  );
}
