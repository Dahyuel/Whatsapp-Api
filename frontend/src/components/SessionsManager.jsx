import { useState } from 'react';
import { useApi, fetchApi } from '../api';
import toast from 'react-hot-toast';
import { Plus, QrCode, Trash2, PowerOff, RefreshCw } from 'lucide-react';

export default function SessionsManager() {
  const { data, loading, refetch } = useApi('/sessions');
  const [newSessionId, setNewSessionId] = useState('');
  const [qrCode, setQrCode] = useState(null);
  const [activeTab, setActiveTab] = useState(null); // the session ID currently showing QR
  const [isCreating, setIsCreating] = useState(false);

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!newSessionId.trim()) return;
    try {
      setIsCreating(true);
      await fetchApi('/sessions', {
        method: 'POST',
        body: JSON.stringify({ id: newSessionId.trim() })
      });
      toast.success('Session created!');
      setNewSessionId('');
      refetch();
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsCreating(false);
    }
  };

  const handleGetQr = async (sessionId) => {
    try {
      const res = await fetchApi(`/sessions/${sessionId}/login`, { method: 'POST' });
      if (res.qr_code) {
        setQrCode(res.qr_code);
        setActiveTab(sessionId);
      } else {
        toast.error('Could not load QR. Is the session already connected?');
      }
    } catch (err) {
      toast.error(err.message);
    }
  };

  const handleDelete = async (sessionId) => {
    if (!confirm('Are you sure you want to delete this session? This will remove all local data for it.')) return;
    try {
      await fetchApi(`/sessions/${sessionId}`, { method: 'DELETE' });
      toast.success('Session deleted');
      if (activeTab === sessionId) {
        setQrCode(null);
        setActiveTab(null);
      }
      refetch();
    } catch (err) {
      toast.error(err.message);
    }
  };

  const handleDisconnect = async (sessionId) => {
    try {
      await fetchApi(`/sessions/${sessionId}/logout`, { method: 'POST' });
      toast.success('Session disconnected');
      refetch();
    } catch (err) {
      toast.error(err.message);
    }
  };

  if (loading) return <div className="animate-fade-in"><p>Loading sessions...</p></div>;

  const sessions = data?.sessions || [];

  return (
    <div className="animate-fade-in">
      <div className="flex justify-between items-center mb-4">
        <h1>Sessions Manager</h1>
        <button onClick={refetch} className="btn btn-outline">
          <RefreshCw size={16} /> Refresh
        </button>
      </div>

      <div className="card">
        <h3>Create New Session</h3>
        <form onSubmit={handleCreate} className="flex gap-4 mt-4">
          <input
            type="text"
            className="form-input"
            placeholder="e.g., brand_account_1"
            value={newSessionId}
            onChange={e => setNewSessionId(e.target.value)}
            style={{ maxWidth: '300px' }}
          />
          <button type="submit" className="btn btn-primary" disabled={isCreating}>
            <Plus size={16} /> Create
          </button>
        </form>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '1.5rem' }}>
        {sessions.map(session => (
          <div key={session.id} className="card relative flex-col justify-between" style={{ display: 'flex', minHeight: '200px' }}>
            <div>
              <div className="flex justify-between items-center mb-2">
                <h3 style={{ margin: 0 }}>{session.id}</h3>
                <span className={`badge ${session.status === 'CONNECTED' ? 'badge-success' : 'badge-warning'}`}>
                  {session.status}
                </span>
              </div>
              <p className="text-sm">JID: {session.jid || 'Not connected'}</p>
            </div>

            {activeTab === session.id && qrCode && (
              <div className="text-center mt-4 bg-white p-2 rounded-lg inline-block self-center">
                <img src={`data:image/png;base64,${qrCode}`} alt="QR Code" style={{ width: '200px', height: '200px' }} />
                <p className="text-sm text-gray-800 mt-2">Scan with WhatsApp</p>
              </div>
            )}

            <div className="flex gap-2 mt-4 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
              {session.status !== 'CONNECTED' && (
                <button 
                  className="btn btn-primary" 
                  style={{ flex: 1 }}
                  onClick={() => handleGetQr(session.id)}
                >
                  <QrCode size={16} /> QR Code
                </button>
              )}
              {session.status === 'CONNECTED' && (
                <button 
                  className="btn btn-outline" 
                  style={{ flex: 1 }}
                  onClick={() => handleDisconnect(session.id)}
                >
                  <PowerOff size={16} /> Logout
                </button>
              )}
              <button 
                className="btn btn-danger"
                onClick={() => handleDelete(session.id)}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        ))}
        {sessions.length === 0 && (
          <p>No active sessions found.</p>
        )}
      </div>
    </div>
  );
}
