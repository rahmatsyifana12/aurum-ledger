<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, logout, session } from './api'

const route = useRoute()
const router = useRouter()
const user = ref(null)
const authenticated = computed(() => session.authenticated && !route.meta.guest)

watch(() => route.fullPath, async () => {
  if (!authenticated.value) { user.value = null; return }
  if (!user.value) user.value = await api('/me').catch(() => null)
}, { immediate: true })

async function signOut() {
  await logout()
  user.value = null
  router.push('/login')
}
</script>

<template>
  <div class="app-shell">
    <header v-if="authenticated" class="topbar">
      <RouterLink to="/metals" class="brand-mark"><span>A</span> Aurum Ledger</RouterLink>
      <div class="user-menu">
        <span class="user-name">{{ user?.full_name }}</span>
        <button class="link-button" @click="signOut">Log out</button>
      </div>
    </header>
    <main :class="{ 'main-content': authenticated }">
      <RouterView />
    </main>
  </div>
</template>

