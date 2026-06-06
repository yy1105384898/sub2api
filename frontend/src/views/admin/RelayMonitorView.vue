<template>
  <AppLayout>
    <div class="space-y-5">
      <!-- 顶部统计卡 -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div class="rounded-xl border border-red-100 bg-red-50 p-5 dark:border-red-900/40 dark:bg-red-950/30">
          <p class="text-sm font-medium text-red-600 dark:text-red-400">{{ t('admin.relayMonitor.upAnnouncements') }}</p>
          <p class="mt-1 text-3xl font-bold text-red-600 dark:text-red-400">{{ summary.up_count }}</p>
        </div>
        <div class="rounded-xl border border-green-100 bg-green-50 p-5 dark:border-green-900/40 dark:bg-green-950/30">
          <p class="text-sm font-medium text-green-600 dark:text-green-400">{{ t('admin.relayMonitor.downAnnouncements') }}</p>
          <p class="mt-1 text-3xl font-bold text-green-600 dark:text-green-400">{{ summary.down_count }}</p>
        </div>
        <div class="rounded-xl border border-blue-100 bg-blue-50 p-5 dark:border-blue-900/40 dark:bg-blue-950/30">
          <p class="text-sm font-medium text-blue-600 dark:text-blue-400">{{ t('admin.relayMonitor.lastRefresh') }}</p>
          <p class="mt-1 text-lg font-semibold text-blue-700 dark:text-blue-300">{{ lastRefreshLabel }}</p>
        </div>
      </div>

      <!-- Tab 切换 -->
      <div class="flex gap-2 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="px-4 py-2 text-sm font-medium -mb-px border-b-2 transition-colors"
          :class="activeTab === tab.key
            ? 'border-primary-500 text-primary-600 dark:text-primary-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- ============ 公告看板 ============ -->
      <div v-show="activeTab === 'changes'" class="space-y-4">
        <!-- 筛选 -->
        <div class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.relayMonitor.filterTitle') }}</h2>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
            <span class="text-red-500">{{ t('admin.relayMonitor.legendUp') }}</span>
            {{ t('admin.relayMonitor.legendSep') }}
            <span class="text-green-500">{{ t('admin.relayMonitor.legendDown') }}</span>
          </p>
          <div class="mt-3 flex flex-col gap-3 md:flex-row md:items-center">
            <select
              v-model="directionFilter"
              class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white md:w-48"
              @change="reloadChanges"
            >
              <option value="">{{ t('admin.relayMonitor.allAnnouncements') }}</option>
              <option value="up">{{ t('admin.relayMonitor.onlyUp') }}</option>
              <option value="down">{{ t('admin.relayMonitor.onlyDown') }}</option>
            </select>
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.relayMonitor.searchPlaceholder')"
              class="flex-1 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
              @keyup.enter="reloadChanges"
            />
            <button
              class="inline-flex items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-60"
              :disabled="probingAll"
              @click="probeAll"
            >
              {{ probingAll ? t('admin.relayMonitor.probing') : t('admin.relayMonitor.probeAll') }}
            </button>
          </div>
        </div>

        <!-- 历史表格 -->
        <div class="overflow-x-auto rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-left text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
                <th class="px-4 py-3 font-medium">#</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colType') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colSite') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colSystem') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colVendor') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colGroup') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colOldRate') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colNewRate') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colChange') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colTime') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colContent') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, idx) in changes"
                :key="row.id"
                class="border-b border-gray-100 last:border-0 dark:border-dark-700/60"
                :class="row.direction === 'up' ? 'bg-red-50/40 dark:bg-red-950/10' : 'bg-green-50/40 dark:bg-green-950/10'"
              >
                <td class="px-4 py-3 text-gray-500">{{ (changesPage - 1) * changesPageSize + idx + 1 }}</td>
                <td class="px-4 py-3">
                  <span
                    class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
                    :class="row.direction === 'up'
                      ? 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
                      : 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'"
                  >
                    {{ row.direction === 'up' ? t('admin.relayMonitor.up') : t('admin.relayMonitor.down') }}
                  </span>
                </td>
                <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{{ row.site }}</td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center rounded bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300">{{ row.system }}</span>
                </td>
                <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ row.vendor || '-' }}</td>
                <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ row.group_name }}</td>
                <td class="px-4 py-3 text-gray-500">{{ formatRate(row.old_rate) }}</td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-200">{{ formatRate(row.new_rate) }}</td>
                <td class="px-4 py-3 font-medium" :class="row.direction === 'up' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'">
                  {{ (row.direction === 'up' ? t('admin.relayMonitor.up') : t('admin.relayMonitor.down')) + ' ' + formatRate(Math.abs(row.new_rate - row.old_rate)) }}
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-gray-500">{{ formatTime(row.detected_at) }}</td>
                <td class="px-4 py-3 max-w-xs truncate text-gray-500" :title="row.content">{{ row.content }}</td>
              </tr>
              <tr v-if="!changesLoading && changes.length === 0">
                <td colspan="11" class="px-4 py-10 text-center text-gray-400">{{ t('admin.relayMonitor.noChanges') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination
          v-if="changesTotal > 0"
          :page="changesPage"
          :total="changesTotal"
          :page-size="changesPageSize"
          @update:page="onChangesPage"
          @update:page-size="onChangesPageSize"
        />
      </div>

      <!-- ============ 站点管理 ============ -->
      <div v-show="activeTab === 'sites'" class="space-y-4">
        <div class="flex justify-end">
          <button
            class="inline-flex items-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700"
            @click="openCreate"
          >
            {{ t('admin.relayMonitor.addSite') }}
          </button>
        </div>

        <div class="overflow-x-auto rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-left text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colName') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colSystem') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colBaseUrl') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colWatched') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colInterval') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colEnabled') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colLastChecked') }}</th>
                <th class="px-4 py-3 font-medium text-right">{{ t('admin.relayMonitor.colActions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in monitors" :key="m.id" class="border-b border-gray-100 last:border-0 dark:border-dark-700/60">
                <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">
                  {{ m.name }}
                  <span v-if="m.credential_decrypt_failed" class="ml-1 text-xs text-red-500">⚠</span>
                  <p v-if="m.last_error" class="text-xs text-red-500 truncate max-w-[200px]" :title="m.last_error">{{ m.last_error }}</p>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center rounded bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300">{{ m.system }}</span>
                </td>
                <td class="px-4 py-3 max-w-xs truncate text-gray-500" :title="m.base_url">{{ m.base_url }}</td>
                <td class="px-4 py-3 text-gray-600 dark:text-gray-300">
                  {{ m.watched_groups.length ? m.watched_groups.join(', ') : t('admin.relayMonitor.noGroupSelected') }}
                </td>
                <td class="px-4 py-3 text-gray-500">{{ m.interval_seconds }}s</td>
                <td class="px-4 py-3">
                  <span class="inline-flex h-2 w-2 rounded-full" :class="m.enabled ? 'bg-green-500' : 'bg-gray-300'"></span>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-gray-500">{{ m.last_checked_at ? formatTime(m.last_checked_at) : '-' }}</td>
                <td class="px-4 py-3 text-right space-x-2 whitespace-nowrap">
                  <button class="text-primary-600 hover:underline disabled:opacity-50" :disabled="probingId === m.id" @click="probeOne(m)">
                    {{ probingId === m.id ? t('admin.relayMonitor.probing') : t('admin.relayMonitor.probe') }}
                  </button>
                  <button class="text-gray-600 hover:underline dark:text-gray-300" @click="openEdit(m)">{{ t('common.edit') }}</button>
                  <button class="text-red-600 hover:underline" @click="confirmDelete(m)">{{ t('common.delete') }}</button>
                </td>
              </tr>
              <tr v-if="!loading && monitors.length === 0">
                <td colspan="8" class="px-4 py-10 text-center text-gray-400">{{ t('admin.relayMonitor.noSites') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 站点表单弹窗 -->
    <BaseDialog :show="showForm" :title="editing ? t('admin.relayMonitor.editSite') : t('admin.relayMonitor.addSite')" width="wide" @close="showForm = false">
      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.colName') }}</label>
            <input v-model="form.name" type="text" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.colSystem') }}</label>
            <select v-model="form.system" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white">
              <option value="sub2api">sub2api</option>
              <option value="newapi">newapi</option>
            </select>
          </div>
          <div class="md:col-span-2">
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.colBaseUrl') }}</label>
            <input v-model="form.base_url" type="text" placeholder="https://example.com" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.colVendor') }}</label>
            <input v-model="form.vendor" type="text" placeholder="OpenAI" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.colInterval') }} (s)</label>
            <input v-model.number="form.interval_seconds" type="number" min="60" max="86400" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </div>
          <div class="md:col-span-2">
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.relayMonitor.credential') }}
              <span class="text-xs font-normal text-gray-400">{{ form.system === 'sub2api' ? t('admin.relayMonitor.credentialRequired') : t('admin.relayMonitor.credentialOptional') }}</span>
            </label>
            <input v-model="form.credential" type="password" autocomplete="off" :placeholder="editing ? t('admin.relayMonitor.credentialKeep') : 'Bearer token'" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </div>
        </div>

        <!-- 分组选择 -->
        <div>
          <div class="mb-1 flex items-center justify-between">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.watchedGroups') }}</label>
            <button class="text-xs text-primary-600 hover:underline disabled:opacity-50" :disabled="fetchingGroups" @click="fetchGroups">
              {{ fetchingGroups ? t('admin.relayMonitor.fetching') : t('admin.relayMonitor.fetchGroups') }}
            </button>
          </div>
          <p class="mb-2 text-xs text-gray-400">{{ t('admin.relayMonitor.watchedGroupsHint') }}</p>
          <div v-if="availableGroups.length" class="max-h-48 space-y-1 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600">
            <label v-for="g in availableGroups" :key="g.group_name" class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-gray-50 dark:hover:bg-dark-700">
              <input type="checkbox" :value="g.group_name" v-model="form.watched_groups" class="rounded" />
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ g.group_name }}</span>
              <span class="ml-auto text-xs text-gray-400">{{ formatRate(g.rate) }}</span>
            </label>
          </div>
          <div v-else-if="form.watched_groups.length" class="flex flex-wrap gap-1">
            <span v-for="g in form.watched_groups" :key="g" class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ g }}</span>
          </div>
          <p v-else class="text-xs text-gray-400">{{ t('admin.relayMonitor.noGroupsFetched') }}</p>
        </div>

        <label class="flex items-center gap-2">
          <input type="checkbox" v-model="form.enabled" class="rounded" />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.enableProbing') }}</span>
        </label>
      </div>

      <template #footer>
        <button class="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-dark-600 dark:text-gray-200" @click="showForm = false">{{ t('common.cancel') }}</button>
        <button class="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-60" :disabled="saving" @click="save">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!deleting"
      :title="t('admin.relayMonitor.deleteTitle')"
      :message="deleting ? t('admin.relayMonitor.deleteConfirm', { name: deleting.name }) : ''"
      :confirm-text="t('common.delete')"
      @confirm="doDelete"
      @cancel="deleting = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import { relayMonitorAPI } from '@/api/admin/relayMonitor'
import type {
  RelayMonitor,
  RelayRateChange,
  RelayGroupRate,
  RelaySystem,
  RateDirection,
} from '@/api/admin/relayMonitor'

const { t } = useI18n()
const appStore = useAppStore()

const tabs = computed(() => [
  { key: 'changes' as const, label: t('admin.relayMonitor.tabChanges') },
  { key: 'sites' as const, label: t('admin.relayMonitor.tabSites') },
])
const activeTab = ref<'changes' | 'sites'>('changes')

// ---- 汇总 ----
const summary = ref({ up_count: 0, down_count: 0 })
const lastRefresh = ref<string>('')
const lastRefreshLabel = computed(() => (lastRefresh.value ? formatTime(lastRefresh.value) : '-'))

// ---- 公告历史 ----
const changes = ref<RelayRateChange[]>([])
const changesTotal = ref(0)
const changesPage = ref(1)
const changesPageSize = ref(20)
const changesLoading = ref(false)
const directionFilter = ref<'' | RateDirection>('')
const searchQuery = ref('')
const probingAll = ref(false)

// ---- 站点 ----
const monitors = ref<RelayMonitor[]>([])
const loading = ref(false)
const probingId = ref<number | null>(null)

// ---- 表单 ----
const showForm = ref(false)
const editing = ref<RelayMonitor | null>(null)
const saving = ref(false)
const fetchingGroups = ref(false)
const availableGroups = ref<RelayGroupRate[]>([])
const deleting = ref<RelayMonitor | null>(null)

const form = reactive({
  name: '',
  system: 'sub2api' as RelaySystem,
  base_url: '',
  vendor: '',
  credential: '',
  watched_groups: [] as string[],
  interval_seconds: 300,
  enabled: true,
})

function formatRate(r: number): string {
  if (r === null || r === undefined || Number.isNaN(r)) return '-'
  return `${parseFloat(r.toFixed(6))}x`
}

function formatTime(s: string): string {
  if (!s) return '-'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

async function loadSummary() {
  try {
    summary.value = await relayMonitorAPI.summary(searchQuery.value.trim() || undefined)
  } catch {
    /* 汇总失败不阻断主流程 */
  }
}

async function reloadChanges() {
  changesPage.value = 1
  await loadChanges()
  await loadSummary()
}

async function loadChanges() {
  changesLoading.value = true
  try {
    const res = await relayMonitorAPI.listChanges({
      page: changesPage.value,
      page_size: changesPageSize.value,
      direction: directionFilter.value || undefined,
      search: searchQuery.value.trim() || undefined,
    })
    changes.value = res.items
    changesTotal.value = res.total
    if (res.items.length > 0) {
      lastRefresh.value = res.items[0].detected_at
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.loadError')))
  } finally {
    changesLoading.value = false
  }
}

function onChangesPage(p: number) {
  changesPage.value = p
  loadChanges()
}

function onChangesPageSize(size: number) {
  changesPageSize.value = size
  changesPage.value = 1
  loadChanges()
}

async function loadMonitors() {
  loading.value = true
  try {
    const res = await relayMonitorAPI.list({ page: 1, page_size: 100 })
    monitors.value = res.items
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.loadError')))
  } finally {
    loading.value = false
  }
}

async function probeAll() {
  probingAll.value = true
  try {
    const res = await relayMonitorAPI.probeAll()
    appStore.showSuccess(t('admin.relayMonitor.probeAllDone', { n: res.probed }))
    lastRefresh.value = new Date().toISOString()
    await loadChanges()
    await loadSummary()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.probeFailed')))
  } finally {
    probingAll.value = false
  }
}

async function probeOne(m: RelayMonitor) {
  probingId.value = m.id
  try {
    const res = await relayMonitorAPI.probe(m.id)
    appStore.showSuccess(t('admin.relayMonitor.probeOneDone', { n: res.changes.length }))
    await loadMonitors()
    if (activeTab.value === 'changes') await loadChanges()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.probeFailed')))
  } finally {
    probingId.value = null
  }
}

function resetForm() {
  form.name = ''
  form.system = 'sub2api'
  form.base_url = ''
  form.vendor = ''
  form.credential = ''
  form.watched_groups = []
  form.interval_seconds = 300
  form.enabled = true
  availableGroups.value = []
}

function openCreate() {
  editing.value = null
  resetForm()
  showForm.value = true
}

function openEdit(m: RelayMonitor) {
  editing.value = m
  form.name = m.name
  form.system = m.system
  form.base_url = m.base_url
  form.vendor = m.vendor
  form.credential = ''
  form.watched_groups = [...m.watched_groups]
  form.interval_seconds = m.interval_seconds
  form.enabled = m.enabled
  availableGroups.value = m.watched_groups.map((g) => ({ group_name: g, rate: NaN }))
  showForm.value = true
}

async function fetchGroups() {
  if (!form.base_url.trim()) {
    appStore.showError(t('admin.relayMonitor.baseUrlRequired'))
    return
  }
  fetchingGroups.value = true
  try {
    availableGroups.value = await relayMonitorAPI.fetchGroups({
      system: form.system,
      base_url: form.base_url.trim(),
      credential: form.credential.trim() || undefined,
      monitor_id: editing.value?.id,
    })
    if (availableGroups.value.length === 0) {
      appStore.showError(t('admin.relayMonitor.noGroupsFetched'))
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.fetchGroupsFailed')))
  } finally {
    fetchingGroups.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      system: form.system,
      base_url: form.base_url.trim(),
      vendor: form.vendor.trim(),
      watched_groups: form.watched_groups,
      interval_seconds: form.interval_seconds,
      enabled: form.enabled,
      ...(form.credential.trim() ? { credential: form.credential.trim() } : {}),
    }
    if (editing.value) {
      await relayMonitorAPI.update(editing.value.id, payload)
      appStore.showSuccess(t('common.updateSuccess'))
    } else {
      await relayMonitorAPI.create(payload)
      appStore.showSuccess(t('common.createSuccess'))
    }
    showForm.value = false
    await loadMonitors()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

function confirmDelete(m: RelayMonitor) {
  deleting.value = m
}

async function doDelete() {
  if (!deleting.value) return
  try {
    await relayMonitorAPI.del(deleting.value.id)
    appStore.showSuccess(t('common.deleteSuccess'))
    deleting.value = null
    await loadMonitors()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

onMounted(() => {
  loadChanges()
  loadSummary()
  loadMonitors()
})
</script>
