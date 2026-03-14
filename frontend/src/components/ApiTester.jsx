import { useState } from 'react';
import { fetchApi } from '../api';
import toast from 'react-hot-toast';
import { Terminal, Play } from 'lucide-react';

export default function ApiTester() {
  const [endpoint, setEndpoint] = useState('/messages/text');
  const [method, setMethod] = useState('POST');
  const [payload, setPayload] = useState('{\n  "session": "session1",\n  "to": "15551234567@s.whatsapp.net",\n  "text": "Hello from API Tester!"\n}');
  const [response, setResponse] = useState('');
  const [loading, setLoading] = useState(false);

  const handleTest = async (e) => {
    e.preventDefault();
    if (!endpoint) return;

    try {
      setLoading(true);
      setResponse('Loading...');
      
      let parsedPayload = undefined;
      if (['POST', 'PUT', 'PATCH'].includes(method) && payload.trim()) {
        try {
          parsedPayload = JSON.parse(payload);
        } catch (err) {
          toast.error('Invalid JSON payload');
          setResponse(`Error: Invalid JSON payload\n${err.message}`);
          setLoading(false);
          return;
        }
      }

      const res = await fetchApi(endpoint, {
        method,
        body: parsedPayload ? JSON.stringify(parsedPayload) : undefined
      });
      
      setResponse(JSON.stringify(res, null, 2));
      toast.success('API request successful');
    } catch (err) {
      setResponse(JSON.stringify({ error: err.message, status: err.status }, null, 2));
      toast.error('API request failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="animate-fade-in">
      <h1>API Tester</h1>
      <p>Directly test any backend endpoint with custom JSON payloads.</p>

      <div style={{ display: 'flex', gap: '2rem', marginTop: '2rem', alignItems: 'flex-start' }}>
        
        {/* Request Panel */}
        <div className="card" style={{ flex: 1 }}>
          <div className="flex items-center gap-2 mb-4">
            <Terminal size={20} className="text-muted" />
            <h3 style={{ margin: 0 }}>Request Configuration</h3>
          </div>

          <form onSubmit={handleTest}>
            <div className="flex gap-4 mb-4">
              <div style={{ flex: '0 0 120px' }}>
                <label className="form-label">Method</label>
                <select 
                  className="form-select" 
                  value={method} 
                  onChange={e => setMethod(e.target.value)}
                  style={{ fontWeight: 'bold' }}
                >
                  <option value="GET">GET</option>
                  <option value="POST">POST</option>
                  <option value="PUT">PUT</option>
                  <option value="PATCH">PATCH</option>
                  <option value="DELETE">DELETE</option>
                </select>
              </div>
              <div style={{ flex: 1 }}>
                <label className="form-label">Endpoint</label>
                <input 
                  type="text" 
                  className="form-input" 
                  value={endpoint} 
                  onChange={e => setEndpoint(e.target.value)}
                  placeholder="/messages/text"
                  required
                />
              </div>
            </div>

            {['POST', 'PUT', 'PATCH'].includes(method) && (
              <div className="form-group">
                <label className="form-label">JSON Payload</label>
                <textarea 
                  className="form-textarea" 
                  style={{ fontFamily: 'monospace', minHeight: '200px', fontSize: '14px' }}
                  value={payload}
                  onChange={e => setPayload(e.target.value)}
                  spellCheck="false"
                />
              </div>
            )}

            <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%', marginTop: '1rem' }}>
              <Play size={16} fill="currentColor" /> {loading ? 'Executing...' : 'Send Request'}
            </button>
          </form>
        </div>

        {/* Response Panel */}
        <div className="card" style={{ flex: 1, minHeight: '400px', display: 'flex', flexDirection: 'column' }}>
          <h3 className="mb-4">Response Output</h3>
          <pre style={{ 
            flex: 1, 
            background: 'rgba(15, 23, 42, 0.8)', 
            padding: '1.5rem', 
            borderRadius: '8px', 
            overflowX: 'auto',
            fontFamily: 'monospace',
            fontSize: '14px',
            color: response.includes('"error"') ? 'var(--danger)' : 'var(--success)',
            margin: 0,
            border: '1px solid var(--border)'
          }}>
            {response || 'Waiting for request...'}
          </pre>
        </div>

      </div>
    </div>
  );
}
