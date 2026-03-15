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

function formatDateHeader(ts) {
  if (!ts) return '';
  try {
    const d = typeof ts === 'number' ? new Date(ts * 1000) : new Date(ts);
    const now = new Date();
    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);

    if (d.toDateString() === now.toDateString()) {
      return 'Today';
    } else if (d.toDateString() === yesterday.toDateString()) {
      return 'Yesterday';
    } else {
      return d.toLocaleDateString(undefined, {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      });
    }
  } catch { return ''; }
}

function getMessageText(msg) {
  // If it's a raw WhatsMeow event payload from SSE, it might have Message/Info
  if (msg.Message) {
    const conv = msg.Message.conversation || msg.Message.extendedTextMessage?.text;
    if (conv) return conv;
    if (msg.Message.imageMessage) return '🖼 Image';
    if (msg.Message.videoMessage) return '🎥 Video';
    if (msg.Message.audioMessage) return '🎵 Audio';
    if (msg.Message.documentMessage) return '📄 ' + (msg.Message.documentMessage.fileName || 'Document');
    if (msg.Message.stickerMessage) return '🤩 Sticker';
    return '[Message]';
  }
  // Otherwise it's our DB ChatMessageRow
  if (msg.Text) return msg.Text;
  if (msg.MsgType === 'image') return '🖼 Image';
  if (msg.MsgType === 'video') return '🎥 Video';
  if (msg.MsgType === 'audio') return '🎵 Audio';
  if (msg.MsgType === 'document') return '📄 Document';
  if (msg.MsgType === 'sticker') return '🤩 Sticker';
  return '[Message]';
}

export default function MessageThread({ sessionId, jid, onNewMessage, lastNewMessage, refreshTrigger }) {
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
        // Send to parent for latest message preview
        onNewMessage?.({ text: getMessageText(last), time: last.Timestamp || last.Info?.Timestamp });
      }
    } catch (e) {
      setMessages([]);
    } finally {
      setLoading(false);
    }
  }, [jid, sessionId]); // removed onNewMessage to avoid loops

  // When a new message comes in over SSE (from parent), immediately fetch again
  useEffect(() => {
    fetchMessages();
  }, [fetchMessages, lastNewMessage, refreshTrigger]);

  useEffect(() => {
    setLoading(true);
    setMessages([]);
    fetchMessages();

    // Poll every 3 seconds for new messages as a fallback
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
      {(() => {
        let lastDateString = null;
        
        return messages.map((msg, i) => {
          // Fallback between DB format and raw WhatsMeow SSE format
          const fromMe = msg.IsFromMe !== undefined ? msg.IsFromMe : msg.Info?.IsFromMe;
          const text = getMessageText(msg);
          const timeStr = msg.Timestamp || msg.Info?.Timestamp;
          const senderJid = msg.SenderJID || msg.Info?.Sender;
          const isGroup = jid.includes('@g.us');
          
          // Fix string timestamp vs unix seconds
          let parsedTime = timeStr;
          if (typeof timeStr === 'string' && timeStr.includes('T')) {
            parsedTime = new Date(timeStr).getTime() / 1000;
          } else if (typeof timeStr === 'string') { // e.g., "2024-03-15..." 
             parsedTime = new Date(timeStr).getTime() / 1000;
          }
          
          const time = formatTime(parsedTime);
          const dateHeaderStr = formatDateHeader(parsedTime);
          
          let showDateHeader = false;
          if (dateHeaderStr !== lastDateString) {
            showDateHeader = true;
            lastDateString = dateHeaderStr;
          }

          return (
            <div key={i} style={{ display: 'flex', flexDirection: 'column' }}>
              {showDateHeader && (
                <div className="date-header-wrap">
                  <span className="date-header-pill">{dateHeaderStr}</span>
                </div>
              )}
              <div className={`message-bubble-wrap ${fromMe ? 'message-bubble-wrap--out' : 'message-bubble-wrap--in'}`}>
            <div className={`message-bubble ${fromMe ? 'message-bubble--out' : 'message-bubble--in'}`}>
              {!fromMe && isGroup && senderJid && (
                <div style={{ fontSize: '0.7rem', fontWeight: 'bold', color: 'var(--primary)', marginBottom: '2px' }}>
                  {formatPhoneNumber(senderJid)}
                </div>
              )}
              {msg.MsgType === 'image' || msg.Message?.imageMessage ? (
                <img 
                  src={`http://localhost:3000/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}/media?session=${sessionId}`} 
                  alt="Image" 
                  style={{ maxWidth: '100%', borderRadius: '4px', marginBottom: '4px' }}
                />
              ) : msg.MsgType === 'video' || msg.Message?.videoMessage ? (
                <video 
                  controls
                  src={`http://localhost:3000/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}/media?session=${sessionId}`}
                  style={{ maxWidth: '100%', borderRadius: '4px', marginBottom: '4px' }}
                />
              ) : msg.MsgType === 'audio' || msg.Message?.audioMessage ? (
                <audio 
                  controls
                  src={`http://localhost:3000/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}/media?session=${sessionId}`}
                  style={{ maxWidth: '100%', marginBottom: '4px' }}
                />
              ) : msg.MsgType === 'document' || msg.Message?.documentMessage ? (
                <a 
                  href={`http://localhost:3000/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}/media?session=${sessionId}`}
                  download
                  className="btn btn-primary"
                  style={{ display: 'inline-block', marginBottom: '4px', textDecoration: 'none', color: '#fff', fontSize: '0.8rem', padding: '0.5rem 1rem' }}
                >
                  📄 Download Document
                </a>
              ) : msg.MsgType === 'sticker' || msg.Message?.stickerMessage ? (
                <img 
                  src={`http://localhost:3000/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}/media?session=${sessionId}`} 
                  alt="Sticker" 
                  style={{ width: '120px', background: 'transparent', marginBottom: '4px' }}
                />
              ) : (
                <span>{text}</span>
              )}
              {text && msg.MsgType !== 'text' && !msg.Message?.conversation && !msg.Message?.extendedTextMessage ? (
                 <div style={{fontSize: '0.85rem', marginTop: 4}}>{text}</div>
              ) : null}
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '4px', marginTop: '2px' }}>
                <span className="message-time">{time}</span>
                {fromMe && (
                  <span style={{ fontSize: '10px', color: (msg.Status === 'READ' || msg.Status === 'PLAYED') ? '#53bdeb' : 'var(--text-muted)', marginBottom: '-3px' }}>
                    {msg.Status === 'READ' || msg.Status === 'PLAYED' ? '✓✓' : msg.Status === 'DELIVERY_ACK' ? '✓✓' : msg.Status === 'SERVER_ACK' ? '✓' : '🕓'}
                  </span>
                )}
              </div>

              {/* Message Actions */}
              <div className="message-actions" style={{ position: 'absolute', top: '4px', right: '4px', display: 'flex', gap: '4px', opacity: 0, transition: 'opacity 0.2s' }}>
                <button 
                  onClick={() => {
                    const toJid = prompt("Enter JID to forward to (e.g. 1234567890@s.whatsapp.net):");
                    if (toJid) {
                      fetchApi(`/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}/forward?session=${sessionId}`, {
                        method: 'POST',
                        body: JSON.stringify({ to_jid: toJid })
                      }).then(() => alert('Message forwarded!'))
                      .catch(e => alert('Failed to forward: ' + e.message));
                    }
                  }}
                  title="Forward"
                  style={{ background: 'rgba(0,0,0,0.5)', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer', padding: '2px 6px', fontSize: '0.7rem' }}
                >
                  ➦
                </button>
                {fromMe && (
                  <button 
                    onClick={() => {
                      if (window.confirm("Revoke this message?")) {
                        fetchApi(`/agent/chats/${encodeURIComponent(jid)}/messages/${msg.MessageID || msg.Info?.ID}?session=${sessionId}`, {
                          method: 'DELETE'
                        }).then(() => alert('Revoke queued.'))
                        .catch(e => alert('Failed to revoke: ' + e.message));
                      }
                    }}
                    title="Revoke (Delete for everyone)"
                    style={{ background: 'rgba(255,0,0,0.6)', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer', padding: '2px 6px', fontSize: '0.7rem' }}
                  >
                    🗑
                  </button>
                )}
              </div>
            </div>
            </div>
          </div>
          );
        });
      })()}
      <div ref={bottomRef} style={{ height: 1 }} />
    </div>
  );
}
