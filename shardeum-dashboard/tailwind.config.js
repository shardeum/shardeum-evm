/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
        shardeum: {
          primary: '#6366f1',
          secondary: '#a855f7',
          accent: '#06b6d4',
        }
      },
    },
  },
  plugins: [],
}