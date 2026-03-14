import { useState, useEffect } from 'react';
import { Key } from 'lucide-react';

export default function ApiSettings({ onSave, isInitial }) {
  const [key, setKey] = useState('');

  useEffect(() => {
    setKey(localStorage.getItem('whatsapp_api_key') || '');
  }, []);

  const handleSave = (e) => {
    e.preventDefault();
    localStorage.setItem('whatsapp_api_key', key);
    // Dispatch storage event manually for same-tab updates
    window.dispatchEvent(new Event('storage'));
    if (onSave) onSave();
  };

  const containerStyle = isInitial ? {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'var(--bg-dark)'
  } : {};

  return (
    <div style={containerStyle} className="animate-fade-in">
      <div className="card" style={{ maxWidth: '400px', width: '100%', margin: isInitial ? '0' : '0 auto' }}>
        <div className="text-center mb-4">
          <div style={{ 
            display: 'inline-flex',
            padding: '1rem',
            background: 'var(--primary)',
            borderRadius: '16px',
            marginBottom: '1rem'
          }}>
            <Key size={32} color="white" />
          </div>
          <h2>Authentication</h2>
          <p>Please enter your X-API-Key to connect to the WhatsApp API server.</p>
        </div>

        <form onSubmit={handleSave}>
          <div className="form-group">
            <label className="form-label">API Key</label>
            <input
              type="password"
              className="form-input"
              value={key}
              onChange={e => setKey(e.target.value)}
              placeholder="Enter your secret key"
              required
            />
          </div>
          <button type="submit" className="btn btn-primary" style={{ width: '100%' }}>
            {isInitial ? 'Connect' : 'Save Settings'}
          </button>
        </form>
      </div>
    </div>
  );
}
