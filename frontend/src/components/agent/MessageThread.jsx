import { useState, useEffect, useRef, useCallback } from 'react';
import { fetchApi } from '../../api';

function formatPhoneNumber(jid) {
  return jid?.replace(/@.+$/, '') || jid || '';
}

function formatTime(ts) {
  if (!ts) return '';
  try {
    const d = typeof ts === 'number' ? new Date(ts * 1000) : new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch { return ''; }
}

function getMessageText(msg) {
  const conv = msg.Message?.conversation || msg.Message?.extendedTextMessage?.text;
  if (conv) return conv;
  if (msg.Message?.imageMessage) return '🖼 Image';
  if (msg.Message?.videoMessage) return '🎥 Video';
  if (msg.Message?.audioMessage) return '🎵 Audio';
  if (msg.Message?.documentMessage) return '📄 ' + (msg.Message.documentMessage.fileName || 'Document');
  return '[Message]';
}

export default function MessageThread({ sessionId, jid, onNewMessage }) {
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  const bottomRef = useRef(null);

  const fetchMessages = useCallback(async () => {
    try {
      const data = await fetchApi(`/agent/chats/${encodeURIComponent(jid)}/messages?session=${sessionId}&count=50`);
      const msgs = data.messages || [];
      setMessages(msgs);
      if (msgs.length > 0) {
        const last = msgs[msgs.length - 1];
        onNewMessage?.({ text: getMessageText(last), time: last.Info?.Timestamp });
      }
    } catch (e) {
      setMessages([]);
    } finally {
      setLoading(false);
    }
  }, [jid, sessionId, onNewMessage]);

  useEffect(() => {
    setLoading(true);
    setMessages([]);
    fetchMessages();

    // Poll every 3 seconds for new messages
    const interval = setInterval(fetchMessages, 3000);
    return () => clearInterval(interval);
  }, [fetchMessages]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  if (loading) {
    return (
      <div className="message-thread-loading">
        <div className="typing-dots"><span /><span /><span /></div>
      </div>
    );
  }

  return (
    <div className="message-thread">
      {messages.length === 0 && (
        <div style={{ textAlign: 'center', color: 'var(--text-muted)', marginTop: '2rem', fontSize: '0.9rem' }}>
          No messages yet
        </div>
      )}
      {messages.map((msg, i) => {
        const fromMe = msg.Info?.IsFromMe;
        const text = getMessageText(msg);
        const time = formatTime(msg.Info?.Timestamp);
        return (
          <div key={i} className={`message-bubble-wrap ${fromMe ? 'message-bubble-wrap--out' : 'message-bubble-wrap--in'}`}>
            <div className={`message-bubble ${fromMe ? 'message-bubble--out' : 'message-bubble--in'}`}>
              <span>{text}</span>
              <span className="message-time">{time}</span>
            </div>
          </div>
        );
      })}
      <div ref={bottomRef} />
    </div>
  );
}
