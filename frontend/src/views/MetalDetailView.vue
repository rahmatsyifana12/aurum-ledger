<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
const route=useRoute();const router=useRouter();const metal=ref(null);const error=ref('');const refreshing=ref(false)
const money=value=>new Intl.NumberFormat('id-ID',{style:'currency',currency:'IDR',maximumFractionDigits:0}).format(value||0)
const gain=computed(()=>metal.value?metal.value.buyback_price-metal.value.bought_price:0)
async function load(){try{metal.value=await api(`/metals/${route.params.id}`)}catch(e){error.value=e.message}}
async function refresh(){refreshing.value=true;error.value='';try{metal.value=await api(`/metals/${route.params.id}/refresh-price`,{method:'POST'})}catch(e){error.value=e.message}finally{refreshing.value=false}}
async function remove(){if(!confirm('Delete this precious metal from your ledger?'))return;await api(`/metals/${route.params.id}`,{method:'DELETE'});router.push('/metals')}
onMounted(load)
</script>
<template><div class="page narrow"><RouterLink to="/metals" class="back-link">← Portfolio</RouterLink><p v-if="error" class="alert">{{ error }}</p><div v-if="metal"><div class="detail-title"><div class="metal-icon large">{{ metal.type.slice(0,2).toUpperCase() }}</div><div><p class="eyebrow">{{ metal.brand }}</p><h1>{{ metal.type }}</h1><p class="muted">{{ metal.code ? `${metal.code} · ` : '' }}{{ metal.weight }} gram holding</p></div></div><section class="hero-value"><span>Current buyback value</span><strong>{{ money(metal.buyback_price) }}</strong><small :class="gain>=0?'positive':'negative'">{{ gain>=0?'+':'' }}{{ money(gain) }} from purchase</small></section><section class="panel detail-grid"><div><span>Code</span><strong>{{ metal.code || 'Not set' }}</strong></div><div><span>Price paid</span><strong>{{ money(metal.bought_price) }}</strong></div><div><span>Brand buy price</span><strong>{{ money(metal.buy_price) }}</strong></div><div><span>Brand buyback price</span><strong>{{ money(metal.buyback_price) }}</strong></div><div><span>Last price update</span><strong>{{ metal.price_updated_at?new Date(metal.price_updated_at).toLocaleString('id-ID'):'Not synced' }}</strong></div></section><div class="action-row"><button class="button primary" :disabled="refreshing" @click="refresh">{{ refreshing?'Refreshing...':'Refresh brand price' }}</button><RouterLink :to="`/metals/${metal.id}/edit`" class="button secondary">Edit</RouterLink><button class="button danger" @click="remove">Delete</button></div></div></div></template>
