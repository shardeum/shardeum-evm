<template>
  <div id="app" class="min-h-screen bg-gray-50">
    <nav class="bg-white shadow-sm border-b">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between items-center h-16">
          <div class="flex items-center space-x-8">
            <div class="flex-shrink-0">
              <h1 class="text-xl font-bold text-shardeum-primary">Shardeum Dashboard</h1>
            </div>
            <div class="hidden md:block">
              <div class="ml-10 flex items-baseline space-x-4">
                <router-link
                  to="/"
                  class="text-gray-900 hover:text-shardeum-primary px-3 py-2 rounded-md text-sm font-medium"
                  :class="{ 'text-shardeum-primary': $route.path === '/' }"
                >
                  Dashboard
                </router-link>
                <router-link
                  to="/validators"
                  class="text-gray-900 hover:text-shardeum-primary px-3 py-2 rounded-md text-sm font-medium"
                  :class="{ 'text-shardeum-primary': $route.path === '/validators' }"
                >
                  Validators
                </router-link>
                <router-link
                  to="/staking"
                  class="text-gray-900 hover:text-shardeum-primary px-3 py-2 rounded-md text-sm font-medium"
                  :class="{ 'text-shardeum-primary': $route.path === '/staking' }"
                >
                  Staking
                </router-link>
                <router-link
                  to="/governance"
                  class="text-gray-900 hover:text-shardeum-primary px-3 py-2 rounded-md text-sm font-medium"
                  :class="{ 'text-shardeum-primary': $route.path === '/governance' }"
                >
                  Governance
                </router-link>
                <router-link
                  to="/blocks"
                  class="text-gray-900 hover:text-shardeum-primary px-3 py-2 rounded-md text-sm font-medium"
                  :class="{ 'text-shardeum-primary': $route.path === '/blocks' }"
                >
                  Blocks
                </router-link>
              </div>
            </div>
          </div>
          <div class="flex items-center space-x-4">
            <WalletConnect />
          </div>
        </div>
      </div>
    </nav>

    <main class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import WalletConnect from '@/components/WalletConnect.vue'
</script>