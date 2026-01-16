<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

const nav = [
  { to: "/", label: "Dashboard" },
  { to: "/tasks", label: "Tasks" },
  { to: "/docker", label: "Docker" },
  { to: "/production", label: "Production" },
  { to: "/rclone", label: "Rclone" },
  { to: "/storage/arr", label: "Arr" },
  { to: "/storage/vfs", label: "VFS" },
  { to: "/logs/journal", label: "Journal" },
  { to: "/firewall", label: "Firewall" },
];

const title = computed(() => nav.find(n => n.to === route.path)?.label ?? "Admin");
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">VPS Admin</div>
      <div class="title">{{ title }}</div>
    </header>

    <main class="content">
      <router-view />
    </main>

    <nav class="bottomnav">
      <router-link v-for="n in nav" :key="n.to" :to="n.to" class="navitem">
        {{ n.label }}
      </router-link>
    </nav>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  background: #0b0f14;
  color: #e8eef6;
  display: grid;
  grid-template-rows: auto 1fr auto;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  padding: 12px 14px;
  background: #0f1620;
  border-bottom: 1px solid #1b2a3a;
  display: flex;
  gap: 12px;
  align-items: baseline;
}

.brand {
  font-weight: 700;
  letter-spacing: 0.5px;
}

.title {
  opacity: 0.85;
}

.content {
  padding: 14px;
  padding-bottom: 76px;
  /* space for bottom nav */
}

.bottomnav {
  position: sticky;
  bottom: 0;
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  background: #0f1620;
  border-top: 1px solid #1b2a3a;
}

.navitem {
  padding: 12px 8px;
  text-align: center;
  text-decoration: none;
  color: #bcd0ea;
  font-size: 13px;
  border-right: 1px solid #1b2a3a;
}

.navitem:last-child {
  border-right: none;
}

.navitem.router-link-active {
  color: #ffffff;
  background: #101e2e;
}
</style>
