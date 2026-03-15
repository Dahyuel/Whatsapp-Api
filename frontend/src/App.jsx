import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import { Toaster, toast } from 'react-hot-toast';
import { LayoutDashboard, Smartphone, MessageSquare, Webhook, Settings, LogOut, Terminal, Users, GitBranch } from 'lucide-react';

import { AuthProvider, useAuth } from './context/AuthContext';
import Login from './components/Login';
import Dashboard from './components/Dashboard';
import SessionsManager from './components/SessionsManager';
import Messaging from './components/Messaging';
import WebhooksManager from './components/WebhooksManager';
import ApiSettings from './components/ApiSettings';
import ApiTester from './components/ApiTester';
import UserManagement from './components/admin/UserManagement';
import ChatAssignment from './components/admin/ChatAssignment';
import ChatUI from './components/agent/ChatUI';

import './index.css';

function AdminSidebar() {
  const location = useLocation();
  const { user, logout } = useAuth();

  const links = [
    { path: '/', label: 'Overview', icon: LayoutDashboard },
    { path: '/sessions', label: 'Sessions', icon: Smartphone },
    { path: '/messaging', label: 'Messaging', icon: MessageSquare },
    { path: '/tester', label: 'API Tester', icon: Terminal },
    { path: '/webhooks', label: 'Webhooks', icon: Webhook },
    { path: '/admin/users', label: 'Users', icon: Users },
    { path: '/admin/chats', label: 'Chat Assignment', icon: GitBranch },
    { path: '/settings', label: 'Settings', icon: Settings },
  ];

  return (
    <aside className="sidebar">
      <div className="mb-4 flex items-center gap-2">
        <div style={{ background: 'var(--primary)', padding: '8px', borderRadius: '8px' }}>
          <MessageSquare size={24} color="white" />
        </div>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.1rem' }}>WhatsApp API</h2>
          <span className="badge badge-primary" style={{ fontSize: '0.65rem', padding: '2px 8px' }}>Admin</span>
        </div>
      </div>

      <nav style={{ flex: 1, marginTop: '2rem' }}>
        <ul style={{ listStyle: 'none' }}>
          {links.map((link) => {
            const Icon = link.icon;
            const isActive = location.pathname === link.path;
            return (
              <li key={link.path} style={{ marginBottom: '0.5rem' }}>
                <Link
                  to={link.path}
                  style={{
                    display: 'flex', alignItems: 'center', gap: '1rem',
                    padding: '0.75rem 1rem', borderRadius: '8px',
                    color: isActive ? 'white' : 'var(--text-muted)',
                    background: isActive ? 'var(--primary)' : 'transparent',
                    textDecoration: 'none', fontWeight: isActive ? 600 : 500,
                    transition: 'all 0.2s'
                  }}
                >
                  <Icon size={20} />
                  {link.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      <div style={{ marginTop: 'auto', borderTop: '1px solid var(--border)', paddingTop: '1.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontWeight: 600, fontSize: '0.875rem' }}>{user?.username}</div>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Administrator</div>
          </div>
          <button className="icon-btn" onClick={logout} title="Sign out">
            <LogOut size={18} />
          </button>
        </div>
      </div>
    </aside>
  );
}

function AdminApp() {
  const [hasApiKey, setHasApiKey] = useState(!!localStorage.getItem('whatsapp_api_key'));

  useEffect(() => {
    const handleStorageChange = () => setHasApiKey(!!localStorage.getItem('whatsapp_api_key'));
    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, []);

  if (!hasApiKey) {
    return <ApiSettings onSave={() => setHasApiKey(true)} isInitial={true} />;
  }

  return (
    <div className="app-container">
      <AdminSidebar />
      <main className="main-content">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/sessions" element={<SessionsManager />} />
          <Route path="/messaging" element={<Messaging />} />
          <Route path="/tester" element={<ApiTester />} />
          <Route path="/webhooks" element={<WebhooksManager />} />
          <Route path="/admin/users" element={<UserManagement />} />
          <Route path="/admin/chats" element={<ChatAssignment />} />
          <Route path="/settings" element={<ApiSettings onSave={() => toast.success('Settings saved')} />} />
        </Routes>
      </main>
    </div>
  );
}

function AppInner() {
  const { user } = useAuth();

  if (!user) return <Login />;
  if (user.role === 'admin') return <AdminApp />;
  if (user.role === 'agent') return <ChatUI />;

  return <Login />;
}

function App() {
  return (
    <AuthProvider>
      <Router>
        <AppInner />
        <Toaster
          position="top-right"
          toastOptions={{
            style: {
              background: 'var(--bg-card)',
              color: 'var(--text-main)',
              border: '1px solid var(--border)'
            }
          }}
        />
      </Router>
    </AuthProvider>
  );
}

export default App;
