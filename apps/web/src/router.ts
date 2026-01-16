import { createRouter, createWebHistory } from "vue-router";
import type { RouteRecordRaw } from "vue-router";

import Dashboard from "./views/Dashboard.vue";
import Tasks from "./views/Tasks.vue";
import Docker from "./views/Docker.vue";

import Production from "./views/Production.vue";
import Rclone from "./views/Rclone.vue";
import StorageArr from "./views/StorageArr.vue";
import StorageVfs from "./views/StorageVfs.vue";
import Journal from "./views/Journal.vue";
import Firewall from "./views/Firewall.vue";

const routes: RouteRecordRaw[] = [
  { path: "/", component: Dashboard },
  { path: "/tasks", component: Tasks },
  { path: "/docker", component: Docker },
  { path: "/production", component: Production },
  { path: "/rclone", component: Rclone },
  { path: "/storage/arr", component: StorageArr },
  { path: "/storage/vfs", component: StorageVfs },
  { path: "/logs/journal", component: Journal },
  { path: "/firewall", component: Firewall },
];

export default createRouter({
  history: createWebHistory(),
  routes,
});

