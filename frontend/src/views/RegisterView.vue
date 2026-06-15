<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, session } from '../api'
const router=useRouter();const form=reactive({full_name:'',username:'',password:''});const error=ref('');const loading=ref(false)
async function submit(){loading.value=true;error.value='';try{session.save(await api('/auth/register',{method:'POST',body:JSON.stringify(form)}));router.push('/metals')}catch(e){error.value=e.message}finally{loading.value=false}}
</script>
<template><div class="auth-page"><section class="auth-intro"><div class="logo-large">A</div><p class="eyebrow">Build your private ledger</p><h1>Know the value<br /><em>you hold.</em></h1><p>A focused portfolio for gold, silver, and every precious holding.</p></section><form class="auth-card" @submit.prevent="submit"><p class="eyebrow">Get started</p><h2>Create account</h2><p v-if="error" class="alert">{{ error }}</p><label>Full name<input v-model.trim="form.full_name" required autocomplete="name" /></label><label>Username<input v-model.trim="form.username" required minlength="3" autocomplete="username" /></label><label>Password<input v-model="form.password" required minlength="8" type="password" autocomplete="new-password" /><small>At least 8 characters</small></label><button class="button primary wide" :disabled="loading">{{ loading?'Creating...':'Create account' }}</button><p class="auth-switch">Already registered? <RouterLink to="/login">Sign in</RouterLink></p></form></div></template>

