<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { apiGet, apiPost } from "../lib/api";

type Container = {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  created: string;
  composeProject?: string;
};

const loading = ref(true);
const error = ref<string | null>(null);
const containers = ref<Container[]>([]);
const selected = ref<Container | null>(null);

const logs = ref<string>("");
const logsError = ref<string | null>(null);
let es: EventSource | null = null;

const grouped = computed(() => {
  const map = new Map<string, Container[]>();
  for (const c of containers.value) {
    const key = c.composeProject ?? "ungrouped";
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(c);
  }
  return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]));
});

async function refresh() {
  loading.value = true;
  error.value = null;
  try {
    containers.value = await apiGet<Container[]>("/api/docker/containers");
  } catch (e: any) {
    error.value = String(e?.message ?? e);
  } finally {
    loading.value = false;
  }
}

async function act(id: string, action: "start" | "stop" | "restart") {
  try {
    await apiPost(`/api/docker/containers/${encodeURIComponent(id)}/${action}`);
    await refresh();
  } catch (e: any) {
    alert(`Action failed: ${String(e?.message ?? e)}`);
  }
}

function openLogs(c: Container) {
  selected.value = c;
  logs.value = "";
  logsError.value = null;

  if (es) { es.close(); es = null; }
  const url = `/api/docker/containers/${encodeURIComponent(c.id)}/logs/stream?tail=200`;
  es = new EventSource(url);

  es.onmessage = (ev) => {
    logs.value += ev.data + "\n";
    // keep it from exploding too much
    if (logs.value.length > 300_000) logs.value = logs.value.slice(-250_000);
    // autoscroll is handled by template ref below
  };

  es.onerror = () => {
    logsError.value = "SSE connection error (maybe container stopped or backend restarted).";
  };
}

function closeLogs() {
  selected.value = null;
  if (es) { es.close(); es = null; }
}

onMounted(refresh);
onUnmounted(() => { if (es) es.close(); });
</script>

<template>
  <div>
    <div class="head">
      <h2>Docker</h2>
      <button class="btn" @click="refresh" :disabled="loading">Refresh</button>
    </div>

    <div v-if="error" class="err">❌ {{ error }}</div>
    <div v-else-if="loading">Lade…</div>

    <div v-else class="groups">
      <div v-for="[group, list] in grouped" :key="group" class="group">
        <div class="groupTitle">
          <span class="gname">{{ group }}</span>
          <span class="gcount">{{ list.length }}</span>
        </div>

        <div v-for="c in list" :key="c.id" class="card">
          <div class="topline">
            <div class="name">{{ c.name }}</div>
            <div class="badge" :class="c.state">{{ c.state }}</div>
          </div>

          <div class="meta">
            <div class="row"><span>Image</span><span class="mono">{{ c.image }}</span></div>
            <div class="row"><span>Status</span><span>{{ c.status }}</span></div>
            <div class="row"><span>ID</span><span class="mono">{{ c.id.slice(0, 12) }}</span></div>
          </div>

          <div class="actions">
            <button class="btn" @click="act(c.id, 'start')">Start</button>
            <button class="btn" @click="act(c.id, 'stop')">Stop</button>
            <button class="btn" @click="act(c.id, 'restart')">Restart</button>
            <button class="btn2" @click="openLogs(c)">Logs</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="selected" class="drawer">
      <div class="drawerHead">
        <div>
          <div class="drawerTitle">{{ selected.name }}</div>
          <div class="drawerSub mono">{{ selected.id }}</div>
        </div>
        <button class="btn" @click="closeLogs">Close</button>
      </div>

      <div v-if="logsError" class="err" style="margin-top:10px">{{ logsError }}</div>

      <pre class="logbox">{{ logs }}</pre>
    </div>
  </div>
</template>

<style scoped>
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

h2 {
  margin: 0;
}

.btn,
.btn2 {
  border: 1px solid #26435e;
  background: #101e2e;
  color: #e8eef6;
  padding: 8px 10px;
  border-radius: 10px;
  font-size: 13px;
}

.btn2 {
  border-color: #3b5c7a;
}

.btn:disabled {
  opacity: .6;
}

.err {
  background: #2a1414;
  border: 1px solid #6a2a2a;
  padding: 10px;
  border-radius: 10px;
  margin: 10px 0;
}

.groups {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 12px;
}

.groupTitle {
  display: flex;
  justify-content: space-between;
  align-items: center;
  opacity: .9;
  padding: 0 4px;
}

.gname {
  font-weight: 700;
}

.gcount {
  opacity: .7;
}

.card {
  border: 1px solid #1b2a3a;
  background: #0f1620;
  border-radius: 12px;
  padding: 12px;
  margin-top: 10px;
}

.topline {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: center;
}

.name {
  font-weight: 700;
}

.badge {
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 12px;
  border: 1px solid #2a3b4f;
  opacity: .9;
}

.badge.running {
  background: #0f2a18;
  border-color: #1d5a33;
}

.badge.exited {
  background: #2a1414;
  border-color: #6a2a2a;
}

.meta {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  opacity: .9;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
  opacity: .9;
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 12px;
}

.drawer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  top: 64px;
  background: rgba(11, 15, 20, 0.98);
  border-top: 1px solid #1b2a3a;
  padding: 12px;
  z-index: 50;
  display: flex;
  flex-direction: column;
}

.drawerHead {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.drawerTitle {
  font-weight: 800;
}

.drawerSub {
  opacity: .7;
  font-size: 12px;
  word-break: break-all;
}

.logbox {
  margin-top: 12px;
  border: 1px solid #1b2a3a;
  background: #0f1620;
  border-radius: 12px;
  padding: 12px;
  overflow: auto;
  flex: 1;
  white-space: pre-wrap;
}
</style>
