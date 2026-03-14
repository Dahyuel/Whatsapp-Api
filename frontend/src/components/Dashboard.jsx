import { useApi } from '../api';
import { Activity, Users, MessageSquare, AlertCircle, Smartphone } from 'lucide-react';

function StatCard({ title, value, icon: Icon, color }) {
  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <h3 style={{ margin: 0, color: 'var(--text-muted)' }}>{title}</h3>
        <div style={{ padding: '8px', borderRadius: '12px', backgroundColor: `${color}20` }}>
          <Icon size={24} color={color} />
        </div>
      </div>
      <div style={{ fontSize: '2.5rem', fontWeight: 700 }}>{value}</div>
    </div>
  );
}

export default function Dashboard() {
  const { data: health, loading: healthLoading } = useApi('/health');
  const { data: sessions, loading: sessionsLoading } = useApi('/sessions');
  const { data: queue, loading: queueLoading } = useApi('/queue');

  const isLoading = healthLoading || sessionsLoading || queueLoading;

  if (isLoading) {
    return (
      <div className="animate-fade-in">
        <h1>Dashboard Overview</h1>
        <div className="card text-center" style={{ padding: '3rem' }}>
          <Activity size={32} className="text-muted" style={{ animation: 'spin 2s linear infinite' }} />
          <p className="mt-4">Loading core metrics...</p>
        </div>
      </div>
    );
  }

  const activeSessions = sessions?.data?.filter(s => s.status === 'CONNECTED')?.length || 0;
  const totalSessions = sessions?.data?.length || 0;
  
  // Assuming queue endpoint returns an array or an object we can reduce
  const totalQueuedMessages = Array.isArray(queue?.data) 
    ? queue.data.reduce((acc, q) => acc + (q.pending || 0), 0)
    : 0;

  return (
    <div className="animate-fade-in">
      <div className="flex justify-between items-center mb-4">
        <h1>System Overview</h1>
        {health?.status === 'ok' ? (
          <span className="badge badge-success flex items-center gap-2">
            <div style={{ width: 8, height: 8, borderRadius: '50%', background: 'currentColor' }}></div>
            API Online (v{health?.version || '1.0.0'})
          </span>
        ) : (
          <span className="badge badge-danger flex items-center gap-2">
            <AlertCircle size={14} />
            API Offline
          </span>
        )}
      </div>

      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', 
        gap: '1.5rem',
        marginBottom: '2rem'
      }}>
        <StatCard 
          title="Active Sessions" 
          value={`${activeSessions} / ${totalSessions}`}
          icon={Smartphone}
          color="var(--success)"
        />
        <StatCard 
          title="Queued Messages" 
          value={totalQueuedMessages}
          icon={MessageSquare}
          color="var(--warning)"
        />
        <StatCard 
          title="Backend Uptime" 
          value="OK"
          icon={Activity}
          color="var(--primary)"
        />
      </div>

      <div className="card">
        <h2>Quick Actions</h2>
        <div className="flex gap-4 mt-4">
          <a href="/sessions" className="btn btn-primary">Manage Sessions</a>
          <a href="/messaging" className="btn btn-outline">Send Message Test</a>
        </div>
      </div>
    </div>
  );
}
