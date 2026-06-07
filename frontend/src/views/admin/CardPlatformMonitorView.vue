<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="grid grid-cols-2 gap-3 md:grid-cols-5">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">平台</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ summaryData.platform_count }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">商品</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ summaryData.product_count }}</p>
        </div>
        <div class="rounded-lg border border-emerald-100 bg-emerald-50 p-4 dark:border-emerald-900/40 dark:bg-emerald-950/30">
          <p class="text-xs text-emerald-600 dark:text-emerald-300">降价/新低</p>
          <p class="mt-1 text-2xl font-bold text-emerald-700 dark:text-emerald-300">{{ summaryData.price_down }}</p>
        </div>
        <div class="rounded-lg border border-cyan-100 bg-cyan-50 p-4 dark:border-cyan-900/40 dark:bg-cyan-950/30">
          <p class="text-xs text-cyan-600 dark:text-cyan-300">补货</p>
          <p class="mt-1 text-2xl font-bold text-cyan-700 dark:text-cyan-300">{{ summaryData.restock }}</p>
        </div>
        <div class="rounded-lg border border-red-100 bg-red-50 p-4 dark:border-red-900/40 dark:bg-red-950/30">
          <p class="text-xs text-red-600 dark:text-red-300">异常平台</p>
          <p class="mt-1 text-2xl font-bold text-red-700 dark:text-red-300">{{ summaryData.error_count }}</p>
        </div>
      </div>

      <div class="flex gap-2 border-b border-gray-200 dark:border-dark-700">
        <button v-for="tab in tabs" :key="tab.key" class="-mb-px border-b-2 px-4 py-2 text-sm font-medium" :class="activeTab === tab.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'" @click="activeTab = tab.key">
          {{ tab.label }}
        </button>
      </div>

      <section v-if="activeTab === 'search'" class="space-y-4">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="grid gap-3 md:grid-cols-6">
            <input v-model="productQuery.search" class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white md:col-span-2" placeholder="搜索全平台商品，如 gpt / chatgpt / plus" @keyup.enter="reloadProducts" />
            <select v-model="productQuery.monitor_id" class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white">
              <option :value="0">全部平台</option>
              <option v-for="m in monitors" :key="m.id" :value="m.id">{{ m.name }}</option>
            </select>
            <select v-model="productQuery.sort" class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white">
              <option value="updated">最近更新</option>
              <option value="priceAsc">价格最低</option>
              <option value="priceDesc">价格最高</option>
              <option value="stockDesc">库存最多</option>
              <option value="newest">最新发现</option>
            </select>
            <label class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700 dark:text-gray-200">
              <input v-model="productQuery.in_stock" type="checkbox" />
              只看有库存
            </label>
            <button class="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700" @click="reloadProducts">搜索</button>
          </div>
        </div>

        <div v-if="productsLoading" class="rounded-lg border border-gray-200 bg-white p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800">加载中...</div>
        <div v-else-if="groupedProducts.length" class="space-y-4">
          <div v-for="(group, groupIndex) in groupedProducts" :key="group.name" class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center justify-between px-4 py-3" :class="siteToneClass(groupIndex)">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ group.name }}</h3>
                <p class="text-xs text-gray-500 dark:text-gray-300">{{ group.items.length }} 个商品 / {{ group.merchants.length }} 个商家</p>
              </div>
              <span class="rounded-full bg-white/70 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700/70 dark:text-gray-200">链动小铺</span>
            </div>

            <div v-for="(merchant, merchantIndex) in group.merchants" :key="`${group.name}-${merchant.name}`" class="border-t border-gray-200 dark:border-dark-700">
              <div class="flex items-center justify-between px-4 py-2 text-xs" :class="merchantToneClass(merchantIndex)">
                <div class="min-w-0 font-medium text-gray-700 dark:text-gray-200">
                  商家：<span class="font-semibold">{{ merchant.name }}</span>
                </div>
                <div class="text-gray-500 dark:text-gray-400">{{ merchant.items.length }} 个商品</div>
              </div>
              <div class="overflow-x-auto">
                <table class="monitor-table min-w-[980px] w-full text-left text-sm">
                  <thead>
                    <tr>
                      <th class="w-12 px-4 py-3">#</th>
                      <th class="px-4 py-3">商品名称</th>
                      <th class="w-24 px-4 py-3 text-right">价格</th>
                      <th class="w-24 px-4 py-3 text-right">优惠价</th>
                      <th class="w-24 px-4 py-3 text-right">库存</th>
                      <th class="w-28 px-4 py-3">商品编号</th>
                      <th class="w-28 px-4 py-3">更新时间</th>
                      <th class="w-32 px-4 py-3">商家</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(p, rowIndex) in merchant.items" :key="p.id">
                      <td class="px-4 py-3 font-medium text-gray-500 dark:text-gray-400">{{ rowIndex + 1 }}</td>
                      <td class="px-4 py-3">
                        <div class="min-w-0">
                          <a v-if="p.product_url" :href="p.product_url" target="_blank" rel="noopener noreferrer" class="line-clamp-2 font-semibold text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-300" :title="p.title">
                            {{ p.title || '-' }}
                          </a>
                          <div v-else class="line-clamp-2 font-semibold text-gray-900 dark:text-white" :title="p.title">{{ p.title || '-' }}</div>
                          <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                            <span>{{ p.category || '未分类' }}</span>
                            <span :class="statusClass(p.status)" class="rounded px-2 py-0.5">{{ statusLabel(p.status) }}</span>
                            <span v-if="p.last_event_type" :class="eventClass(p.last_event_type)" class="rounded px-2 py-0.5">{{ eventLabel(p.last_event_type) }}</span>
                          </div>
                        </div>
                      </td>
                      <td class="px-4 py-3 text-right font-mono text-base font-bold text-red-600 tabular-nums dark:text-red-400">¥{{ money(p.price) }}</td>
                      <td class="px-4 py-3 text-right font-mono tabular-nums text-gray-700 dark:text-gray-200">{{ promoPrice(p) }}</td>
                      <td class="px-4 py-3 text-right font-mono tabular-nums text-gray-800 dark:text-gray-100">{{ stockText(p.stock) }}</td>
                      <td class="px-4 py-3 font-mono text-gray-500 dark:text-gray-400" translate="no">{{ shortId(p.external_product_id) }}</td>
                      <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{{ relativeTime(p.updated_at || p.last_seen_at) }}</td>
                      <td class="px-4 py-3 font-medium text-gray-700 dark:text-gray-200">{{ p.merchant || '未知店铺' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="rounded-lg border border-gray-200 bg-white p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800">暂无商品，先添加平台并扫描。</div>
      </section>

      <section v-else-if="activeTab === 'platforms'" class="space-y-4">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ editingId ? '编辑平台' : '新增平台' }}</h3>
          <div class="mt-3 grid gap-3 md:grid-cols-6">
            <input v-model="form.name" class="input" placeholder="平台名称" />
            <select v-model="form.platform_type" class="input"><option value="ldxp">链动小铺（自动）</option></select>
            <select v-model="form.auth_mode" class="input">
              <option value="push">浏览器推送（推荐）</option>
              <option value="token">Token（直连，多被反爬挡）</option>
              <option value="cookie">Cookie（直连）</option>
            </select>
            <label class="flex items-center gap-2 text-sm dark:text-gray-200"><input v-model="form.enabled" type="checkbox" />启用</label>
            <input v-if="form.auth_mode !== 'push'" v-model="form.credential" class="input md:col-span-2" type="password" :placeholder="form.auth_mode === 'cookie' ? '完整 Cookie' : 'Merchant-Token'" />
            <input v-if="form.auth_mode !== 'push'" v-model.number="form.interval_seconds" class="input" type="number" min="60" placeholder="间隔秒" />
            <input v-if="form.auth_mode !== 'push'" v-model.number="form.fetch_pages" class="input" type="number" min="1" max="500" placeholder="扫描页数" />
            <input v-model="form.base_url" class="input md:col-span-3" placeholder="自定义接口地址（可选，不填自动识别链动小铺）" />
            <input v-model="form.note" class="input md:col-span-3" placeholder="备注" />
            <p v-if="form.auth_mode === 'push'" class="md:col-span-6 text-xs text-gray-500 dark:text-gray-400">推送模式：链动接口被阿里云反爬保护，服务器抓不到。保存后在下方列表点「推送脚本」，安装油猴脚本到浏览器即可自动上报（浏览器天然过反爬）。</p>
            <p v-if="form.auth_mode === 'cookie'" class="md:col-span-6 text-xs text-gray-500 dark:text-gray-400">Cookie 模式（不用油猴，服务器直接搜）：浏览器登录链动后台 → F12 → Network 任意请求 → 复制完整 Cookie 粘到上面。服务器带这段 Cookie 直接定时抓取。<b>Cookie 含反爬放行票，会过期，过期后需重新粘贴。</b></p>
          </div>
          <div class="mt-3 flex gap-2">
            <button class="rounded-lg bg-primary-600 px-4 py-2 text-sm text-white" @click="saveMonitor">{{ editingId ? '保存' : '新增' }}</button>
            <button v-if="editingId" class="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-dark-600 dark:text-gray-200" @click="resetForm">取消</button>
          </div>
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <table class="min-w-full text-sm">
            <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-300">
              <tr><th class="px-4 py-2 text-left">平台</th><th class="px-4 py-2 text-left">地址</th><th class="px-4 py-2">状态</th><th class="px-4 py-2">扫描</th><th class="px-4 py-2 text-right">操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="m in monitors" :key="m.id" class="border-t border-gray-100 dark:border-dark-700">
                <td class="px-4 py-3"><div class="font-medium text-gray-900 dark:text-white">{{ m.name }}</div><div class="text-xs text-gray-500">{{ m.last_error || '正常' }}</div></td>
                <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ platformAddressLabel(m.base_url) }}</td>
                <td class="px-4 py-3 text-center"><span :class="m.enabled ? 'bg-emerald-50 text-emerald-700' : 'bg-gray-100 text-gray-500'" class="rounded px-2 py-1 text-xs">{{ m.enabled ? '启用' : '停用' }}</span></td>
                <td class="px-4 py-3 text-center text-gray-500">{{ m.fetch_pages }} 页 / {{ m.interval_seconds }} 秒</td>
                <td class="px-4 py-3 text-right">
                  <button v-if="m.push_mode" class="mr-2 text-emerald-600" @click="openPush(m)">推送脚本</button>
                  <button v-else class="mr-2 text-primary-600" @click="runProbe(m)">扫描</button>
                  <button class="mr-2 text-gray-600 dark:text-gray-300" @click="editMonitor(m)">编辑</button>
                  <button class="text-red-600" @click="deleteMonitor(m)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <table class="min-w-full text-sm">
          <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-300">
            <tr><th class="px-4 py-2 text-left">时间</th><th class="px-4 py-2 text-left">平台</th><th class="px-4 py-2 text-left">商品</th><th class="px-4 py-2">事件</th><th class="px-4 py-2">价格</th><th class="px-4 py-2">库存</th></tr>
          </thead>
          <tbody>
            <tr v-for="e in eventRows" :key="e.id" class="border-t border-gray-100 dark:border-dark-700">
              <td class="px-4 py-3 text-gray-500">{{ dateTime(e.detected_at) }}</td>
              <td class="px-4 py-3">{{ e.platform }}</td>
              <td class="px-4 py-3">{{ e.title }}</td>
              <td class="px-4 py-3 text-center"><span :class="eventClass(e.event_type)" class="rounded px-2 py-1 text-xs">{{ eventLabel(e.event_type) }}</span></td>
              <td class="px-4 py-3 text-center">¥{{ money(e.old_price) }} -> ¥{{ money(e.new_price) }}</td>
              <td class="px-4 py-3 text-center">{{ e.old_stock ?? '-' }} -> {{ e.new_stock ?? '-' }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>

    <BaseDialog :show="pushModal.open" title="浏览器推送设置" width="wide" @close="pushModal.open = false">
      <div v-if="pushModal.monitor" class="space-y-4 text-sm">
        <p class="text-gray-600 dark:text-gray-300">
          链动小铺接口被阿里云反爬保护，服务器抓不到。装上这个油猴脚本，
          <b>它在你浏览器里抓数据（天然过反爬）并自动推送到本平台</b>。脚本里已经填好你的专属密钥。
        </p>
        <div>
          <div class="mb-1 font-medium text-gray-700 dark:text-gray-200">推送地址</div>
          <div class="flex items-center gap-2">
            <code class="flex-1 truncate rounded bg-gray-100 px-2 py-1 text-xs dark:bg-dark-700">{{ ingestUrl }}</code>
            <button class="rounded border border-gray-300 px-2 py-1 text-xs dark:border-dark-600 dark:text-gray-200" @click="copyText(ingestUrl)">复制</button>
          </div>
        </div>
        <div>
          <div class="mb-1 flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-gray-200">推送密钥</span>
            <button class="text-xs text-red-500 hover:underline" @click="regenerateKey(pushModal.monitor)">重置密钥</button>
          </div>
          <div class="flex items-center gap-2">
            <code class="flex-1 truncate rounded bg-gray-100 px-2 py-1 text-xs dark:bg-dark-700">{{ pushModal.monitor.ingest_key }}</code>
            <button class="rounded border border-gray-300 px-2 py-1 text-xs dark:border-dark-600 dark:text-gray-200" @click="copyText(pushModal.monitor.ingest_key)">复制</button>
          </div>
        </div>
        <div>
          <div class="mb-1 flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-gray-200">油猴脚本（已填好密钥）</span>
            <div class="flex gap-2">
              <button class="rounded border border-gray-300 px-2 py-1 text-xs dark:border-dark-600 dark:text-gray-200" @click="copyText(userscript)">复制脚本</button>
              <button class="rounded bg-primary-600 px-2 py-1 text-xs text-white" @click="downloadUserscript()">下载 .user.js</button>
            </div>
          </div>
          <textarea readonly class="h-48 w-full rounded-lg border border-gray-200 bg-gray-50 p-2 font-mono text-[11px] dark:border-dark-600 dark:bg-dark-900 dark:text-gray-200" :value="userscript"></textarea>
        </div>
        <ol class="list-decimal space-y-1 pl-5 text-xs text-gray-500 dark:text-gray-400">
          <li>浏览器装 <a href="https://www.tampermonkey.net/" target="_blank" class="text-primary-600 underline">Tampermonkey</a> 扩展。</li>
          <li>点「下载 .user.js」或「复制脚本」→ Tampermonkey 新建脚本粘贴 → 保存。</li>
          <li>登录 <code>pay.ldxp.cn</code> 后台并保持一个标签页打开，脚本会按间隔自动抓取并推送。</li>
          <li>本页「变化记录 / 全平台搜索」即可看到推送来的商品与涨跌。</li>
        </ol>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import cardAPI, { type CardPlatformMonitor, type CardProduct, type CardPriceEvent, type CardSummary, type CardEventType, type CardAuthMode } from '@/api/admin/cardPlatformMonitor'
import { useAppStore } from '@/stores'

const tabs = [
  { key: 'search', label: '全平台搜索' },
  { key: 'platforms', label: '监控平台' },
  { key: 'events', label: '变化记录' },
] as const

const activeTab = ref<(typeof tabs)[number]['key']>('search')
const monitors = ref<CardPlatformMonitor[]>([])
const productRows = ref<CardProduct[]>([])
const eventRows = ref<CardPriceEvent[]>([])
const productsLoading = ref(false)
const summaryData = reactive<CardSummary>({ platform_count: 0, product_count: 0, price_down: 0, restock: 0, error_count: 0 })
const editingId = ref<number | null>(null)
const appStore = useAppStore()

const productQuery = reactive({ search: 'gpt', monitor_id: 0, sort: 'priceAsc', in_stock: true })
const form = reactive({ name: '', platform_type: 'ldxp' as const, base_url: '', shop_url: '', auth_mode: 'push' as CardAuthMode, credential: '', enabled: true, interval_seconds: 300, fetch_pages: 5, note: '' })

// 浏览器推送设置弹窗
const pushModal = reactive({ open: false, monitor: null as CardPlatformMonitor | null })
const ingestUrl = computed(() => `${window.location.origin}/api/v1/card-ingest`)
const userscript = computed(() => pushModal.monitor ? buildUserscript(pushModal.monitor, ingestUrl.value) : '')

function openPush(m: CardPlatformMonitor) {
  pushModal.monitor = m
  pushModal.open = true
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess('已复制')
  } catch {
    appStore.showError('复制失败，请手动选择复制')
  }
}

async function regenerateKey(m: CardPlatformMonitor) {
  if (!confirm('重置后旧脚本立即失效，需要重新安装新脚本，确认？')) return
  try {
    const updated = await cardAPI.regenerateKey(m.id)
    pushModal.monitor = updated
    await reloadMonitors()
    appStore.showSuccess('密钥已重置')
  } catch (err) {
    appStore.showError(errorText(err, '重置失败'))
  }
}

function downloadUserscript() {
  if (!pushModal.monitor) return
  const blob = new Blob([userscript.value], { type: 'text/javascript' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `ldxp-push-${pushModal.monitor.id}.user.js`
  a.click()
  URL.revokeObjectURL(a.href)
}

function buildUserscript(m: CardPlatformMonitor, url: string): string {
  const intervalMin = Math.max(1, Math.round((m.interval_seconds || 300) / 60))
  const pages = Math.max(1, m.fetch_pages || 5)
  return `// ==UserScript==
// @name         链动小铺价格推送 - ${m.name}
// @namespace    sub2api-card-monitor
// @version      1.0
// @description  在浏览器抓取链动小铺货源并推送到监控平台（自动过反爬）
// @match        https://pay.ldxp.cn/*
// @match        https://admin.ldxp.cn/*
// @match        https://www.ldxp.cn/*
// @grant        GM_xmlhttpRequest
// @connect      ${(() => { try { return new URL(url).hostname } catch { return '*' } })()}
// @run-at       document-idle
// ==/UserScript==
(function () {
  'use strict';
  var INGEST_URL = ${JSON.stringify(url)};
  var KEY = ${JSON.stringify(m.ingest_key)};
  var PAGES = ${pages};
  var INTERVAL_MS = ${intervalMin} * 60 * 1000;
  var API = '/merchantApi/MyParent/searchGoodsList';

  function getToken() {
    var keys = ['auth-token','Merchant-Token','merchant-token','token','Authorization','access_token'];
    var stores = [localStorage, sessionStorage];
    for (var s = 0; s < stores.length; s++) {
      var store = stores[s];
      for (var i = 0; i < keys.length; i++) {
        var v = store.getItem(keys[i]);
        var token = readTokenValue(v);
        if (token) return token;
      }
      for (var j = 0; j < store.length; j++) {
        var key = store.key(j) || '';
        if (!/token|auth|merchant/i.test(key)) continue;
        var token2 = readTokenValue(store.getItem(key));
        if (token2) return token2;
      }
    }
    return '';
  }

  function readTokenValue(v) {
    if (!v) return '';
    try {
      var p = JSON.parse(v);
      v = p.value || p.token || p.access_token || p.auth_token || p.merchantToken || v;
    } catch (e) {}
    return String(v).replace(/^Bearer\\s+/i, '').trim();
  }

  async function fetchPage(token, page) {
    var resp = await fetch(API, {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json;charset=UTF-8', 'Accept': 'application/json, text/plain, */*', 'Merchant-Token': token },
      body: JSON.stringify({ current: page, pageSize: 50, name: '', goods_type: '', keywords: '' })
    });
    var data = await resp.json();
    if (data.code !== 1) throw new Error(data.msg || data.message || ('code ' + data.code));
    return (data.data && data.data.list) || [];
  }

  function push(products) {
    return new Promise(function (resolve, reject) {
      GM_xmlhttpRequest({
        method: 'POST', url: INGEST_URL,
        headers: { 'Content-Type': 'application/json' },
        data: JSON.stringify({ key: KEY, products: products }),
        onload: function (r) { (r.status >= 200 && r.status < 300) ? resolve(r) : reject(new Error('ingest HTTP ' + r.status + ' ' + r.responseText)); },
        onerror: function () { reject(new Error('ingest network error')); }
      });
    });
  }

  async function run() {
    try {
      var token = getToken();
      if (!token) { console.warn('[ldxp-push] 未登录，跳过'); return; }
      var all = [];
      for (var p = 1; p <= PAGES; p++) {
        var list = await fetchPage(token, p);
        all = all.concat(list);
        if (list.length < 50) break;
      }
      if (all.length) { await push(all); console.log('[ldxp-push] 已推送 ' + all.length + ' 个商品'); }
    } catch (e) { console.error('[ldxp-push] 失败', e); }
  }

  setTimeout(run, 4000);
  setInterval(run, INTERVAL_MS);
})();
`
}

const groupedProducts = computed(() => {
  const map = new Map<string, CardProduct[]>()
  for (const item of productRows.value) {
    const key = item.platform_name || '未知平台'
    map.set(key, [...(map.get(key) || []), item])
  }
  return Array.from(map.entries()).map(([name, items]) => {
    const merchantMap = new Map<string, CardProduct[]>()
    for (const item of items) {
      const merchant = item.merchant || '未知店铺'
      merchantMap.set(merchant, [...(merchantMap.get(merchant) || []), item])
    }
    const merchants = Array.from(merchantMap.entries()).map(([merchantName, merchantItems]) => ({
      name: merchantName,
      items: merchantItems,
    }))
    return { name, items, merchants }
  })
})

async function reloadAll() {
  await Promise.all([reloadMonitors(), reloadProducts(), reloadEvents(), reloadSummary()])
}

async function reloadMonitors() {
  const res = await cardAPI.list({ page_size: 100 })
  monitors.value = res.items
}

async function reloadProducts() {
  productsLoading.value = true
  try {
    const res = await cardAPI.products({ page_size: 100, search: productQuery.search, monitor_id: productQuery.monitor_id || undefined, sort: productQuery.sort, in_stock: productQuery.in_stock })
    productRows.value = res.items
    await reloadSummary()
  } finally {
    productsLoading.value = false
  }
}

async function reloadEvents() {
  const res = await cardAPI.events({ page_size: 100, search: productQuery.search })
  eventRows.value = res.items
}

async function reloadSummary() {
  Object.assign(summaryData, await cardAPI.summary(productQuery.search))
}

async function saveMonitor() {
  try {
    const payload = { ...form }
    if (editingId.value && !payload.credential) delete (payload as any).credential
    if (editingId.value) await cardAPI.update(editingId.value, payload)
    else await cardAPI.create(payload)
    appStore.showSuccess(editingId.value ? '保存成功' : '新增成功')
    resetForm()
    await reloadMonitors()
  } catch (err) {
    appStore.showError(errorText(err, '保存失败'))
  }
}

function editMonitor(m: CardPlatformMonitor) {
  editingId.value = m.id
  // 后端把「自动」存成哨兵 'ldxp'，编辑时显示为空，避免误当成地址校验失败。
  Object.assign(form, { name: m.name, platform_type: m.platform_type, base_url: m.base_url === 'ldxp' ? '' : m.base_url, shop_url: m.shop_url, auth_mode: m.auth_mode, credential: '', enabled: m.enabled, interval_seconds: m.interval_seconds, fetch_pages: m.fetch_pages, note: m.note })
}

function resetForm() {
  editingId.value = null
  Object.assign(form, { name: '', platform_type: 'ldxp', base_url: '', shop_url: '', auth_mode: 'push', credential: '', enabled: true, interval_seconds: 300, fetch_pages: 5, note: '' })
}

async function runProbe(m: CardPlatformMonitor) {
  try {
    const res = await cardAPI.probe(m.id)
    appStore.showSuccess(`扫描完成，商品 ${res.products.length} 个，变化 ${res.events.length} 个`)
    await reloadAll()
  } catch (err) {
    appStore.showError(errorText(err, '扫描失败'), 8000)
    await reloadMonitors()
  }
}

async function deleteMonitor(m: CardPlatformMonitor) {
  if (!confirm(`确认删除 ${m.name}？`)) return
  await cardAPI.del(m.id)
  await reloadAll()
}

function money(v: number | null | undefined) { return typeof v === 'number' ? v.toFixed(2).replace(/\.00$/, '') : '-' }
function dateTime(v: string | null | undefined) { return v ? new Date(v).toLocaleString() : '-' }
function promoPrice(p: CardProduct) { return p.cost_price != null && p.cost_price !== p.price ? `¥${money(p.cost_price)}` : '—' }
function stockText(v: number | null | undefined) { return typeof v === 'number' ? String(v) : '有货' }
function shortId(v: string | null | undefined) { return v ? (v.length > 8 ? `${v.slice(0, 6)}…` : v) : '-' }
function platformAddressLabel(v: string | null | undefined) { return !v || v === 'ldxp' ? '自动预设' : v }
function relativeTime(v: string | null | undefined) {
  if (!v) return '-'
  const diff = Date.now() - new Date(v).getTime()
  if (!Number.isFinite(diff) || diff < 0) return dateTime(v)
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  return `${Math.floor(hours / 24)}天前`
}
function siteToneClass(index: number) {
  return [
    'bg-blue-50 dark:bg-blue-900/20',
    'bg-emerald-50 dark:bg-emerald-900/20',
    'bg-amber-50 dark:bg-amber-900/20',
    'bg-violet-50 dark:bg-violet-900/20',
  ][index % 4]
}
function merchantToneClass(index: number) {
  return [
    'bg-gray-50 dark:bg-dark-700/60',
    'bg-blue-50/60 dark:bg-blue-900/10',
    'bg-emerald-50/60 dark:bg-emerald-900/10',
    'bg-amber-50/60 dark:bg-amber-900/10',
  ][index % 4]
}
function statusLabel(v: string) { return ({ online: '上架', offline: '下架', sold_out: '售罄', unknown: '未知' } as Record<string, string>)[v] || v }
function statusClass(v: string) { return v === 'online' ? 'bg-emerald-50 text-emerald-700' : v === 'sold_out' ? 'bg-amber-50 text-amber-700' : 'bg-gray-100 text-gray-600' }
function eventLabel(v: CardEventType | string) { return ({ new_product: '新商品', price_down: '降价', price_up: '涨价', new_low: '新低价', restock: '补货', sold_out: '售罄', offline: '下架', online: '上架', changed: '变化' } as Record<string, string>)[v] || v }
function eventClass(v: CardEventType | string) { return ['price_down', 'new_low', 'restock'].includes(v) ? 'bg-emerald-50 text-emerald-700' : ['price_up', 'offline', 'sold_out'].includes(v) ? 'bg-red-50 text-red-700' : 'bg-blue-50 text-blue-700' }
function errorText(err: unknown, fallback: string) {
  if (err && typeof err === 'object' && 'response' in err) {
    const data = (err as { response?: { data?: { detail?: unknown; message?: unknown; error?: unknown } } }).response?.data
    const message = data?.detail || data?.message || data?.error
    if (message) return String(message)
  }
  if (err && typeof err === 'object' && 'message' in err) {
    const message = String((err as { message?: unknown }).message || '')
    if (message) return message
  }
  return fallback
}

watch(activeTab, () => { if (activeTab.value === 'events') reloadEvents() })
onMounted(reloadAll)
</script>

<style scoped>
.input {
  @apply rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white;
}

.monitor-table thead {
  @apply bg-gray-50 text-xs font-semibold uppercase tracking-normal text-gray-500 dark:bg-dark-700 dark:text-gray-300;
}

.monitor-table tbody tr {
  @apply border-t border-gray-100 bg-white dark:border-dark-700 dark:bg-dark-800;
}

.monitor-table tbody tr:hover {
  @apply bg-gray-50 dark:bg-dark-700/60;
}
</style>
