/**
 * A safe wrapper for localStorage to prevent "The operation is insecure" 
 * or "Access Denied" errors in some browser environments (e.g. Incognito, 
 * strictly blocked cookies, or certain proxy/iframe contexts).
 */
export const safeStorage = {
  getItem: (key: string): string | null => {
    try {
      if (typeof window === 'undefined') return null;
      return localStorage.getItem(key);
    } catch (e) {
      console.warn(`LocalStorage access failed for key "${key}":`, e);
      return null;
    }
  },
  
  setItem: (key: string, value: string): void => {
    try {
      if (typeof window === 'undefined') return;
      localStorage.setItem(key, value);
    } catch (e) {
      console.error(`LocalStorage write failed for key "${key}":`, e);
    }
  },
  
  removeItem: (key: string): void => {
    try {
      if (typeof window === 'undefined') return;
      localStorage.removeItem(key);
    } catch (e) {
      console.error(`LocalStorage remove failed for key "${key}":`, e);
    }
  }
};
