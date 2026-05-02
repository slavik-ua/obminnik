/** @type {import('next').NextConfig} */
const nextConfig = {
  logging: {
    fetches: {
      fullUrl: true,
    },
  },
  output: 'standalone',
  // This fixes the HMR warning you received
  devIndicators: {
    appIsrStatus: false,
  },
  experimental: {
    allowedDevOrigins: [
      '127.0.0.1',
      'localhost',
      'localhost:3000',
      'localhost:3001',
      '127.0.0.1:3000',
      '127.0.0.1:3001',
    ],
  },
};

export default nextConfig;
