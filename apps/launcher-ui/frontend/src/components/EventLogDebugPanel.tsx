import React, { useState } from 'react';
import { useEventLog } from '../stores/eventLog.store';
import { usePatchFailureLog } from '../stores/patchFailureLog.store';
import { StateReconstructor } from '../event/StateReconstructor';
import { EventReplay } from '../event/EventReplay';

export const EventLogDebugPanel: React.FC<{ sessionId: string }> = ({ sessionId }) => {
  const eventLog = useEventLog();
  const failureLog = usePatchFailureLog();
  const [activeTab, setActiveTab] = useState<'events' | 'failures' | 'stats' | 'replay'>('events');
  const [expandedEvent, setExpandedEvent] = useState<string | null>(null);

  const sessionEvents = eventLog.getEntriesBySession(sessionId);
  const sessionFailures = failureLog.getFailuresBySession(sessionId);
  const stats = eventLog.getStats();

  const handleReplay = () => {
    const result = EventReplay.replaySession(sessionId, { printStats: true });
    console.log('Replay result:', result);
    alert(`Replayed ${result.eventsReplayed} events in ${result.duration}ms`);
  };

  const handleVerifyConsistency = () => {
    const verification = EventReplay.verifyStateConsistency(sessionId);
    if (verification.isValid) {
      alert('✓ State is consistent');
    } else {
      alert(`✗ State violations:\n${verification.violations.join('\n')}`);
    }
  };

  const handleExportLog = () => {
    const data = {
      sessionId,
      events: sessionEvents,
      failures: sessionFailures,
      timestamp: new Date().toISOString(),
    };
    const json = JSON.stringify(data, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `event-log-${sessionId}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="bg-gray-900 text-gray-100 rounded-lg p-4 font-mono text-xs border border-gray-700">
      <div className="mb-4">
        <h3 className="text-sm font-bold text-blue-400 mb-3">Event Log Debug Panel</h3>

        {/* Tabs */}
        <div className="flex gap-2 mb-4 border-b border-gray-700">
          {(['events', 'failures', 'stats', 'replay'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-3 py-1 text-xs font-medium transition-colors ${
                activeTab === tab
                  ? 'text-blue-400 border-b-2 border-blue-400'
                  : 'text-gray-400 hover:text-gray-300'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        {/* Events Tab */}
        {activeTab === 'events' && (
          <div className="max-h-96 overflow-y-auto space-y-2">
            {sessionEvents.length === 0 ? (
              <p className="text-gray-500">No events logged</p>
            ) : (
              sessionEvents.map((entry) => (
                <div
                  key={entry.id}
                  className="border border-gray-700 rounded p-2 cursor-pointer hover:bg-gray-800"
                  onClick={() => setExpandedEvent(expandedEvent === entry.id ? null : entry.id)}
                >
                  <div className="flex justify-between items-center">
                    <span className={`font-bold ${entry.status === 'applied' ? 'text-green-400' : 'text-red-400'}`}>
                      {entry.event.type}
                    </span>
                    <span className="text-gray-500">{new Date(entry.appliedAt).toLocaleTimeString()}</span>
                  </div>

                  {expandedEvent === entry.id && (
                    <div className="mt-2 bg-gray-800 p-2 rounded text-xs text-gray-300 overflow-x-auto">
                      <pre>{JSON.stringify(entry.event, null, 2)}</pre>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        )}

        {/* Failures Tab */}
        {activeTab === 'failures' && (
          <div className="max-h-96 overflow-y-auto space-y-2">
            {sessionFailures.length === 0 ? (
              <p className="text-gray-500">No failures logged</p>
            ) : (
              sessionFailures.map((failure) => (
                <div key={failure.id} className="border border-red-700 bg-red-950 rounded p-2">
                  <div className="flex justify-between items-center mb-1">
                    <span className="font-bold text-red-300">{failure.eventType}</span>
                    <span className="text-gray-500 text-xs">{failure.domain}</span>
                  </div>
                  <div className="text-red-200 text-xs space-y-1">
                    {failure.reason.map((r, i) => (
                      <div key={i}>• {r}</div>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {/* Stats Tab */}
        {activeTab === 'stats' && (
          <div className="space-y-2">
            <div className="bg-gray-800 p-2 rounded">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="text-gray-400">Total Events:</span>
                  <span className="ml-2 text-yellow-400 font-bold">{stats.totalEvents}</span>
                </div>
                <div>
                  <span className="text-gray-400">Applied:</span>
                  <span className="ml-2 text-green-400 font-bold">{stats.applied}</span>
                </div>
                <div>
                  <span className="text-gray-400">Rejected:</span>
                  <span className="ml-2 text-red-400 font-bold">{stats.rejected}</span>
                </div>
                <div>
                  <span className="text-gray-400">Success Rate:</span>
                  <span className="ml-2 text-blue-400 font-bold">
                    {stats.totalEvents > 0 ? ((stats.applied / stats.totalEvents) * 100).toFixed(1) : 0}%
                  </span>
                </div>
              </div>

              <div className="mt-3 pt-3 border-t border-gray-700">
                <span className="text-gray-400 block mb-2">Events by Type:</span>
                <div className="space-y-1">
                  {Object.entries(stats.byDomain).map(([domain, count]) => (
                    <div key={domain} className="flex justify-between text-xs">
                      <span className="text-gray-300">{domain}</span>
                      <span className="text-purple-400">{count}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Replay Tab */}
        {activeTab === 'replay' && (
          <div className="space-y-2">
            <button
              onClick={handleReplay}
              className="w-full bg-blue-600 hover:bg-blue-700 text-white px-3 py-2 rounded text-xs font-medium transition-colors"
            >
              Replay All Events
            </button>
            <button
              onClick={handleVerifyConsistency}
              className="w-full bg-green-600 hover:bg-green-700 text-white px-3 py-2 rounded text-xs font-medium transition-colors"
            >
              Verify State Consistency
            </button>
            <button
              onClick={handleExportLog}
              className="w-full bg-purple-600 hover:bg-purple-700 text-white px-3 py-2 rounded text-xs font-medium transition-colors"
            >
              Export Event Log
            </button>
          </div>
        )}
      </div>

      {/* Footer Stats */}
      <div className="text-xs text-gray-500 border-t border-gray-700 pt-2 mt-4">
        <div className="flex justify-between">
          <span>Session: {sessionId.slice(0, 8)}...</span>
          <span>Events: {sessionEvents.length}</span>
          <span>Failures: {sessionFailures.length}</span>
        </div>
      </div>
    </div>
  );
};
