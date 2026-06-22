<script setup>
import { reactive, watch } from 'vue'
import { PreciousMetalBrand, preciousMetalBrands } from '../constants/brands'
import { PreciousMetalType, preciousMetalTypes } from '../constants/metalTypes'

const props = defineProps({ initial: { type: Object, default: () => ({}) }, saving: Boolean })
const emit = defineEmits(['submit'])
const form = reactive({ type: PreciousMetalType.GOLD, code: '', brand: PreciousMetalBrand.ANTAM, bought_price: '', buy_price: '', buyback_price: '', weight: '' })
watch(() => props.initial, value => Object.assign(form, value || {}, { code: value?.code || '' }), { immediate: true })

function submit() {
  const code = form.code.trim()
  emit('submit', {
    ...form,
    code: code || null,
    bought_price: Number(form.bought_price),
    buy_price: Number(form.buy_price || 0),
    buyback_price: Number(form.buyback_price || 0),
    weight: Number(form.weight)
  })
}
</script>

<template>
  <form class="panel metal-form" @submit.prevent="submit">
    <div class="form-grid">
      <label>
        Metal type
        <select v-model="form.type" required>
          <option v-for="type in preciousMetalTypes" :key="type" :value="type">{{ type }}</option>
        </select>
      </label>
      <label>
        Brand
        <select v-model="form.brand" required>
          <option v-for="brand in preciousMetalBrands" :key="brand" :value="brand">{{ brand }}</option>
        </select>
      </label>
      <label>Code<input v-model.trim="form.code" placeholder="LM001" /><small>Optional product, certificate, or internal code</small></label>
      <label>Weight (gram)<input v-model="form.weight" required min="0.001" step="0.001" type="number" inputmode="decimal" placeholder="10" /></label>
      <label>Bought price (IDR)<input v-model="form.bought_price" required min="0" step="1" type="number" inputmode="numeric" placeholder="15000000" /><small>Total price paid when purchased</small></label>
    </div>
    <p class="form-note">Sell and buyback prices are retrieved automatically for the selected brand and exact weight.</p>
    <button class="button primary wide" :disabled="saving">{{ saving ? 'Saving...' : 'Save precious metal' }}</button>
  </form>
</template>
