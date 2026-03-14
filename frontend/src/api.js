import { useState, useEffect, useCallback } from 'react';

// Central API handler
// The user can configure the local base URL if needed, but defaults to same host / backend port
const API_BASE = 'http://localhost:3000';

export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.status = status;
  }
}

export const fetchApi = async (endpoint, options = {}) => {
  const apiKey = localStorage.getItem('whatsapp_api_key') || '';
  
  const headers = {
    'Content-Type': 'application/json',
    ...(apiKey ? { 'X-API-Key': apiKey } : {}),
    ...options.headers,
  };

  // If we are sending FormData, remove Content-Type so browser can set it with boundary
  if (options.body instanceof FormData) {
    delete headers['Content-Type'];
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  const isJson = response.headers.get('content-type')?.includes('application/json');
  const data = isJson ? await response.json() : await response.text();

  if (!response.ok) {
    throw new ApiError(data?.error || data || 'API request failed', response.status);
  }

  return data;
};

// Hook for data fetching
export function useApi(endpoint, dependencies = []) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const execute = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await fetchApi(endpoint);
      setData(result);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [endpoint]);

  useEffect(() => {
    execute();
  }, [execute, ...dependencies]);

  return { data, loading, error, refetch: execute };
}
