import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '@/views/Dashboard.vue'
import Validators from '@/views/Validators.vue'
import Staking from '@/views/Staking.vue'
import Governance from '@/views/Governance.vue'
import Blocks from '@/views/Blocks.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: Dashboard
    },
    {
      path: '/validators',
      name: 'validators',
      component: Validators
    },
    {
      path: '/staking',
      name: 'staking',
      component: Staking
    },
    {
      path: '/governance',
      name: 'governance',
      component: Governance
    },
    {
      path: '/blocks',
      name: 'blocks',
      component: Blocks
    }
  ]
})

export default router