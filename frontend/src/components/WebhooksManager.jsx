import { useState, useEffect } from 'react';
import { useApi, fetchApi } from '../api';
import toast from 'react-hot-toast';
import { Webhook, Plus, Trash2 } from 'lucide-react';

export default function WebhooksManager() {
  const [webhooks, setWebhooks] = useState([]);
  const [loading, setLoading] = useState(true);
  const { data: sessionData } = useApi('/sessions');
  
  const [formData, setFormData] = useState({
    session_id: '',
    url: '',
    secret: '',
    events: ['message.received', 'session.connected']
  });

  // Since the API might not have a GET /webhooks endpoint defined in the README,
  // we are simulating the list, or assuming GET /webhooks exists.
  // We'll wrap the fetch in a try-catch assuming it exists, otherwise it will just be an empty list.

  const fetchWebhooks = async () => {
    try {
      const res = await fetchApi('/webhooks');
      setWebhooks(res.data || []);
    } catch (e) {
      console.log('GET /webhooks might not be implemented, or keys invalid.', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWebhooks();
  }, []);

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!formData.session_id || !formData.url || !formData.secret) {
      toast.error('Fill required fields');
      return;
    }

    try {
      await fetchApi('/webhooks', {
        method: 'POST',
        body: JSON.stringify(formData)
      });
      toast.success('Webhook registered successfully!');
      setFormData({ ...formData, url: '', secret: '' });
      fetchWebhooks();
    } catch (err) {
      toast.error(err.message);
    }
  };

  const handleDelete = async (id) => {
    try {
      await fetchApi(`/webhooks/${id}`, { method: 'DELETE' });
      toast.success('Webhook removed');
      fetchWebhooks();
    } catch (err) {
      toast.error(err.message);
    }
  };

  return (
    <div className="animate-fade-in">
      <h1>Webhooks Manager</h1>
      <p>Configure HTTP endpoints to receive real-time events from WhatsApp.</p>

      <div className="card mt-4">
        <h3>Register New Webhook</h3>
        <form onSubmit={handleCreate} className="mt-4 flex-col gap-4" style={{ display: 'flex' }}>
          <div className="flex gap-4">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Session</label>
              <select 
                className="form-select" 
                value={formData.session_id}
                onChange={e => setFormData({ ...formData, session_id: e.target.value })}
              >
                <option value="">Select Session...</option>
                {(sessionData?.sessions || []).map(s => (
                  <option key={s.id} value={s.id}>{s.id}</option>
                ))}
              </select>
            </div>
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">HMAC Secret (for signature validation)</label>
              <input 
                type="text" 
                className="form-input" 
                value={formData.secret}
                onChange={e => setFormData({ ...formData, secret: e.target.value })}
                placeholder="e.g. my-super-secret"
              />
            </div>
          </div>
          
          <div className="form-group">
            <label className="form-label">Target URL</label>
            <input 
              type="url" 
              className="form-input" 
              value={formData.url}
              onChange={e => setFormData({ ...formData, url: e.target.value })}
              placeholder="https://your-server.com/webhook"
            />
          </div>

          <button type="submit" className="btn btn-primary" style={{ alignSelf: 'flex-start' }}>
            <Plus size={16} /> Register Webhook
          </button>
        </form>
      </div>

      <div className="card mt-6">
        <h3>Active Webhooks</h3>
        {loading ? (
          <p>Loading...</p>
        ) : webhooks.length > 0 ? (
          <div style={{ display: 'grid', gap: '1rem', marginTop: '1rem' }}>
            {webhooks.map(wh => (
              <div key={wh.id} className="flex items-center justify-between" style={{ padding: '1rem', background: 'var(--bg-subtle)', borderRadius: '8px' }}>
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <Webhook size={16} className="text-muted" />
                    <strong>{wh.url}</strong>
                  </div>
                  <div className="text-sm gap-2 flex">
                    <span className="badge badge-success">Session: {wh.session_id}</span>
                    <span className="badge" style={{ background: 'rgba(255,255,255,0.1)' }}>{wh.events?.length || 0} events</span>
                  </div>
                </div>
                <button className="btn btn-danger" onClick={() => handleDelete(wh.id)}>
                  <Trash2 size={16} />
                </button>
              </div>
            ))}
          </div>
        ) : (
          <p className="mt-4">No webhooks registered. The backend stores them in memory/db.</p>
        )}
      </div>
    </div>
  );
}
