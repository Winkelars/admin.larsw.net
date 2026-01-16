<script setup lang="ts">
import { ref, onMounted } from "vue";
import { apiGet } from "../lib/api";

type Health = { ok: boolean; version: string; time: string };

const health = ref<Health | null>(null);
const err = ref<string | null>(null);

onMounted(async () => {
  try {
    health.value = await apiGet<Health>("/api/health");
  } catch (e: any) {
    err.value = String(e?.message ?? e);
  }
});
</script>

<template>
  <div>
    <h2>Dashboard</h2>
    <p style="opacity:.8">Grundgerüst läuft. Als erstes ist Docker voll implementiert.</p>

    <div class="card">
      <div class="row">
        <span>Status</span>
        <span v-if="health">✅ API ok ({{ health.version }})</span>
        <span v-else-if="err">❌ API error: {{ err }}</span>
        <span v-else>…</span>
      </div>
      <div class="row" v-if="health">
        <span>Zeit</span><span>{{ health.time }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card {
  border: 1px solid #1b2a3a;
  background: #0f1620;
  border-radius: 10px;
  padding: 12px;
  margin-top: 12px;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 0;
}

h2 {
  margin: 0 0 8px 0;
}
</style>
