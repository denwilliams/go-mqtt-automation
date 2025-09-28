import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'export',
  trailingSlash: true,
  distDir: '../static',
  images: {
    unoptimized: true
  }
};

export default nextConfig;
