<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { apiGet, apiPost } from "../lib/api";

type Project = {
  name: string;
  path: string;
  hasEnv: boolean;
  containerTotal: number;
  containerRun: number;
  containerStop: number;
};

type FileResp = { name: string; path: string; text: string };

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const error = ref<string | null>(null);
const projects = ref<Project[]>([]);

const selected = ref<Project | null>(null);
const composeText = ref<string>("");
const envText = ref<string>("");
const output = ref<string>("");
const tab = ref<"compose" | "env" | "output">("compose");

const queryProject = computed(() => String(route.query.project ?? ""));

async function refresh() {
  loading.value = true;
  error.value = null;
  try {
    projects.value = await apiGet<Project[]>("/api/production/projects");
    // auto-open if /production?project=name
    if (queryProject.value) {
      const p = projects.value.find(x => x.name === queryProject.value);
      if (p) await openProject(p);
    }
  } catch (e: any) {
    error.value = String(e?.message ?? e);
  } finally {
    loading.value = false;
  }
}

async function openProject(p: Project) {
  selected.value = p;
  tab.value = "compose";
  output.value = "";

  const compose = await apiGet<FileResp>(`/api/production/projects/${encodeURIComponent(p.name)}/compose`);
  composeText.value = compose.text;

  envText.value = "";
  if (p.hasEnv) {
    try {
      const env = await apiGet<FileResp>(`/api/production/projects/${encodeURIComponent(p.name)}/env`);
      envText.value = env.text;
    } catch {
      envText.value = "(konnte .env nicht lesen)";
    }
  } else {
    envText.value = "(keine .env vorhanden)";
  }

  // keep URL in sync so Docker page can deep-link
  router.replace({ path: "/production", query: { project: p.name } });
}

function closeProject() {
  selected.value = null;
  router.replace({ path: "/production" });
}

async function act(action: "pull" | "up" | "down" | "restart") {
  if (!selected.value) return;

  // confirm for potentially disruptive actions
  if ((action === "down") && !confirm(`Wirklich "${selected.value.name}" DOWN ausführen?`)) return;
  if ((action === "restart") && !confirm(`Wirklich "${selected.value.name}" RESTART ausführen?`)) return;

  tab.value = "output";
  output.value = "Running…\n";

  try {
    const res = await apiPost<any>(`/api/production/projects/${encodeURIComponent(selected.value.name)}/${action}`);
    output.value = res?.output || "OK";
    await refresh();
  } catch (e: any) {
    output.value = `ERROR:\n${String(e?.message ?? e)}`;
  }
}

onMounted(refresh);
</script>

<template>
  <div>
    <div class="head">
      <h2>Production</h2>
      <button class="btn" @click="refresh" :disabled="loading">Refresh</button>
    </div>

    <div v-if="error" class="err">❌ {{ error }}</div>
    <div v-else-if="loading">Lade…</div>

    <div v-else class="grid">
      <div v-for="p in projects" :key="p.name" class="card">
        <div class="top">
          <div class="name">{{ p.name }}</div>
          <div class="stat">
            <span class="pill">{{ p.containerRun }}/{{ p.containerTotal }} running</span>
          </div>
        </div>

        <div class="meta mono">{{ p.path }}</div>

        <div class="actions">
          <button class="btn2" @click="openProject(p)">Open</button>
          <button class="btn"
            @click="apiPost(`/api/production/projects/${encodeURIComponent(p.name)}/pull`).then(refresh).catch(e => alert(e))">
            Pull Images
          </button>
        </div>
      </div>
    </div>

    <div v-if="selected" class="drawer">
      <div class="drawerHead">
        <div>
          <div class="drawerTitle">{{ selected.name }}</div>
          <div class="drawerSub mono">{{ selected.path }}</div>
        </div>
        <button class="btn" @click="closeProject">Close</button>
      </div>

      <div class="drawerActions">
        <button class="btn" @click="act('pull')">Pull</button>
        <button class="btn" @click="act('up')">Up -d</button>
        <button class="btn" @click="act('restart')">Restart</button>
        <button class="btn danger" @click="act('down')">Down</button>
      </div>

      <div class="tabs">
        <button class="tab" :class="{ active: tab === 'compose' }" @click="tab = 'compose'">docker-compose.yml</button>
        <button class="tab" :class="{ active: tab === 'env' }" @click="tab = 'env'">.env</button>
        <button class="tab" :class="{ active: tab === 'output' }" @click="tab = 'output'">Output</button>
      </div>

      <pre v-if="tab === 'compose'" class="box">{{ composeText }}</pre>
      <pre v-else-if="tab === 'env'" class="box">{{ envText }}</pre>
      <pre v-else class="box">{{ output }}</pre>
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

.btn.danger {
  border-color: #6a2a2a;
  background: #2a1414;
}

.err {
  background: #2a1414;
  border: 1px solid #6a2a2a;
  padding: 10px;
  border-radius: 10px;
  margin: 10px 0;
}

.grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 12px;
}

.card {
  border: 1px solid #1b2a3a;
  background: #0f1620;
  border-radius: 12px;
  padding: 12px;
}

.top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.name {
  font-weight: 800;
}

.pill {
  border: 1px solid #2a3b4f;
  border-radius: 999px;
  padding: 4px 8px;
  font-size: 12px;
  opacity: .9;
}

.meta {
  margin-top: 8px;
  opacity: .85;
  font-size: 12px;
  word-break: break-all;
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 12px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
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

.drawerActions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 10px;
}

.tabs {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  flex-wrap: wrap;
}

.tab {
  border: 1px solid #26435e;
  background: #0f1620;
  color: #bcd0ea;
  padding: 6px 10px;
  border-radius: 999px;
  font-size: 12px;
}

.tab.active {
  background: #101e2e;
  color: #fff;
}

.box {
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
