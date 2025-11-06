// Supply microservice server
// Run: node supply_server.js
// Or: node supply_server.js --port=8080

const http = require('http');
const { calculateTotalSupply, calculateCirculatingSupply } = require('./calculate_supply.js');

// Get port from command line or default to 3000
const args = process.argv.slice(2);
const portArg = args.find(arg => arg.startsWith('--port='));
const PORT = portArg ? parseInt(portArg.split('=')[1]) : 3000;

// Cache configuration
const CACHE_TTL = 60000; // 1 minute in milliseconds

// Cache storage
const cache = {
  totalSupply: { value: null, timestamp: 0 },
  circulatingSupply: { value: null, timestamp: 0 }
};

// In-flight promise storage to prevent duplicate calculations
const inFlight = {
  totalSupply: null,
  circulatingSupply: null
};

// Generic cached getter that prevents re-entrant calculations
async function getCached(cacheKey, calculateFn) {
  const now = Date.now();

  // Return cached value if still valid
  if (cache[cacheKey].value !== null && (now - cache[cacheKey].timestamp) < CACHE_TTL) {
    return cache[cacheKey].value;
  }

  // If a calculation is already in progress, return that promise
  if (inFlight[cacheKey]) {
    return await inFlight[cacheKey];
  }

  // Start a new calculation
  inFlight[cacheKey] = (async () => {
    try {
      const value = await calculateFn();
      cache[cacheKey] = { value, timestamp: Date.now() };
      return value;
    } finally {
      // Clear in-flight promise after completion
      inFlight[cacheKey] = null;
    }
  })();

  return await inFlight[cacheKey];
}

const server = http.createServer(async (req, res) => {
  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  if (req.method !== 'GET') {
    res.writeHead(405, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Method not allowed' }));
    return;
  }

  try {
    if (req.url === '/total_supply') {
      const totalSupply = await getCached('totalSupply', calculateTotalSupply);
      res.writeHead(200, { 'Content-Type': 'text/plain' });
      res.end(totalSupply.toString());
    } else if (req.url === '/circulating_supply') {
      const circulatingSupply = await getCached('circulatingSupply', calculateCirculatingSupply);
      res.writeHead(200, { 'Content-Type': 'text/plain' });
      res.end(circulatingSupply.toString());
    } else {
      res.writeHead(404, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Not found' }));
    }
  } catch (error) {
    console.error('Error processing request:', error);
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      error: 'Internal server error',
      message: error.message
    }));
  }
});

server.listen(PORT, () => {
  console.log(`Supply microservice running on http://localhost:${PORT}`);
  console.log(`  GET http://localhost:${PORT}/total_supply`);
  console.log(`  GET http://localhost:${PORT}/circulating_supply`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM signal received: closing HTTP server');
  server.close(() => {
    console.log('HTTP server closed');
  });
});
