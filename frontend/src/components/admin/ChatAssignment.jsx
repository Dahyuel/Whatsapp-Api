import { useState, useEffect, useCallback } from 'react';
import { fetchApi } from '../../api';
import { Users, MessageSquare, Plus, Minus, RefreshCw } from 'lucide-react';
import { toast } from 'react-hot-toast';

export default function ChatAssignment() {
  const [agents, setAgents] = useState([]);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const [sessions, setSessions] = useState([]);
  const [selectedSession, setSelectedSession] = useState('');
  const [availableChats, setAvailableChats] = useState([]);
  const [assignedChats, setAssignedChats] = useState([]);
  const [loadingChats, setLoadingChats] = useState(false);

  const fetchAgents = useCallback(async () => {
    try {
      const data = await fetchApi('/admin/users');
      const agentList = (data.users || []).filter(u => u.role === 'agent');
      setAgents(agentList);
    } catch (e) {
      toast.error('Could not load agents: ' + e.message);
    }
  }, []);

  const fetchSessions = useCallback(async () => {
    try {
      const data = await fetchApi('/sessions');
      setSessions(data.sessions || []);
    } catch (e) { /* no sessions yet */ }
  }, []);

  useEffect(() => {
    fetchAgents();
    fetchSessions();
  }, [fetchAgents, fetchSessions]);

  const fetchAvailableChats = useCallback(async (sessionId) => {
    if (!sessionId) { setAvailableChats([]); return; }
    setLoadingChats(true);
    try {
      const data = await fetchApi(`/chats?session=${sessionId}`);
      setAvailableChats(data.chats || []);
    } catch (e) {
      toast.error('Could not load chats: ' + e.message);
      setAvailableChats([]);
    } finally {
      setLoadingChats(false);
    }
  }, []);

  const fetchAssigned = useCallback(async (agentId) => {
    if (!agentId) return;
    try {
      const data = await fetchApi(`/admin/users/${agentId}/chats`);
      setAssignedChats(data.chats || []);
    } catch (e) {
      setAssignedChats([]);
    }
  }, []);

  const handleAgentChange = (agent) => {
    setSelectedAgent(agent);
    fetchAssigned(agent.id);
  };

  const handleSessionChange = (sessId) => {
    setSelectedSession(sessId);
    fetchAvailableChats(sessId);
  };

  const isAssigned = (jid) =>
    assignedChats.some(ac => ac.jid === jid && ac.session_id === selectedSession);

  const assignChat = async (jid) => {
    if (!selectedAgent || !selectedSession) return;
    try {
      await fetchApi(`/admin/users/${selectedAgent.id}/chats`, {
        method: 'POST',
        body: JSON.stringify({ session_id: selectedSession, jids: [jid] }),
      });
      toast.success('Chat assigned');
      fetchAssigned(selectedAgent.id);
    } catch (e) {
      toast.error(e.message);
    }
  };

  const unassignChat = async (chatId) => {
    if (!selectedAgent) return;
    try {
      await fetchApi(`/admin/users/${selectedAgent.id}/chats/${chatId}`, { method: 'DELETE' });
      toast.success('Chat unassigned');
      fetchAssigned(selectedAgent.id);
    } catch (e) {
      toast.error(e.message);
    }
  };

  const formatJID = (jid) => {
    // Strip @s.whatsapp.net and show just the number
    return jid?.replace(/@.+$/, '') || jid;
  };

  return (
    <div className="animate-fade-in">
      <div className="mb-4">
        <h2>Chat Assignment</h2>
        <p style={{ margin: 0 }}>Assign WhatsApp chats to agents</p>
      </div>

      <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
        {/* Left: Agent selector */}
        <div className="card" style={{ flex: '0 0 260px', minWidth: 220 }}>
          <div className="flex items-center gap-2 mb-4">
            <Users size={18} />
            <h3 style={{ margin: 0 }}>Select Agent</h3>
          </div>
          {agents.length === 0 ? (
            <p style={{ fontSize: '0.875rem' }}>No agents found. Create one in Users.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {agents.map(a => (
                <button
                  key={a.id}
                  onClick={() => handleAgentChange(a)}
                  className={`agent-select-btn ${selectedAgent?.id === a.id ? 'agent-select-btn--active' : ''}`}
                >
                  {a.username}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Right: Chats */}
        {selectedAgent && (
          <div style={{ flex: 1, minWidth: 300 }}>
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 style={{ margin: 0 }}>Chats for <span style={{ color: 'var(--primary)' }}>{selectedAgent.username}</span></h3>
                <button className="btn btn-outline" style={{ padding: '0.25rem 0.75rem', fontSize: '0.8rem' }} onClick={() => fetchAssigned(selectedAgent.id)}>
                  <RefreshCw size={14} />
                </button>
              </div>

              <div className="form-group">
                <label className="form-label">Session</label>
                <select className="form-select" value={selectedSession} onChange={e => handleSessionChange(e.target.value)}>
                  <option value="">Select a session…</option>
                  {sessions.map(s => (
                    <option key={s.id} value={s.id}>{s.id} ({s.status})</option>
                  ))}
                </select>
              </div>

              {selectedSession && (
                <>
                  <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.75rem' }}>
                    Click <strong>+</strong> to assign a chat, <strong>−</strong> to unassign
                  </p>
                  {loadingChats ? (
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>Loading chats…</p>
                  ) : availableChats.length === 0 ? (
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>No chats in this session</p>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', maxHeight: '400px', overflowY: 'auto' }}>
                      {availableChats.map(chat => {
                        const jid = chat.JID || chat.jid || chat.id;
                        const assigned = isAssigned(jid);
                        const assignedEntry = assignedChats.find(ac => ac.jid === jid && ac.session_id === selectedSession);
                        return (
                          <div key={jid} className={`chat-assign-row ${assigned ? 'chat-assign-row--assigned' : ''}`}>
                            <MessageSquare size={14} />
                            <span style={{ flex: 1, fontFamily: 'monospace', fontSize: '0.85rem' }}>
                              {formatJID(jid)}
                            </span>
                            {assigned ? (
                              <button
                                className="btn btn-danger"
                                style={{ padding: '0.2rem 0.5rem', fontSize: '0.75rem' }}
                                onClick={() => unassignChat(assignedEntry?.id)}
                              >
                                <Minus size={12} />
                              </button>
                            ) : (
                              <button
                                className="btn btn-primary"
                                style={{ padding: '0.2rem 0.5rem', fontSize: '0.75rem' }}
                                onClick={() => assignChat(jid)}
                              >
                                <Plus size={12} />
                              </button>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </>
              )}
            </div>

            {/* Currently assigned chats list */}
            {assignedChats.length > 0 && (
              <div className="card" style={{ marginTop: 0 }}>
                <h3 style={{ margin: '0 0 0.75rem' }}>All Assigned Chats</h3>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                  {assignedChats.map(ac => (
                    <div key={ac.id} style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', padding: '0.5rem', background: 'rgba(99,102,241,0.08)', borderRadius: '8px' }}>
                      <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>{ac.session_id}</span>
                      <span style={{ flex: 1, fontFamily: 'monospace', fontSize: '0.85rem' }}>{formatJID(ac.jid)}</span>
                      <button
                        className="btn btn-danger"
                        style={{ padding: '0.2rem 0.5rem', fontSize: '0.75rem' }}
                        onClick={() => unassignChat(ac.id)}
                      >
                        <Minus size={12} />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
