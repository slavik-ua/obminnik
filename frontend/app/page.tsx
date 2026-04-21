'use client';

import { useAuth } from '../context/AuthContext';
import { Login } from '../components/Auth/Login';
import { Dashboard } from '../components/Dashboard';

export default function Home() {
  const { isAuthenticated } = useAuth();

  // If user is logged in, show the Dashboard, otherwise show Login
  return (
    <main>
      {isAuthenticated ? <Dashboard /> : <Login />}
    </main>
  );
}