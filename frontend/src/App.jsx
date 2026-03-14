import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import { Toaster, toast } from 'react-hot-toast';
import { LayoutDashboard, Smartphone, MessageSquare, Webhook, Settings, LogOut, Terminal } from 'lucide-react';

import Dashboard from './components/Dashboard';
import SessionsManager from './components/SessionsManager';
import Messaging from './components/Messaging';
import WebhooksManager from './components/WebhooksManager';
import ApiSettings from './components/ApiSettings';
import ApiTester from './components/ApiTester';

import './index.css';

function Sidebar() {
  const location = useLocation();

  const links = [
    { path: '/', label: 'Overview', icon: LayoutDashboard },
    { path: '/sessions', label: 'Sessions', icon: Smartphone },
    { path: '/messaging', label: 'Messaging', icon: MessageSquare },
    { path: '/tester', label: 'API Tester', icon: Terminal },
    { path: '/webhooks', label: 'Webhooks', icon: Webhook },
    { path: '/settings', label: 'Settings', icon: Settings },
  ];

  return (
    <aside className="sidebar">
      <div className="mb-4 flex items-center gap-2">
        <div style={{ background: 'var(--primary)', padding: '8px', borderRadius: '8px' }}>
          <MessageSquare size={24} color="white" />
        </div>
        <h2 style={{ margin: 0 }}>WhatsApp API</h2>
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
                    display: 'flex',
                    alignItems: 'center',
                    gap: '1rem',
                    padding: '0.75rem 1rem',
                    borderRadius: '8px',
                    color: isActive ? 'white' : 'var(--text-muted)',
                    background: isActive ? 'var(--primary)' : 'transparent',
                    textDecoration: 'none',
                    fontWeight: isActive ? 600 : 500,
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
        <div style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>
          Powered by WhatsMeow
        </div>
      </div>
    </aside>
  );
}

function App() {
  const [hasApiKey, setHasApiKey] = useState(!!localStorage.getItem('whatsapp_api_key'));

  useEffect(() => {
    const handleStorageChange = () => {
      setHasApiKey(!!localStorage.getItem('whatsapp_api_key'));
    };
    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, []);

  if (!hasApiKey) {
    return <ApiSettings onSave={() => setHasApiKey(true)} isInitial={true} />;
  }

  return (
    <Router>
      <div className="app-container">
        <Sidebar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/sessions" element={<SessionsManager />} />
            <Route path="/messaging" element={<Messaging />} />
            <Route path="/tester" element={<ApiTester />} />
            <Route path="/webhooks" element={<WebhooksManager />} />
            <Route path="/settings" element={<ApiSettings onSave={() => toast.success('Settings saved')} />} />
          </Routes>
        </main>
      </div>
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
  );
}

export default App;
