import { createContext, useContext, useState, useCallback } from 'react';

const API_BASE = 'http://localhost:3000';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const stored = JSON.parse(localStorage.getItem('wapi_user') || 'null');
  const [user, setUser] = useState(stored);

  const login = useCallback(async (username, password) => {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Login failed');
    const userObj = {
      token: data.token,
      role: data.role,
      user_id: data.user_id,
      username: data.username,
    };
    localStorage.setItem('wapi_user', JSON.stringify(userObj));
    setUser(userObj);
    return userObj;
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('wapi_user');
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
