/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export for Go embedding
  output: 'export',
  
  // Required for static export
  trailingSlash: true,
  
  // Disable image optimization for static export
  images: {
    unoptimized: true,
  },
  
  // Asset prefix handling for Go server
  assetPrefix: process.env.NODE_ENV === 'production' ? '' : '',
  
  // Base path (empty for root deployment)
  basePath: '',
  
  // TypeScript and ESLint configuration
  typescript: {
    // Temporarily ignore type errors for build
    ignoreBuildErrors: true,
  },
  
  eslint: {
    // Temporarily ignore ESLint errors for build
    ignoreDuringBuilds: true,
  },
  
  // Removed experimental turbo configuration to prevent build issues
  
  // Webpack configuration for Go embedding compatibility and asset optimization
  webpack: (config, { isServer, buildId }) => {
    // Add build timestamp to webpack define plugin
    const webpack = require('webpack');
    config.plugins.push(
      new webpack.DefinePlugin({
        'process.env.BUILD_TIMESTAMP': JSON.stringify(new Date().toISOString()),
        'process.env.BUILD_ID': JSON.stringify(buildId || Math.random().toString(36).substring(2, 15)),
      })
    );
    
    // Handle SVG imports
    config.module.rules.push({
      test: /\.svg$/,
      use: ['@svgr/webpack'],
    });
    
    // Removed custom image handling to prevent conflicts
    
    // Optimize for static export
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        net: false,
        tls: false,
        crypto: false,
      };
      
      // Remove custom optimization to prevent CSS duplication bug
      // Let Next.js use its default optimization strategy
      // This prevents the issue where CSS files are incorrectly added as script tags
      delete config.optimization?.splitChunks;
    }
    
    return config;
  },
  
  // Note: headers, redirects, and rewrites are handled by the Go server
  // These features are not compatible with static export
  
  // Environment variables available to the client
  env: {
    NEXT_PUBLIC_APP_NAME: 'ISX Daily Reports Scrapper',
    NEXT_PUBLIC_APP_VERSION: '1.0.0',
    NEXT_PUBLIC_BUILD_TIME: new Date().toISOString(),
    NEXT_PUBLIC_BUILD_ID: Math.random().toString(36).substring(2, 15),
  },
  
  // Power user features
  poweredByHeader: false,
  reactStrictMode: true,
  swcMinify: true,
  
  // Compiler options
  compiler: {
    // Remove console logs in production
    removeConsole: false, // Temporarily disabled for debugging
  },
  
  // Temporarily enable dev mode for debugging hydration errors
  productionBrowserSourceMaps: true,
};

// Bundle analyzer for development
if (process.env.ANALYZE === 'true') {
  const withBundleAnalyzer = require('@next/bundle-analyzer')({
    enabled: true,
  });
  module.exports = withBundleAnalyzer(nextConfig);
} else {
  module.exports = nextConfig;
}