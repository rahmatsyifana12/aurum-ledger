<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, session } from '../api'

const router = useRouter()
const form = reactive({ username: '', password: '' })
const error = ref('')
const loading = ref(false)
async function submit() {
  loading.value = true; error.value = ''
  try { session.save(await api('/auth/login', { method: 'POST', body: JSON.stringify(form) })); router.push('/metals') }
  catch (e) { error.value = e.message }
  finally { loading.value = false }
}
</script>

<template><div class="auth-page"><section class="auth-intro"><div class="logo-large">A</div><p class="eyebrow">Personal wealth, clearly kept</p><h1>Your metals.<br /><em>Your ledger.</em></h1><p>Track what you own, what you paid, and what it is worth today.</p></section><form class="auth-card" @submit.prevent="submit"><p class="eyebrow">Welcome back</p><h2>Sign in</h2><p v-if="error" class="alert">{{ error }}</p><label>Username<input v-model.trim="form.username" required autocomplete="username" /></label><label>Password<input v-model="form.password" required type="password" autocomplete="current-password" /></label><button class="button primary wide" :disabled="loading">{{ loading ? 'Signing in...' : 'Sign in' }}</button><p class="auth-switch">New to Aurum? <RouterLink to="/register">Create account</RouterLink></p></form></div></template>

