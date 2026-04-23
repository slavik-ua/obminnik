import { ApiError } from '../types';
import { safeStorage } from '../utils/storage';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

/**
 * Custom event to notify the application when a session has expired
 * or the user is unauthorized.
 */
export const UNAUTHORIZED_EVENT = 'app:unauthorized';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  // Guard for Server Side Rendering (localStorage only exists in browser)
  const token = safeStorage.getItem('token');
  
  const headers = new Headers(options.headers);
  headers.set('Content-Type', 'application/json');
  
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const config: RequestInit = {
    ...options,
    headers,
  };

  try {
    const response = await fetch(`${API_BASE}${path}`, config);

    // Handle session expiration
    if (response.status === 401) {
      window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT));
      throw new Error('Session expired. Please login again.');
    }

    // Handle Rate Limiting (from your Go middleware)
    if (response.status === 429) {
      throw new Error('Too many requests. Please slow down.');
    }

    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
      const errorData = data as ApiError;
      throw new Error(errorData.detail || errorData.title || `Request failed with status ${response.status}`);
    }

    return data as T;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unexpected network error occurred');
  }
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  
  post: <T>(path: string, body: unknown) => 
    request<T>(path, { 
      method: 'POST', 
      body: JSON.stringify(body) 
    }),
};