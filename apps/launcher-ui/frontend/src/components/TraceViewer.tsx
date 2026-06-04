import { useState, useRef, useEffect } from 'react';
import { useTraceStore, TraceNode } from '../stores/trace.store';
import { Download, CheckCircle, AlertCircle, Search, FileJson, Clock } from 'lucide-react';

interface TraceViewerProps {
  sessionId?: string;
}

export function TraceViewer({ sessionId }: TraceViewerProps) {
  const { traces, activeTrace, setActiveTrace, exportTrace } = useTraceStore();
  const [filter, setFilter] = useState<string>('all');
  const scrollRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  // Use provided sessionId or active trace
  const currentSessionId = sessionId || activeTrace;
  const traceNodes = currentSessionId ? traces.get(currentSessionId) || [] : [];

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [traceNodes, autoScroll]);

  // Filter nodes
  const filteredNodes = traceNodes.filter((node) => {
    if (filter === 'all') return true;
    if (filter === 'downloads') return node.type === 'download';
    if (filter === 'errors') return node.type === 'error';
    if (filter === 'complete') return node.type === 'complete';
    return true;
  });

  // Stats
  const totalNodes = traceNodes.length;
  const completedNodes = traceNodes.filter((n) => n.type === 'complete').length;
  const errorNodes = traceNodes.filter((n) => n.type === 'error').length;
  const downloadNodes = traceNodes.filter((n) => n.type === 'download').length;

  const handleExport = () => {
    if (currentSessionId) {
      const json = exportTrace(currentSessionId);
      // Create download
      const blob = new Blob([json], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `trace-${currentSessionId}.json`;
      a.click();
      URL.revokeObjectURL(url);
    }
  };

  if (!currentSessionId) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-400">
        <FileJson size={48} className="mb-4 opacity-50" />
        <p>No active trace session</p>
        <p className="text-sm mt-2">Join a server to see execution trace</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-slate-200">
            Session Trace
          </h3>
          <p className="text-sm text-slate-400">{currentSessionId}</p>
        </div>
        
        <div className="flex items-center gap-2">
          {/* Filter */}
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-slate-300"
          >
            <option value="all">All Events</option>
            <option value="downloads">Downloads</option>
            <option value="errors">Errors</option>
            <option value="complete">Complete</option>
          </select>

          {/* Auto-scroll toggle */}
          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`p-2 rounded-lg transition-colors ${
              autoScroll ? 'bg-emerald-600 text-white' : 'bg-slate-700 text-slate-400'
            }`}
            title="Auto-scroll"
          >
            ↓
          </button>

          {/* Export */}
          <button
            onClick={handleExport}
            className="flex items-center gap-2 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm text-slate-300 transition-colors"
          >
            <FileJson size={14} />
            Export
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-3 mb-4">
        <StatCard label="Total" value={totalNodes} color="blue" />
        <StatCard label="Downloads" value={downloadNodes} color="purple" />
        <StatCard label="Complete" value={completedNodes} color="emerald" />
        <StatCard label="Errors" value={errorNodes} color={errorNodes > 0 ? 'red' : 'slate'} />
      </div>

      {/* Timeline */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto bg-slate-800 rounded-lg border border-slate-700 p-4"
      >
        {filteredNodes.length === 0 ? (
          <div className="text-center text-slate-500 py-8">
            No events match the current filter
          </div>
        ) : (
          <div className="space-y-2">
            {filteredNodes.map((node, index) => (
              <TraceNodeItem
                key={node.id}
                node={node}
                isLast={index === filteredNodes.length - 1}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: 'blue' | 'purple' | 'emerald' | 'red' | 'slate';
}) {
  const colors = {
    blue: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
    purple: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
    emerald: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
    red: 'bg-red-500/20 text-red-400 border-red-500/30',
    slate: 'bg-slate-700 text-slate-400 border-slate-600',
  };

  return (
    <div className={`p-3 rounded-lg border ${colors[color]}`}>
      <div className="text-2xl font-bold">{value}</div>
      <div className="text-xs opacity-80">{label}</div>
    </div>
  );
}

function TraceNodeItem({ node, isLast }: { node: TraceNode; isLast: boolean }) {
  const formatTime = (timestamp: number) => {
    const date = new Date(timestamp * 1000);
    const ms = String(date.getMilliseconds()).padStart(3, '0');
    const time = date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
    return `${time}.${ms}`;
  };

  const getIcon = () => {
    switch (node.type) {
      case 'resolve':
        return <Search size={16} className="text-blue-400" />;
      case 'download':
        return <Download size={16} className="text-purple-400" />;
      case 'verify':
        return <CheckCircle size={16} className="text-emerald-400" />;
      case 'complete':
        return <CheckCircle size={16} className="text-emerald-400" />;
      case 'error':
        return <AlertCircle size={16} className="text-red-400" />;
      default:
        return <Clock size={16} className="text-slate-400" />;
    }
  };

  const getTypeLabel = () => {
    switch (node.type) {
      case 'resolve':
        return 'Resolve';
      case 'download':
        return 'Download';
      case 'verify':
        return 'Verify';
      case 'install':
        return 'Install';
      case 'complete':
        return 'Complete';
      case 'error':
        return 'Error';
      default:
        return node.type;
    }
  };

  return (
    <div className={`flex items-start gap-3 py-2 ${!isLast ? 'border-b border-slate-700/50' : ''}`}>
      {/* Icon */}
      <div className="mt-0.5">{getIcon()}</div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-500 font-mono">{formatTime(node.timestamp)}</span>
          <span className="text-xs px-2 py-0.5 rounded bg-slate-700 text-slate-300">
            {getTypeLabel()}
          </span>
          {node.provider && (
            <span className="text-xs px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400">
              {node.provider}
            </span>
          )}
        </div>

        <div className="mt-1">
          <span className="text-sm text-slate-200 font-medium">{node.modName}</span>
          {node.providerReason && (
            <span className="text-xs text-slate-500 ml-2">({node.providerReason})</span>
          )}
        </div>

        {/* Progress bar for downloads */}
        {node.type === 'download' && node.progress && (
          <div className="mt-2">
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-1">
              <span>{node.progress.percent}%</span>
              {node.progress.speed && (
                <span>{(node.progress.speed / 1024 / 1024).toFixed(1)} MB/s</span>
              )}
            </div>
            <div className="w-full bg-slate-700 rounded-full h-1.5">
              <div
                className="bg-purple-500 h-1.5 rounded-full transition-all duration-300"
                style={{ width: `${node.progress.percent}%` }}
              />
            </div>
          </div>
        )}

        {/* Error message */}
        {node.error && (
          <div className="mt-1 text-sm text-red-400">{node.error}</div>
        )}
      </div>
    </div>
  );
}
