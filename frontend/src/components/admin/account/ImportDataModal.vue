<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ filesLabel }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.accounts.dataImportMultiHint') }}</div>
          </div>
          <div class="flex shrink-0 gap-2">
            <button type="button" class="btn btn-secondary" @click="openFilePicker">
              {{ t('admin.accounts.dataImportChooseFiles') }}
            </button>
            <button type="button" class="btn btn-secondary" @click="openFolderPicker">
              {{ t('admin.accounts.dataImportChooseFolder') }}
            </button>
          </div>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json"
          multiple
          @change="handleFileChange"
        />
        <input
          ref="folderInput"
          type="file"
          class="hidden"
          multiple
          @change="handleFolderChange"
        />
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminDataImportResult, AdminDataPayload } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const files = ref<File[]>([])
const result = ref<AdminDataImportResult | null>(null)

const fileInput = ref<HTMLInputElement | null>(null)
const folderInput = ref<HTMLInputElement | null>(null)
const filesLabel = computed(() => {
  if (!files.value.length) return t('admin.accounts.dataImportSelectFile')
  if (files.value.length === 1) return files.value[0].name
  return t('admin.accounts.dataImportFileCount', { count: files.value.length })
})

const errorItems = computed(() => result.value?.errors || [])

const onlyJSON = (list: FileList | null): File[] =>
  Array.from(list || []).filter((f) => f.name.toLowerCase().endsWith('.json'))

watch(
  () => props.show,
  (open) => {
    if (open) {
      files.value = []
      result.value = null
      if (fileInput.value) fileInput.value.value = ''
      if (folderInput.value) folderInput.value.value = ''
    }
  }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const openFolderPicker = () => {
  if (folderInput.value) {
    // webkitdirectory 不是标准 TS 属性，运行时设置以让用户选整个文件夹。
    folderInput.value.setAttribute('webkitdirectory', '')
    folderInput.value.click()
  }
}

const handleFileChange = (event: Event) => {
  files.value = onlyJSON((event.target as HTMLInputElement).files)
}

const handleFolderChange = (event: Event) => {
  files.value = onlyJSON((event.target as HTMLInputElement).files)
}

// 把任意形态的 JSON 归一化为导入负载：整导出对象 / 账号数组 / 单账号对象。
const normalizeToPayload = (parsed: any): AdminDataPayload => {
  if (parsed && typeof parsed === 'object' && Array.isArray(parsed.accounts)) {
    return parsed as AdminDataPayload
  }
  const accounts = Array.isArray(parsed) ? parsed : [parsed]
  return { exported_at: new Date().toISOString(), proxies: [], accounts } as AdminDataPayload
}

const handleClose = () => {
  if (importing.value) return
  emit('close')
}

const readFileAsText = async (sourceFile: File): Promise<string> => {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }

  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }

  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

const handleImport = async () => {
  if (!files.value.length) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  const agg: AdminDataImportResult = {
    proxy_created: 0, proxy_reused: 0, proxy_failed: 0,
    account_created: 0, account_failed: 0, errors: [],
  }
  try {
    for (const f of files.value) {
      try {
        const parsed = JSON.parse(await readFileAsText(f))
        const res = await adminAPI.accounts.importData({
          data: normalizeToPayload(parsed),
          skip_default_group_bind: true,
        })
        agg.proxy_created += res.proxy_created
        agg.proxy_reused += res.proxy_reused
        agg.proxy_failed += res.proxy_failed
        agg.account_created += res.account_created
        agg.account_failed += res.account_failed
        for (const e of res.errors || []) {
          agg.errors!.push({ ...e, name: `${f.name} · ${e.name || e.proxy_key || ''}` })
        }
      } catch (err: any) {
        // 单个文件解析/导入失败不影响其余文件。
        agg.account_failed += 1
        agg.errors!.push({
          kind: 'account',
          name: f.name,
          message: err instanceof SyntaxError ? t('admin.accounts.dataImportParseFailed') : (err?.message || t('admin.accounts.dataImportFailed')),
        })
      }
    }

    result.value = agg
    const msgParams = {
      account_created: agg.account_created, account_failed: agg.account_failed,
      proxy_created: agg.proxy_created, proxy_reused: agg.proxy_reused, proxy_failed: agg.proxy_failed,
    }
    if (agg.account_failed > 0 || agg.proxy_failed > 0) {
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.dataImportSuccess', msgParams))
    }
    if (agg.account_created > 0 || agg.proxy_created > 0) {
      emit('imported')
    }
  } finally {
    importing.value = false
  }
}
</script>
