<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import MetalForm from '../components/MetalForm.vue'
const route=useRoute();const router=useRouter();const metal=ref(null);const error=ref('');const saving=ref(false);const editing=Boolean(route.params.id)
onMounted(async()=>{if(editing){try{metal.value=await api(`/metals/${route.params.id}`)}catch(e){error.value=e.message}}})
async function save(data){saving.value=true;error.value='';try{const saved=await api(editing?`/metals/${route.params.id}`:'/metals',{method:editing?'PUT':'POST',body:JSON.stringify(data)});router.push(`/metals/${saved.id}`)}catch(e){error.value=e.message}finally{saving.value=false}}
</script>
<template><div class="page narrow"><RouterLink :to="editing?`/metals/${route.params.id}`:'/metals'" class="back-link">← Back</RouterLink><div class="page-heading"><div><p class="eyebrow">{{ editing?'Update holding':'New holding' }}</p><h1>{{ editing?'Edit precious metal':'Add precious metal' }}</h1></div></div><p v-if="error" class="alert">{{ error }}</p><MetalForm v-if="!editing||metal" :initial="metal||{}" :saving="saving" @submit="save" /></div></template>

