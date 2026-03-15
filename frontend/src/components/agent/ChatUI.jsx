import { useState, useEffect, useRef, useCallback } from 'react';
import { fetchApi } from '../../api';
import { useAuth } from '../../context/AuthContext';
import { Search, LogOut, Phone, MoreVertical } from 'lucide-react';
import MessageThread from './MessageThread';
import MessageInput from './MessageInput';
import { toast } from 'react-hot-toast';

const API_BASE = 'http://localhost:3000';

function formatPhoneNumber(jid) {
  // Extract number from JID like 123456789@s.whatsapp.net
  return jid?.replace(/@.+$/, '') || jid || 'Unknown';
}

function formatTime(ts) {
  if (!ts) return '';
  try {
    const d = new Date(ts * 1000);
    const now = new Date();
    if (d.toDateString() === now.toDateString()) {
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    return d.toLocaleDateString([], { day: '2-digit', month: '2-digit' });
  } catch { return ''; }
}

export default function ChatUI() {
  const { user, logout } = useAuth();
  const [chats, setChats] = useState([]);
  const [selectedChat, setSelectedChat] = useState(null);
  const [search, setSearch] = useState('');
  const [lastMessages, setLastMessages] = useState({}); // jid → {text, time}
  const evtSourceRef = useRef(null);

  const fetchChats = useCallback(async () => {
    try {
      const data = await fetchApi('/agent/chats');
      setChats(data.chats || []);
    } catch (e) {
      toast.error('Could not load chats: ' + e.message);
    }
  }, []);

  // Connect to SSE for real-time events
  useEffect(() => {
    if (!user?.token) return;

    const es = new EventSource(`${API_BASE}/agent/events`, {
      withCredentials: false,
    });

    // EventSource doesn't support custom headers; use URL param as workaround
    // We'll reconnect with token in URL query
    es.close();

    // Use a polyfill approach: fetch-event-source with token in URL
    // Simple approach: reconnect via EventSource with token in query string
    const url = `${API_BASE}/agent/events?token=${encodeURIComponent(user.token)}`;
    const sse = new EventSource(url);

    sse.addEventListener('new_chat', (e) => {
      const ev = JSON.parse(e.data);
      toast.success(`New chat assigned: ${formatPhoneNumber(ev.data?.jid)}`);
      fetchChats();
    });

    sse.addEventListener('new_message', (e) => {
      const ev = JSON.parse(e.data);
      const { jid, text, time } = ev.data || {};
      if (jid) {
        setLastMessages(prev => ({ ...prev, [jid]: { text, time } }));
      }
    });

    sse.onerror = () => {
      // Auto-reconnects
    };

    evtSourceRef.current = sse;
    return () => sse.close();
  }, [user?.token, fetchChats]);

  useEffect(() => { fetchChats(); }, [fetchChats]);

  const filtered = chats.filter(c =>
    formatPhoneNumber(c.jid).includes(search)
  );

  return (
    <div className="agent-layout">
      {/* Left sidebar */}
      <div className="agent-sidebar">
        <div className="agent-sidebar-header">
          <div className="flex items-center gap-2">
            <div className="agent-avatar">{user?.username?.[0]?.toUpperCase()}</div>
            <div>
              <div style={{ fontWeight: 600, fontSize: '0.9rem' }}>{user?.username}</div>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Agent</div>
            </div>
          </div>
          <button className="icon-btn" onClick={logout} title="Sign out">
            <LogOut size={18} />
          </button>
        </div>

        <div className="agent-search-wrap">
          <Search size={16} className="agent-search-icon" />
          <input
            className="agent-search"
            placeholder="Search chats…"
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>

        <div className="agent-chat-list">
          {filtered.length === 0 && (
            <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)', fontSize: '0.875rem' }}>
              No chats assigned yet
            </div>
          )}
          {filtered.map(chat => {
            const phone = formatPhoneNumber(chat.jid);
            const last = lastMessages[chat.jid];
            const isActive = selectedChat?.jid === chat.jid;
            return (
              <div
                key={chat.assignment_id}
                className={`agent-chat-item ${isActive ? 'agent-chat-item--active' : ''}`}
                onClick={() => setSelectedChat(chat)}
              >
                <div className="agent-chat-avatar">
                  <Phone size={16} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div className="flex justify-between">
                    <span className="agent-chat-name">{phone}</span>
                    {last?.time && <span className="agent-chat-time">{formatTime(last.time)}</span>}
                  </div>
                  {last?.text && (
                    <div className="agent-chat-preview">{last.text}</div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Right: message panel */}
      <div className="agent-main">
        {selectedChat ? (
          <>
            <div className="agent-chat-header">
              <div className="flex items-center gap-3">
                <div className="agent-chat-avatar">
                  <Phone size={18} />
                </div>
                <div>
                  <div style={{ fontWeight: 600 }}>{formatPhoneNumber(selectedChat.jid)}</div>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>{selectedChat.session_id}</div>
                </div>
              </div>
              <MoreVertical size={20} style={{ color: 'var(--text-muted)' }} />
            </div>

            <MessageThread
              key={selectedChat.jid}
              sessionId={selectedChat.session_id}
              jid={selectedChat.jid}
              onNewMessage={(msg) => {
                setLastMessages(prev => ({
                  ...prev,
                  [selectedChat.jid]: { text: msg.text, time: msg.time }
                }));
              }}
            />

            <MessageInput
              sessionId={selectedChat.session_id}
              jid={selectedChat.jid}
            />
          </>
        ) : (
          <div className="agent-empty-state">
            <div className="agent-empty-icon">💬</div>
            <h3>Select a chat to start</h3>
            <p>Choose a conversation from the sidebar</p>
          </div>
        )}
      </div>
    </div>
  );
}
