<template>
  <n-space vertical size="large">
    <div class="page-header">
      <div class="header-info">
        <h2 class="page-title">{{ t("distributedKeys.title") }}</h2>
        <div class="page-subtitle">{{ t("distributedKeys.subtitle") }}</div>
      </div>
      <n-space>
        <n-button secondary :loading="loading" @click="refresh">
          <template #icon>
            <n-icon :component="RefreshOutline" />
          </template>
          {{ t("distributedKeys.refresh") }}
        </n-button>
        <n-button type="primary" @click="openCreate">
          <template #icon>
            <n-icon :component="AddOutline" />
          </template>
          {{ t("distributedKeys.create") }}
        </n-button>
      </n-space>
    </div>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :columns="columns"
        :data="items"
        :loading="loading"
        :row-key="rowKey"
        :pagination="pagination"
        size="small"
      />
    </n-card>

    <n-modal
      v-model:show="showCreate"
      preset="card"
      :title="t('distributedKeys.createModal.title')"
      style="max-width: 640px"
      class="custom-modal"
    >
      <n-form :model="createForm" label-placement="top">
        <n-form-item :label="t('distributedKeys.form.name')">
          <n-input v-model:value="createForm.name" :maxlength="64" />
        </n-form-item>
        <n-form-item :label="t('distributedKeys.form.note')">
          <n-input
            v-model:value="createForm.note"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 5 }"
          />
        </n-form-item>
        <n-grid :cols="2" :x-gap="12">
          <n-form-item-gi :label="t('distributedKeys.form.rateLimit')">
            <n-input-number
              v-model:value="createForm.rate_limit_per_minute"
              :min="0"
              style="width: 100%"
            >
              <template #suffix>{{ t("distributedKeys.units.perMinute") }}</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :label="t('distributedKeys.form.expiresAt')">
            <n-date-picker
              v-model:value="createForm.expires_at"
              type="datetime"
              clearable
              style="width: 100%"
            />
          </n-form-item-gi>
        </n-grid>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreate = false">{{ t("common.cancel") }}</n-button>
          <n-button type="primary" :loading="saving" @click="createKey">
            {{ t("distributedKeys.createModal.submit") }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal
      v-model:show="showEdit"
      preset="card"
      :title="t('distributedKeys.editModal.title')"
      style="max-width: 640px"
      class="custom-modal"
    >
      <n-form :model="editForm" label-placement="top">
        <n-form-item :label="t('distributedKeys.form.name')">
          <n-input v-model:value="editForm.name" :maxlength="64" />
        </n-form-item>
        <n-form-item :label="t('distributedKeys.form.note')">
          <n-input
            v-model:value="editForm.note"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 5 }"
          />
        </n-form-item>
        <n-grid :cols="2" :x-gap="12">
          <n-form-item-gi :label="t('distributedKeys.form.rateLimit')">
            <n-input-number
              v-model:value="editForm.rate_limit_per_minute"
              :min="0"
              style="width: 100%"
            >
              <template #suffix>{{ t("distributedKeys.units.perMinute") }}</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :label="t('distributedKeys.form.expiresAt')">
            <n-date-picker
              v-model:value="editForm.expires_at"
              type="datetime"
              clearable
              style="width: 100%"
            />
          </n-form-item-gi>
        </n-grid>
        <n-space justify="space-between" align="center">
          <n-space align="center">
            <n-switch v-model:value="editForm.is_active" />
            <span>{{
              editForm.is_active ? t("common.active") : t("common.disabled")
            }}</span>
          </n-space>
          <n-checkbox v-model:checked="editForm.clear_expires_at">
            {{ t("distributedKeys.form.clearExpiresAt") }}
          </n-checkbox>
        </n-space>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEdit = false">{{ t("common.cancel") }}</n-button>
          <n-button type="primary" :loading="saving" @click="saveEdit">
            {{ t("distributedKeys.editModal.submit") }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal
      v-model:show="showPlainKey"
      preset="card"
      :title="t('distributedKeys.secretModal.title')"
      style="max-width: 640px"
      class="custom-modal"
    >
      <n-space vertical>
        <n-alert type="warning" :show-icon="true" size="small">
          {{ t("distributedKeys.secretModal.warning") }}
        </n-alert>
        <n-input-group>
          <n-input :value="latestPlainKey" readonly type="password" show-password-on="mousedown" />
          <n-button type="primary" ghost @click="copyPlainKey">
            <template #icon>
              <n-icon :component="CopyOutline" />
            </template>
          </n-button>
        </n-input-group>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button type="primary" @click="showPlainKey = false">
            {{ t("common.dismiss") }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <n-drawer v-model:show="showStats" :width="760">
      <n-drawer-content :title="statsTitle">
        <n-space vertical size="large">
          <n-space :size="[8, 8]">
            <n-tag type="info" round>{{ t("distributedKeys.stats.total") }}: {{ stats?.totals.total_count ?? 0 }}</n-tag>
            <n-tag type="success" round>2xx: {{ stats?.totals.status_2xx ?? 0 }}</n-tag>
            <n-tag type="warning" round>4xx: {{ stats?.totals.status_4xx ?? 0 }}</n-tag>
            <n-tag type="error" round>5xx: {{ stats?.totals.status_5xx ?? 0 }}</n-tag>
          </n-space>
          <n-space align="center">
            <span>{{ t("distributedKeys.stats.days") }}</span>
            <n-select
              v-model:value="statsDays"
              :options="statsDayOptions"
              size="small"
              style="width: 120px"
              @update:value="onStatsDaysChanged"
            />
          </n-space>
          <n-data-table
            :columns="statsColumns"
            :data="stats?.series ?? []"
            :loading="statsLoading"
            :row-key="statsRowKey"
            size="small"
            :pagination="{ pageSize: 10 }"
          />
        </n-space>
      </n-drawer-content>
    </n-drawer>
  </n-space>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from "vue";
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NDatePicker,
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NIcon,
  NInput,
  NInputGroup,
  NInputNumber,
  NModal,
  NPopconfirm,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns,
} from "naive-ui";
import {
  AddOutline,
  CopyOutline,
  CreateOutline,
  RefreshOutline,
  RepeatOutline,
  StatsChartOutline,
  TrashOutline,
} from "@vicons/ionicons5";

import { api } from "../api/client";
import { t } from "../i18n";
import type {
  DistributedKeyItem,
  DistributedKeyStats,
  DistributedKeyUsagePoint,
} from "../types";
import { writeClipboardText } from "../utils/clipboard";

const message = useMessage();

const items = ref<DistributedKeyItem[]>([]);
const loading = ref(false);
const saving = ref(false);

const showCreate = ref(false);
const showEdit = ref(false);
const editId = ref<number | null>(null);

const latestPlainKey = ref("");
const showPlainKey = ref(false);

const showStats = ref(false);
const statsLoading = ref(false);
const stats = ref<DistributedKeyStats | null>(null);
const statsKeyID = ref<number | null>(null);
const statsDays = ref<number>(30);

const statsDayOptions = [
  { label: "7", value: 7 },
  { label: "30", value: 30 },
  { label: "90", value: 90 },
];

const createForm = reactive<{
  name: string;
  note: string;
  rate_limit_per_minute: number;
  expires_at: number | null;
}>({
  name: "",
  note: "",
  rate_limit_per_minute: 60,
  expires_at: null,
});

const editForm = reactive<{
  name: string;
  note: string;
  is_active: boolean;
  rate_limit_per_minute: number;
  expires_at: number | null;
  clear_expires_at: boolean;
}>({
  name: "",
  note: "",
  is_active: true,
  rate_limit_per_minute: 60,
  expires_at: null,
  clear_expires_at: false,
});

const pagination = reactive({
  pageSize: 10,
});

const statsTitle = computed(() => {
  return stats.value?.item?.name
    ? `${t("distributedKeys.stats.title")} - ${stats.value.item.name}`
    : t("distributedKeys.stats.title");
});

function rowKey(row: DistributedKeyItem) {
  return row.id;
}

function statsRowKey(row: DistributedKeyUsagePoint) {
  return row.date;
}

function formatDateTime(input: string | null | undefined): string {
  if (!input) return "-";
  const date = new Date(input);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString();
}

function isExpired(expiresAt: string | null | undefined): boolean {
  if (!expiresAt) return false;
  const date = new Date(expiresAt);
  if (Number.isNaN(date.getTime())) return false;
  return date.getTime() <= Date.now();
}

function toRFC3339(value: number | null): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toISOString();
}

function openCreate() {
  createForm.name = "";
  createForm.note = "";
  createForm.rate_limit_per_minute = 60;
  createForm.expires_at = null;
  showCreate.value = true;
}

async function refresh() {
  loading.value = true;
  try {
    const { data } = await api.get<{ items: DistributedKeyItem[] }>(
      "/api/distributed-keys"
    );
    items.value = data.items;
  } catch (err: any) {
    message.error(err?.response?.data?.error ?? t("distributedKeys.errors.load"));
  } finally {
    loading.value = false;
  }
}

async function createKey() {
  saving.value = true;
  try {
    const payload: Record<string, unknown> = {
      name: createForm.name,
      note: createForm.note,
      rate_limit_per_minute: Math.max(0, Math.floor(createForm.rate_limit_per_minute || 0)),
    };
    const expiresAt = toRFC3339(createForm.expires_at);
    if (expiresAt) payload.expires_at = expiresAt;

    const { data } = await api.post<{ plain_key: string }>(
      "/api/distributed-keys",
      payload
    );
    showCreate.value = false;
    await refresh();

    latestPlainKey.value = data.plain_key;
    showPlainKey.value = true;
    message.success(t("distributedKeys.messages.created"));
  } catch (err: any) {
    message.error(err?.response?.data?.error ?? t("distributedKeys.errors.create"));
  } finally {
    saving.value = false;
  }
}

function openEdit(row: DistributedKeyItem) {
  editId.value = row.id;
  editForm.name = row.name;
  editForm.note = row.note ?? "";
  editForm.is_active = row.is_active;
  editForm.rate_limit_per_minute = row.rate_limit_per_minute ?? 0;
  editForm.expires_at = row.expires_at ? new Date(row.expires_at).getTime() : null;
  editForm.clear_expires_at = false;
  showEdit.value = true;
}

async function saveEdit() {
  if (editId.value == null) return;
  saving.value = true;
  try {
    const payload: Record<string, unknown> = {
      name: editForm.name,
      note: editForm.note,
      is_active: editForm.is_active,
      rate_limit_per_minute: Math.max(0, Math.floor(editForm.rate_limit_per_minute || 0)),
      clear_expires_at: editForm.clear_expires_at,
    };
    if (!editForm.clear_expires_at) {
      const expiresAt = toRFC3339(editForm.expires_at);
      if (expiresAt) payload.expires_at = expiresAt;
    }

    await api.put(`/api/distributed-keys/${editId.value}`, payload);
    showEdit.value = false;
    await refresh();
    message.success(t("distributedKeys.messages.updated"));
  } catch (err: any) {
    message.error(err?.response?.data?.error ?? t("distributedKeys.errors.update"));
  } finally {
    saving.value = false;
  }
}

async function rotateKey(row: DistributedKeyItem) {
  try {
    const { data } = await api.post<{ plain_key: string }>(
      `/api/distributed-keys/${row.id}/rotate`
    );
    latestPlainKey.value = data.plain_key;
    showPlainKey.value = true;
    await refresh();
    message.success(t("distributedKeys.messages.rotated"));
  } catch (err: any) {
    message.error(err?.response?.data?.error ?? t("distributedKeys.errors.rotate"));
  }
}

async function deleteKey(row: DistributedKeyItem) {
  try {
    await api.delete(`/api/distributed-keys/${row.id}`);
    await refresh();
    message.success(t("distributedKeys.messages.deleted"));
  } catch (err: any) {
    message.error(err?.response?.data?.error ?? t("distributedKeys.errors.delete"));
  }
}

async function openStats(row: DistributedKeyItem) {
  statsKeyID.value = row.id;
  showStats.value = true;
  await fetchStats();
}

async function onStatsDaysChanged() {
  if (!showStats.value || statsKeyID.value == null) return;
  await fetchStats();
}

async function fetchStats() {
  if (statsKeyID.value == null) return;
  statsLoading.value = true;
  try {
    const { data } = await api.get<DistributedKeyStats>(
      `/api/distributed-keys/${statsKeyID.value}/stats`,
      {
        params: { days: statsDays.value },
      }
    );
    stats.value = data;
  } catch (err: any) {
    message.error(err?.response?.data?.error ?? t("distributedKeys.errors.stats"));
  } finally {
    statsLoading.value = false;
  }
}

async function copyPlainKey() {
  if (!latestPlainKey.value) return;
  try {
    await writeClipboardText(latestPlainKey.value);
    message.success(t("common.copiedToClipboard"));
  } catch {
    message.error(t("common.copyFailed"));
  }
}

const columns: DataTableColumns<DistributedKeyItem> = [
  {
    title: () => t("distributedKeys.table.name"),
    key: "name",
    width: 180,
    render: (row) => h("div", { class: "name-cell" }, row.name),
  },
  {
    title: () => t("distributedKeys.table.keyPrefix"),
    key: "key_prefix",
    width: 160,
    render: (row) => h("code", { class: "key-prefix" }, `${row.key_prefix}...`),
  },
  {
    title: () => t("distributedKeys.table.rateLimit"),
    key: "rate_limit_per_minute",
    width: 130,
    align: "center",
    render: (row) =>
      row.rate_limit_per_minute === 0
        ? t("distributedKeys.unlimited")
        : `${row.rate_limit_per_minute}/m`,
  },
  {
    title: () => t("distributedKeys.table.status"),
    key: "is_active",
    width: 220,
    render: (row) =>
      h(
        NSpace,
        { size: 6 },
        {
          default: () => [
            h(
              NTag,
              { type: row.is_active ? "success" : "default", size: "small" },
              { default: () => (row.is_active ? t("common.active") : t("common.disabled")) }
            ),
            row.expires_at
              ? h(
                  NTag,
                  { type: isExpired(row.expires_at) ? "error" : "warning", size: "small" },
                  {
                    default: () =>
                      isExpired(row.expires_at)
                        ? t("distributedKeys.expired")
                        : `${t("distributedKeys.expiresAt")}: ${formatDateTime(row.expires_at)}`,
                  }
                )
              : h(
                  NTag,
                  { type: "info", size: "small" },
                  { default: () => t("distributedKeys.neverExpires") }
                ),
          ],
        }
      ),
  },
  {
    title: () => t("distributedKeys.table.usage"),
    key: "usage",
    render: (row) =>
      h(
        NSpace,
        { size: 6, align: "center" },
        {
          default: () => [
            h(NTag, { type: "info", size: "small" }, { default: () => `${t("distributedKeys.stats.total")}: ${row.total_count}` }),
            h(NTag, { type: "success", size: "small" }, { default: () => `2xx ${row.status_2xx}` }),
            h(NTag, { type: "warning", size: "small" }, { default: () => `4xx ${row.status_4xx}` }),
            h(NTag, { type: "error", size: "small" }, { default: () => `5xx ${row.status_5xx}` }),
          ],
        }
      ),
  },
  {
    title: () => t("distributedKeys.table.actions"),
    key: "actions",
    width: 210,
    align: "right",
    render: (row) =>
      h(
        NSpace,
        { size: "small", justify: "end" },
        {
          default: () => [
            h(
              NButton,
              {
                size: "small",
                quaternary: true,
                circle: true,
                onClick: () => openStats(row),
              },
              { icon: () => h(NIcon, { component: StatsChartOutline }) }
            ),
            h(
              NButton,
              {
                size: "small",
                quaternary: true,
                circle: true,
                onClick: () => openEdit(row),
              },
              { icon: () => h(NIcon, { component: CreateOutline }) }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => rotateKey(row) },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      quaternary: true,
                      circle: true,
                    },
                    { icon: () => h(NIcon, { component: RepeatOutline }) }
                  ),
                default: () => t("distributedKeys.confirm.rotate"),
              }
            ),
            h(
              NPopconfirm,
              { onPositiveClick: () => deleteKey(row) },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      quaternary: true,
                      circle: true,
                      type: "error",
                    },
                    { icon: () => h(NIcon, { component: TrashOutline }) }
                  ),
                default: () => t("distributedKeys.confirm.delete"),
              }
            ),
          ],
        }
      ),
  },
];

const statsColumns: DataTableColumns<DistributedKeyUsagePoint> = [
  { title: "Date", key: "date", width: 140 },
  { title: "Total", key: "total_count", width: 90 },
  { title: "2xx", key: "status_2xx", width: 80 },
  { title: "4xx", key: "status_4xx", width: 80 },
  { title: "5xx", key: "status_5xx", width: 80 },
];

onMounted(async () => {
  await refresh();
});
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.page-subtitle {
  color: #888;
  font-size: 13px;
}

.table-card {
  border-radius: 12px;
}

.table-card :deep(.n-card__content) {
  padding: 0;
}

.name-cell {
  font-weight: 600;
}

.key-prefix {
  background: rgba(0, 0, 0, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
}

.custom-modal {
  border-radius: 16px;
}

:deep(.n-data-table-td) {
  padding: 12px 16px;
}
</style>
