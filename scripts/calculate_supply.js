// save as supply.js, run: node supply.js
// Node 18+ required (has global fetch)

const LCD = "https://rest.shardeum.org";
const DENOM = "ashm";
const DECIMALS = 18;                 // display precision for human-readable output

// Module accounts to exclude from circulation.
// Adjust if your chain uses different names.
const EXCLUDED_MODULES = new Set([
  "bonded_tokens_pool",
  "not_bonded_tokens_pool",
  "distribution",
  "gov",
  "fee_collector",
  "mint",
  "ibc-transfer",
  "staking",
  "transfer",
  "erc20",
  "evm",
  "feemarket",
  "precisebank"
]);

// Add any explicit non-circulating user addresses here (foundation, vesting, treasury, etc.)
const NON_CIRC_ADDRESSES = [
    "shardeum1lg3dd9d0zmvkpcrmrkr5guc6dn8s2k43gr3p6j", // Foundation Cold Storage
    "shardeum16r5ta5cpvcnkfgrkxvgnnnjwfd4eqlhend3ly8", // Team Cold Storage
    "shardeum1twv0y2s9426yqxmmh3u3r30q86ha8j0wcu7ajf", // Ecosystem Cold Storage
    "shardeum1uev04qcth9hgxkrvcs2dnntuyazlppg9nr8ul6"  // Sale Cold Storage
];

function toIntSafe(str) {
  // LCD usually returns integer strings; if decimal, drop fraction best-effort.
  if (str.includes(".")) return BigInt(str.split(".")[0] || "0");
  return BigInt(str);
}
function fmt(raw, decimals = DECIMALS) {
  // Convert BigInt to decimal string with proper precision
  const divisor = 10n ** BigInt(decimals);
  const integerPart = raw / divisor;
  const fractionalPart = raw % divisor;
  const fractionalStr = fractionalPart.toString().padStart(decimals, '0');
  return `${integerPart}.${fractionalStr}`;
}
async function getJSON(url) {
  const r = await fetch(url);
  if (!r.ok) {
    const t = await r.text().catch(() => "");
    throw new Error(`GET ${url} -> ${r.status} ${r.statusText} ${t ? "- " + t : ""}`);
  }
  return r.json();
}
async function supplyOf(denom) {
  const u = `${LCD}/cosmos/bank/v1beta1/supply`;
  const j = await getJSON(u);
  // The response has a "supply" array of coins
  for (const coin of j.supply || []) {
    if (coin.denom === denom) {
      return toIntSafe(coin.amount);
    }
  }
  return 0n;
}
async function moduleAccounts() {
  const u = `${LCD}/cosmos/auth/v1beta1/module_accounts`;
  const j = await getJSON(u);
  const out = [];
  for (const acc of j.accounts || []) {
    const name = acc?.name ?? acc?.base_vesting_account?.name ?? acc?.value?.name;
    const base = acc?.base_account ?? acc?.base_vesting_account?.base_account ?? acc?.value?.base_account ?? {};
    const address = base?.address;
    if (name && address) out.push({ name, address });
  }
  return out;
}
async function balanceOf(address, denom) {
  const u = `${LCD}/cosmos/bank/v1beta1/balances/${encodeURIComponent(address)}/by_denom?denom=${encodeURIComponent(denom)}`;
  try {
    const j = await getJSON(u);
    return toIntSafe(j?.balance?.amount ?? "0");
  } catch {
    return 0n;
  }
}
async function communityPool(denom) {
  const u = `${LCD}/cosmos/distribution/v1beta1/community_pool`;
  const j = await getJSON(u);
  for (const coin of j.pool || []) {
    if (coin.denom === denom) return toIntSafe(coin.amount);
  }
  return 0n;
}

// Returns the total supply in SHM (converted from ashm by dividing by 10^18)
async function calculateTotalSupply() {
  const totalAshm = await supplyOf(DENOM);
  const divisor = 10n ** BigInt(DECIMALS);
  return totalAshm / divisor;
}

// Returns the circulating supply in SHM (converted from ashm by dividing by 10^18)
async function calculateCirculatingSupply() {
  // Fetch total supply, module accounts, and community pool in parallel
  const [T, mods, C] = await Promise.all([
    supplyOf(DENOM),
    moduleAccounts(),
    communityPool(DENOM)
  ]);

  // M: sum of excluded module account balances
  // Fetch all excluded module balances in parallel
  const excludedMods = mods.filter(m => EXCLUDED_MODULES.has(m.name));
  const moduleBalances = await Promise.all(
    excludedMods.map(m => balanceOf(m.address, DENOM))
  );
  const M = moduleBalances.reduce((sum, balance) => sum + balance, 0n);

  // E: explicit non-circulating addresses
  // Fetch all excluded address balances in parallel
  const excludedBalances = await Promise.all(
    NON_CIRC_ADDRESSES.map(a => balanceOf(a, DENOM))
  );
  const E = excludedBalances.reduce((sum, balance) => sum + balance, 0n);

  const circulating = T - M - C - E > 0n ? T - M - C - E : 0n;

  // Convert from ashm to SHM
  const divisor = 10n ** BigInt(DECIMALS);
  return circulating / divisor;
}

// Export functions for use in other modules
module.exports = {
  calculateTotalSupply,
  calculateCirculatingSupply
};
