import { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { apiClient } from '../api/client';

interface AdminContextType {
  isAdmin: boolean;
  apiKey: string | null;
  login: (apiKey: string) => Promise<boolean>;
  logout: () => void;
}

const AdminContext = createContext<AdminContextType | undefined>(undefined);

const STORAGE_KEY = 'adminApiKey';

export function AdminProvider({ children }: { children: ReactNode }) {
  const [apiKey, setApiKey] = useState<string | null>(null);

  // Load API key from localStorage on mount
  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      setApiKey(stored);
    }
  }, []);

  const login = async (key: string): Promise<boolean> => {
    try {
      // Validate the API key by calling a protected admin endpoint through the API client
      await apiClient.validateAdminKey(key);

      // Key is valid, store it
      localStorage.setItem(STORAGE_KEY, key);
      setApiKey(key);
      return true;
    } catch {
      return false;
    }
  };

  const logout = () => {
    localStorage.removeItem(STORAGE_KEY);
    setApiKey(null);
  };

  return (
    <AdminContext.Provider
      value={{
        isAdmin: !!apiKey,
        apiKey,
        login,
        logout,
      }}
    >
      {children}
    </AdminContext.Provider>
  );
}

export function useAdmin() {
  const context = useContext(AdminContext);
  if (!context) {
    throw new Error('useAdmin must be used within AdminProvider');
  }
  return context;
}
