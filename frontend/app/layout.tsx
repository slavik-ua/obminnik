import './globals.css';

import { ToastProvider } from '../components/Toast';
import { AuthProvider } from '../context/AuthContext';

export const metadata = {
  title: 'OBMINNIK | Professional Orderbook',
  description: 'High-frequency trading interface',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="bg-slate-950 text-slate-200 antialiased">
        <ToastProvider>
          <AuthProvider>{children}</AuthProvider>
        </ToastProvider>
      </body>
    </html>
  );
}
