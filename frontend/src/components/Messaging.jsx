import { useState } from 'react';
import { fetchApi, useApi } from '../api';
import toast from 'react-hot-toast';
import { Send, Image, MessageSquare, MapPin } from 'lucide-react';

export default function Messaging() {
  const { data: sessionData } = useApi('/sessions');
  const sessions = sessionData?.sessions || [];
  
  const [activeTab, setActiveTab] = useState('text');
  const [loading, setLoading] = useState(false);
  
  // Form State
  const [sessionId, setSessionId] = useState('');
  const [to, setTo] = useState('');
  const [text, setText] = useState('');
  const [mediaUrl, setMediaUrl] = useState('');
  const [caption, setCaption] = useState('');

  const formatJid = (phone) => {
    let clean = phone.replace(/[^0-9]/g, '');
    if (!clean.includes('@')) {
      clean += '@s.whatsapp.net';
    }
    return clean;
  };

  const handleSendText = async (e) => {
    e.preventDefault();
    if (!sessionId || !to || !text) {
      toast.error('Please fill required fields');
      return;
    }

    try {
      setLoading(true);
      const res = await fetchApi('/messages/text', {
        method: 'POST',
        body: JSON.stringify({
          session: sessionId,
          to: formatJid(to),
          text
        })
      });
      toast.success(`Message queued! ID: ${res.tracking_id}`);
      setText('');
    } catch (err) {
      toast.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSendMediaUrl = async (e) => {
    e.preventDefault();
    if (!sessionId || !to || !mediaUrl) {
      toast.error('Please fill required fields');
      return;
    }

    try {
      setLoading(true);
      const res = await fetchApi('/messages/media', {
        method: 'POST',
        body: JSON.stringify({
          session: sessionId,
          to: formatJid(to),
          media_type: 'image', // simplified for demo
          url: mediaUrl,
          caption
        })
      });
      toast.success(`Media queued! ID: ${res.tracking_id}`);
      setMediaUrl('');
      setCaption('');
    } catch (err) {
      toast.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="animate-fade-in">
      <h1>Messaging Sender</h1>
      <p>Dispatch messages via the Anti-Ban Queue. Delay logic is handled by the backend.</p>

      <div className="card" style={{ maxWidth: '600px', marginTop: '2rem' }}>
        <div className="flex gap-4 mb-6" style={{ borderBottom: '1px solid var(--border)', paddingBottom: '1rem' }}>
          <button 
            className={`btn ${activeTab === 'text' ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => setActiveTab('text')}
          >
            <MessageSquare size={16} /> Text
          </button>
          <button 
            className={`btn ${activeTab === 'media' ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => setActiveTab('media')}
          >
            <Image size={16} /> Media (URL)
          </button>
        </div>

        <div className="form-group">
          <label className="form-label">Sender Session</label>
          <select 
            className="form-select" 
            value={sessionId} 
            onChange={e => setSessionId(e.target.value)}
          >
            <option value="">Select a session...</option>
            {sessions.map(s => (
              <option key={s.id} value={s.id}>{s.id} ({s.status})</option>
            ))}
          </select>
        </div>

        <div className="form-group">
          <label className="form-label">Recipient Phone (with country code)</label>
          <input 
            type="text" 
            className="form-input" 
            placeholder="e.g. 15551234567" 
            value={to}
            onChange={e => setTo(e.target.value)}
          />
        </div>

        {activeTab === 'text' && (
          <form onSubmit={handleSendText}>
            <div className="form-group">
              <label className="form-label">Message Text</label>
              <textarea 
                className="form-textarea" 
                rows="4" 
                placeholder="Hello from API!"
                value={text}
                onChange={e => setText(e.target.value)}
              />
            </div>
            <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
              <Send size={16} /> {loading ? 'Sending...' : 'Send Text'}
            </button>
          </form>
        )}

        {activeTab === 'media' && (
          <form onSubmit={handleSendMediaUrl}>
            <div className="form-group">
              <label className="form-label">Media URL (Direct Link)</label>
              <input 
                type="url" 
                className="form-input" 
                placeholder="https://example.com/image.jpg"
                value={mediaUrl}
                onChange={e => setMediaUrl(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Caption (Optional)</label>
              <input 
                type="text" 
                className="form-input" 
                placeholder="Check this out"
                value={caption}
                onChange={e => setCaption(e.target.value)}
              />
            </div>
            <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
              <Send size={16} /> {loading ? 'Sending...' : 'Send Media'}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
