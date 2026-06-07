<template>
  <AppLayout>
    <div class="space-y-5">
      <!-- 顶部 KPI 条 -->
      <div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.relayMonitor.kpiSites') }}</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ siteCount }}</p>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.relayMonitor.kpiGroups') }}</p>
          <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ groupCount }}</p>
        </div>
        <div class="rounded-xl border border-red-100 bg-red-50 p-4 dark:border-red-900/40 dark:bg-red-950/30">
          <p class="text-xs font-medium text-red-600 dark:text-red-400">{{ t('admin.relayMonitor.upAnnouncements') }}</p>
          <p class="mt-1 text-2xl font-bold text-red-600 dark:text-red-400">{{ summary.up_count }}</p>
        </div>
        <div class="rounded-xl border border-green-100 bg-green-50 p-4 dark:border-green-900/40 dark:bg-green-950/30">
          <p class="text-xs font-medium text-green-600 dark:text-green-400">{{ t('admin.relayMonitor.downAnnouncements') }}</p>
          <p class="mt-1 text-2xl font-bold text-green-600 dark:text-green-400">{{ summary.down_count }}</p>
        </div>
        <div class="col-span-2 rounded-xl border border-blue-100 bg-blue-50 p-4 dark:border-blue-900/40 dark:bg-blue-950/30 md:col-span-3 xl:col-span-1">
          <p class="text-xs font-medium text-blue-600 dark:text-blue-400">{{ t('admin.relayMonitor.lastRefresh') }}</p>
          <p class="mt-1 text-sm font-semibold text-blue-700 dark:text-blue-300">{{ lastRefreshLabel }}</p>
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

      <!-- 统一筛选 -->
      <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="t('admin.relayMonitor.searchPlaceholder')"
            class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white xl:col-span-2"
            @keyup.enter="reloadChanges"
          />
          <select
            v-model="siteFilter"
            class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            @change="reloadChanges"
          >
            <option value="">{{ t('admin.relayMonitor.allSites') }}</option>
            <option v-for="site in siteFilterOptions" :key="site" :value="site">{{ site }}</option>
          </select>
          <select
            v-model="vendorFilter"
            class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            @change="reloadChanges"
          >
            <option value="">{{ t('admin.relayMonitor.allVendors') }}</option>
            <option v-for="vendor in vendorFilterOptions" :key="vendor" :value="vendor">{{ vendor }}</option>
          </select>
          <select
            v-model="systemFilter"
            class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            @change="reloadChanges"
          >
            <option value="">{{ t('admin.relayMonitor.allSystems') }}</option>
            <option v-for="system in systemFilterOptions" :key="system" :value="system">{{ system }}</option>
          </select>
          <select
            v-model="planFilter"
            class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            @change="reloadChanges"
          >
            <option value="">{{ t('admin.relayMonitor.allPlans') }}</option>
            <option v-for="p in availablePlanFilters" :key="p.key" :value="p.key">{{ p.label }}</option>
          </select>
        </div>
        <div class="mt-3 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <select
            v-if="activeTab === 'changes'"
            v-model="directionFilter"
            class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white md:w-48"
            @change="reloadChanges"
          >
            <option value="">{{ t('admin.relayMonitor.allAnnouncements') }}</option>
            <option value="up">{{ t('admin.relayMonitor.onlyUp') }}</option>
            <option value="down">{{ t('admin.relayMonitor.onlyDown') }}</option>
          </select>
          <p v-else class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.relayMonitor.filterHint') }}</p>
          <div class="flex gap-2">
            <button
              class="inline-flex items-center justify-center rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-700"
              @click="clearFilters"
            >
              {{ t('admin.relayMonitor.clearFilters') }}
            </button>
            <button
              class="inline-flex items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-60"
              :disabled="probingAll"
              @click="probeAll"
            >
              {{ probingAll ? t('admin.relayMonitor.probing') : t('admin.relayMonitor.probeAll') }}
            </button>
          </div>
        </div>
      </div>

      <!-- ============ 比价（默认） ============ -->
      <div v-show="activeTab === 'compare'" class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.relayMonitor.compareTitle') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.relayMonitor.compareHint') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <select
              v-model="compareSort"
              class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            >
              <option value="rate_asc">{{ t('admin.relayMonitor.sortRateAsc') }}</option>
              <option value="rate_desc">{{ t('admin.relayMonitor.sortRateDesc') }}</option>
              <option value="change">{{ t('admin.relayMonitor.sortChange') }}</option>
            </select>
            <span class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-300">
              {{ compareGroups.length }} {{ t('admin.relayMonitor.bucketsUnit') }}
            </span>
          </div>
        </div>

        <div v-if="compareByVendor.length" class="space-y-3">
          <div
            v-for="vs in compareByVendor"
            :key="vs.vendor"
            class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700"
          >
            <button
              type="button"
              class="flex w-full items-center gap-2 bg-gray-100 px-4 py-3 text-left transition-colors hover:bg-gray-200/70 dark:bg-dark-800 dark:hover:bg-dark-700"
              @click="toggleVendor(vs.vendor)"
            >
              <svg class="h-4 w-4 flex-shrink-0 text-gray-400 transition-transform" :class="isVendorCollapsed(vs.vendor) ? '-rotate-90' : ''" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="m6 9 6 6 6-6" />
              </svg>
              <span class="text-sm font-bold text-gray-900 dark:text-white">{{ vs.vendor }}</span>
              <span class="ml-auto text-xs text-gray-500 dark:text-gray-400">{{ vs.buckets.length }} {{ t('admin.relayMonitor.bucketsUnit') }}</span>
            </button>
            <div v-show="!isVendorCollapsed(vs.vendor)" class="divide-y divide-gray-100 dark:divide-dark-700">
              <div v-for="g in vs.buckets" :key="g.key">
                <div class="flex flex-wrap items-center gap-2 bg-gray-50/70 px-4 py-2 dark:bg-dark-800/40">
                  <span class="rounded px-2 py-0.5 text-xs font-medium" :class="g.tier.cls">{{ g.tier.label }}</span>
                  <span class="ml-auto text-xs text-gray-500 dark:text-gray-400">{{ g.rows.length }} {{ t('admin.relayMonitor.quotesUnit') }}</span>
                </div>
                <table class="min-w-full text-sm">
              <thead>
                <tr class="border-b border-gray-100 text-left text-[11px] text-gray-400 dark:border-dark-700/60">
                  <th class="w-12 px-3 py-2 font-medium">#</th>
                  <th class="px-3 py-2 font-medium">{{ t('admin.relayMonitor.colSite') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('admin.relayMonitor.colGroup') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('admin.relayMonitor.colCurrentRate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('admin.relayMonitor.colChange') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('admin.relayMonitor.colChangedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, i) in g.rows"
                  :key="row.monitor_id + '/' + row.group_name"
                  class="border-b border-gray-100 last:border-0 dark:border-dark-700/50"
                  :class="row.current_rate === g.minRate ? 'bg-emerald-50/70 dark:bg-emerald-950/20' : ''"
                >
                  <td class="px-3 py-2.5">
                    <span v-if="row.current_rate === g.minRate" class="text-base" :title="t('admin.relayMonitor.cheapest')">🏆</span>
                    <span v-else class="text-gray-400">{{ i + 1 }}</span>
                  </td>
                  <td class="px-3 py-2.5">
                    <div class="flex items-center gap-1.5">
                      <span class="font-medium text-gray-900 dark:text-white">{{ row.site }}</span>
                      <span class="rounded bg-indigo-50 px-1.5 py-0.5 text-[10px] text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300">{{ row.system }}</span>
                    </div>
                  </td>
                  <td class="max-w-[200px] truncate px-3 py-2.5" :title="t('admin.relayMonitor.viewHistory')">
                    <button class="truncate text-gray-500 hover:text-primary-600 hover:underline dark:text-gray-400" @click="openHistory(row)">{{ row.group_name }}</button>
                  </td>
                  <td class="px-3 py-2.5">
                    <div class="flex items-center gap-2">
                      <span class="font-semibold tabular-nums" :class="row.current_rate === g.minRate ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-900 dark:text-gray-100'">{{ formatRate(row.current_rate) }}</span>
                      <div class="hidden h-1.5 w-20 overflow-hidden rounded bg-gray-100 md:block dark:bg-dark-700">
                        <div class="h-full rounded" :class="row.current_rate === g.minRate ? 'bg-emerald-500' : 'bg-gray-300 dark:bg-dark-500'" :style="{ width: rateBarWidth(row.current_rate, g.maxRate) }"></div>
                      </div>
                    </div>
                  </td>
                  <td class="px-3 py-2.5">
                    <span
                      v-if="row.has_change"
                      class="font-medium"
                      :class="row.direction === 'up' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'"
                    >
                      {{ (row.direction === 'up' ? t('admin.relayMonitor.up') : t('admin.relayMonitor.down')) + ' ' + formatRate(Math.abs(row.new_rate - row.old_rate)) }}
                    </span>
                    <span v-else class="text-gray-300 dark:text-gray-600">—</span>
                  </td>
                  <td class="whitespace-nowrap px-3 py-2.5 text-gray-500">{{ row.changed_at ? formatTime(row.changed_at) : '-' }}</td>
                </tr>
              </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="!overviewLoading" class="rounded-lg border border-dashed border-gray-300 bg-white px-4 py-10 text-center text-gray-400 dark:border-dark-600 dark:bg-dark-800">
          {{ t('admin.relayMonitor.noOverview') }}
        </div>
      </div>

      <!-- ============ 倍率总览（默认） ============ -->
      <div v-show="activeTab === 'overview'" class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.relayMonitor.siteBoardTitle') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.relayMonitor.siteBoardHint') }}</p>
          </div>
          <span class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-300">
            {{ filteredOverview.length }} {{ t('admin.relayMonitor.groupsUnit') }}
          </span>
        </div>

        <div v-if="overviewSites.length" class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-200 bg-gray-50 text-left text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400">
                <th class="px-4 py-2.5 font-medium">{{ t('admin.relayMonitor.colPlan') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ t('admin.relayMonitor.colGroup') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ t('admin.relayMonitor.colCurrentRate') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ t('admin.relayMonitor.colChange') }}</th>
                <th class="px-4 py-2.5 font-medium">{{ t('admin.relayMonitor.colChangedAt') }}</th>
              </tr>
            </thead>
            <tbody
              v-for="(site, siteIndex) in overviewSites"
              :key="site.key"
              class="border-b-4 border-white dark:border-dark-900"
            >
              <!-- 站点一级标题行 -->
              <tr :class="siteTone(siteIndex).header">
                <td colspan="5" class="px-4 py-2">
                  <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
                    <button
                      class="text-base leading-none"
                      :class="isFav(site.site) ? 'text-amber-400' : 'text-gray-300 hover:text-amber-400 dark:text-gray-500'"
                      :title="isFav(site.site) ? t('admin.relayMonitor.unpin') : t('admin.relayMonitor.pin')"
                      @click="toggleFav(site.site)"
                    >★</button>
                    <span class="h-2.5 w-2.5 flex-shrink-0 rounded-full" :class="siteTone(siteIndex).dot"></span>
                    <span class="text-[15px] font-bold text-gray-900 dark:text-white">{{ site.site }}</span>
                    <span class="rounded bg-white/70 px-1.5 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-950/40 dark:text-gray-200">{{ site.system }}</span>
                    <span class="rounded bg-white/70 px-1.5 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-950/40 dark:text-gray-200">{{ site.vendor || t('admin.relayMonitor.noVendor') }}</span>
                    <span class="ml-auto flex flex-wrap items-center gap-1.5 text-[11px]">
                      <span class="rounded-full bg-white/80 px-2 py-0.5 text-gray-600 dark:bg-dark-950/50 dark:text-gray-300">{{ site.rows.length }} {{ t('admin.relayMonitor.groupsUnit') }}</span>
                      <span v-if="site.upCount" class="rounded-full bg-red-100 px-2 py-0.5 font-medium text-red-700 dark:bg-red-900/40 dark:text-red-300">{{ t('admin.relayMonitor.up') }} {{ site.upCount }}</span>
                      <span v-if="site.downCount" class="rounded-full bg-green-100 px-2 py-0.5 font-medium text-green-700 dark:bg-green-900/40 dark:text-green-300">{{ t('admin.relayMonitor.down') }} {{ site.downCount }}</span>
                      <span v-if="site.lastChangedAt" class="text-gray-500 dark:text-gray-400">· {{ formatTime(site.lastChangedAt) }}</span>
                    </span>
                  </div>
                </td>
              </tr>
              <!-- 分组行 -->
              <tr
                v-for="row in site.rows"
                :key="row.monitor_id + '/' + row.group_name"
                class="border-b border-gray-100 last:border-0 dark:border-dark-700/50"
                :class="row.has_change
                  ? (row.direction === 'up' ? 'bg-red-50/70 dark:bg-red-950/15' : 'bg-green-50/70 dark:bg-green-950/15')
                  : siteTone(siteIndex).row"
              >
                <td class="px-4 py-2.5">
                  <div class="flex items-center gap-1.5">
                    <span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ row.vendor || t('admin.relayMonitor.noVendor') }}</span>
                    <span class="rounded px-2 py-0.5 text-xs font-medium" :class="planTier(row.group_name).cls">{{ planTier(row.group_name).label }}</span>
                  </div>
                </td>
                <td class="px-4 py-2.5">
                  <button class="font-medium text-gray-900 hover:text-primary-600 hover:underline dark:text-white" :title="t('admin.relayMonitor.viewHistory')" @click="openHistory(row)">{{ row.group_name }}</button>
                </td>
                <td class="px-4 py-2.5 font-semibold text-gray-900 dark:text-gray-100">{{ formatRate(row.current_rate) }}</td>
                <td class="px-4 py-2.5">
                  <span
                    v-if="row.has_change"
                    class="font-medium"
                    :class="row.direction === 'up' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'"
                  >
                    {{ (row.direction === 'up' ? t('admin.relayMonitor.up') : t('admin.relayMonitor.down')) + ' ' + formatRate(Math.abs(row.new_rate - row.old_rate)) }}
                    <span class="ml-1 font-normal text-gray-400">({{ formatRate(row.old_rate) }} → {{ formatRate(row.new_rate) }})</span>
                  </span>
                  <span v-else class="text-gray-400">{{ t('admin.relayMonitor.stable') }}</span>
                </td>
                <td class="px-4 py-2.5 whitespace-nowrap text-gray-500">{{ row.changed_at ? formatTime(row.changed_at) : '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="!overviewLoading" class="rounded-lg border border-dashed border-gray-300 bg-white px-4 py-10 text-center text-gray-400 dark:border-dark-600 dark:bg-dark-800">
          {{ t('admin.relayMonitor.noOverview') }}
        </div>
      </div>

      <!-- ============ 公告看板 ============ -->
      <div v-show="activeTab === 'changes'" class="space-y-4">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">
            <span class="text-red-500">{{ t('admin.relayMonitor.legendUp') }}</span>
            {{ t('admin.relayMonitor.legendSep') }}
            <span class="text-green-500">{{ t('admin.relayMonitor.legendDown') }}</span>
          </p>
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
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colPlan') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colOldRate') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colNewRate') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colChange') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colTime') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.relayMonitor.colContent') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, idx) in filteredChanges"
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
                <td class="px-4 py-3">
                  <span class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium" :class="planTier(row.group_name).cls">{{ planTier(row.group_name).label }}</span>
                </td>
                <td class="px-4 py-3 text-gray-500">{{ formatRate(row.old_rate) }}</td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-200">{{ formatRate(row.new_rate) }}</td>
                <td class="px-4 py-3 font-medium" :class="row.direction === 'up' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'">
                  {{ (row.direction === 'up' ? t('admin.relayMonitor.up') : t('admin.relayMonitor.down')) + ' ' + formatRate(Math.abs(row.new_rate - row.old_rate)) }}
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-gray-500">{{ formatTime(row.detected_at) }}</td>
                <td class="px-4 py-3 max-w-xs truncate text-gray-500" :title="row.content">{{ row.content }}</td>
              </tr>
              <tr v-if="!changesLoading && filteredChanges.length === 0">
                <td colspan="12" class="px-4 py-10 text-center text-gray-400">{{ t('admin.relayMonitor.noChanges') }}</td>
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
            <input v-model="form.vendor" list="relay-vendor-options" type="text" placeholder="OpenAI" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
            <datalist id="relay-vendor-options">
              <option v-for="v in vendorOptions" :key="v" :value="v" />
            </datalist>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.colInterval') }} (s)</label>
            <input v-model.number="form.interval_seconds" type="number" min="60" max="86400" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </div>
          <template v-if="form.system === 'sub2api'">
            <div class="md:col-span-2">
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.relayMonitor.authMethod') }}</label>
              <div class="flex gap-4 text-sm text-gray-700 dark:text-gray-200">
                <label class="flex cursor-pointer items-center gap-1.5"><input type="radio" value="password" v-model="form.auth_mode" /> {{ t('admin.relayMonitor.authModePassword') }}</label>
                <label class="flex cursor-pointer items-center gap-1.5"><input type="radio" value="token" v-model="form.auth_mode" /> {{ t('admin.relayMonitor.authModeToken') }}</label>
              </div>
            </div>
            <template v-if="form.auth_mode === 'password'">
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.relayMonitor.authAccount') }}
                  <span class="text-xs font-normal text-gray-400">{{ t('admin.relayMonitor.credentialRequired') }}</span>
                </label>
                <input v-model="form.auth_account" type="email" autocomplete="off" placeholder="you@example.com" class="w-full rounded-lg border px-3 py-2 text-sm dark:bg-dark-700 dark:text-white" :class="form.auth_account && !isEmail(form.auth_account) ? 'border-red-400 dark:border-red-500' : 'border-gray-300 dark:border-dark-600'" />
                <p v-if="form.auth_account && !isEmail(form.auth_account)" class="mt-1 text-xs text-red-500">{{ t('admin.relayMonitor.authAccountEmailHint') }}</p>
              </div>
              <div>
                <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.relayMonitor.password') }}
                  <span class="text-xs font-normal text-gray-400">{{ t('admin.relayMonitor.credentialRequired') }}</span>
                </label>
                <input v-model="form.credential" type="password" autocomplete="new-password" :placeholder="editing ? t('admin.relayMonitor.credentialKeep') : t('admin.relayMonitor.passwordPlaceholder')" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
              </div>
              <p class="md:col-span-2 -mt-2 text-xs text-gray-400">{{ t('admin.relayMonitor.sub2apiAuthHint') }}</p>
            </template>
            <div v-else class="md:col-span-2">
              <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.relayMonitor.tokenLabel') }}
                <span class="text-xs font-normal text-gray-400">{{ t('admin.relayMonitor.credentialRequired') }}</span>
              </label>
              <input v-model="form.credential" type="password" autocomplete="new-password" :placeholder="editing ? t('admin.relayMonitor.credentialKeep') : 'eyJhbGciOi...'" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
              <p class="mt-1 text-xs text-gray-400">{{ t('admin.relayMonitor.tokenHint') }}</p>
            </div>
          </template>
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

    <!-- 倍率历史折线 -->
    <BaseDialog :show="history.open" :title="t('admin.relayMonitor.historyTitle')" width="wide" @close="history.open = false">
      <div class="space-y-3">
        <p class="text-sm text-gray-600 dark:text-gray-300">
          <span class="font-semibold text-gray-900 dark:text-white">{{ history.site }}</span>
          <span class="mx-1 text-gray-400">/</span>{{ history.group }}
        </p>
        <div v-if="history.loading" class="flex h-64 items-center justify-center text-sm text-gray-400">{{ t('common.loading') }}</div>
        <div v-else-if="historyChartData" class="h-64">
          <Line :data="historyChartData" :options="historyChartOptions" />
        </div>
        <div v-else class="flex h-64 items-center justify-center text-sm text-gray-400">{{ t('admin.relayMonitor.noHistory') }}</div>
      </div>
    </BaseDialog>
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
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { relayMonitorAPI } from '@/api/admin/relayMonitor'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend)
import type {
  RelayMonitor,
  RelayRateChange,
  RelayOverviewRow,
  RelayGroupRate,
  RelaySystem,
  RateDirection,
} from '@/api/admin/relayMonitor'

const { t } = useI18n()
const appStore = useAppStore()

const tabs = computed(() => [
  { key: 'compare' as const, label: t('admin.relayMonitor.tabCompare') },
  { key: 'overview' as const, label: t('admin.relayMonitor.tabOverview') },
  { key: 'changes' as const, label: t('admin.relayMonitor.tabChanges') },
  { key: 'sites' as const, label: t('admin.relayMonitor.tabSites') },
])
const activeTab = ref<'compare' | 'overview' | 'changes' | 'sites'>('compare')

// ---- 汇总 ----
const summary = ref({ up_count: 0, down_count: 0 })
const lastRefresh = ref<string>('')
const lastRefreshLabel = computed(() => (lastRefresh.value ? formatTime(lastRefresh.value) : '-'))

// ---- 倍率总览 ----
const overview = ref<RelayOverviewRow[]>([])
const overviewLoading = ref(false)

// ---- 公告历史 ----
const changes = ref<RelayRateChange[]>([])
const changesTotal = ref(0)
const changesPage = ref(1)
const changesPageSize = ref(20)
const changesLoading = ref(false)
const directionFilter = ref<'' | RateDirection>('')
const searchQuery = ref('')
const siteFilter = ref('')
const vendorFilter = ref('')
const systemFilter = ref<'' | RelaySystem>('')
const probingAll = ref(false)

// 比价排序：倍率升 / 倍率降 / 涨跌幅
const compareSort = ref<'rate_asc' | 'rate_desc' | 'change'>('rate_asc')

// 收藏/置顶站点（localStorage 持久化，按站点名）
const FAV_KEY = 'relay_fav_sites'
const favorites = ref<Set<string>>(new Set())
function loadFavorites() {
  try {
    const raw = localStorage.getItem(FAV_KEY)
    if (raw) favorites.value = new Set(JSON.parse(raw) as string[])
  } catch { /* ignore */ }
}
function isFav(site: string): boolean {
  return favorites.value.has(site)
}
function toggleFav(site: string) {
  const next = new Set(favorites.value)
  if (next.has(site)) next.delete(site)
  else next.add(site)
  favorites.value = next
  try { localStorage.setItem(FAV_KEY, JSON.stringify([...next])) } catch { /* ignore */ }
}

// 倍率历史折线弹窗
const history = reactive({
  open: false,
  loading: false,
  site: '',
  group: '',
  points: [] as { t: string; rate: number }[],
})

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
  auth_mode: 'password' as 'password' | 'token',
  auth_account: '',
  credential: '',
  watched_groups: [] as string[],
  interval_seconds: 300,
  enabled: true,
})

// Token 模式不带邮箱（后端据此把 credential 当 Bearer token）。
function effectiveAuthAccount(): string {
  if (form.system === 'sub2api' && form.auth_mode === 'token') return ''
  return form.auth_account.trim()
}

// 厂商下拉候选（datalist，仍可自定义输入）。
const vendorOptions = ['OpenAI', 'Claude', 'Gemini', 'Grok', 'DeepSeek']

// 套餐档位：按分组名关键字识别（覆盖 OpenAI/Claude/Gemini/Grok 各家）。
// 顺序由具体到一般：team→enterprise→max→ultra→pro→plus→free。
interface PlanTier {
  key: string
  label: string
  cls: string
  panelCls: string
  borderCls: string
}
interface OverviewSite {
  key: string
  site: string
  system: RelaySystem
  vendor: string
  rows: RelayOverviewRow[]
  upCount: number
  downCount: number
  lastChangedAt: string | null
}

interface SiteTone {
  header: string
  row: string
  dot: string
}

const PLAN_OTHER: PlanTier = {
  key: 'other',
  label: 'Other',
  cls: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300',
  panelCls: 'bg-gray-50 dark:bg-dark-800/80',
  borderCls: 'border-gray-200 dark:border-dark-700',
}
function planTier(group: string): PlanTier {
  const g = (group || '').toLowerCase()
  if (g.includes('team')) return { key: 'team', label: 'Team', cls: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300', panelCls: 'bg-purple-50/80 dark:bg-purple-950/20', borderCls: 'border-purple-100 dark:border-purple-900/40' }
  if (g.includes('enterprise') || /\bent\b/.test(g)) return { key: 'enterprise', label: 'Enterprise', cls: 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-200', panelCls: 'bg-slate-50 dark:bg-slate-900/30', borderCls: 'border-slate-200 dark:border-slate-700' }
  if (g.includes('max')) return { key: 'max', label: 'Max', cls: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300', panelCls: 'bg-orange-50/80 dark:bg-orange-950/20', borderCls: 'border-orange-100 dark:border-orange-900/40' }
  if (g.includes('ultra')) return { key: 'ultra', label: 'Ultra', cls: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300', panelCls: 'bg-amber-50/80 dark:bg-amber-950/20', borderCls: 'border-amber-100 dark:border-amber-900/40' }
  if (/\bpro\b/.test(g) || g.includes('pro')) return { key: 'pro', label: 'Pro', cls: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300', panelCls: 'bg-blue-50/80 dark:bg-blue-950/20', borderCls: 'border-blue-100 dark:border-blue-900/40' }
  if (g.includes('plus')) return { key: 'plus', label: 'Plus', cls: 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300', panelCls: 'bg-teal-50/80 dark:bg-teal-950/20', borderCls: 'border-teal-100 dark:border-teal-900/40' }
  if (g.includes('free')) return { key: 'free', label: 'Free', cls: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300', panelCls: 'bg-green-50/80 dark:bg-green-950/20', borderCls: 'border-green-100 dark:border-green-900/40' }
  return PLAN_OTHER
}

// 每个站点一个色系：标题行用较深的 header 背景，分组行用同色系的浅 row 背景，
// 相邻站点颜色不同，便于区分「哪些分组属于哪个站点」。
const SITE_TONES: SiteTone[] = [
  { header: 'bg-sky-100 dark:bg-sky-900/40', row: 'bg-sky-50/50 dark:bg-sky-950/15', dot: 'bg-sky-500' },
  { header: 'bg-violet-100 dark:bg-violet-900/40', row: 'bg-violet-50/50 dark:bg-violet-950/15', dot: 'bg-violet-500' },
  { header: 'bg-emerald-100 dark:bg-emerald-900/40', row: 'bg-emerald-50/50 dark:bg-emerald-950/15', dot: 'bg-emerald-500' },
  { header: 'bg-amber-100 dark:bg-amber-900/40', row: 'bg-amber-50/50 dark:bg-amber-950/15', dot: 'bg-amber-500' },
  { header: 'bg-rose-100 dark:bg-rose-900/40', row: 'bg-rose-50/50 dark:bg-rose-950/15', dot: 'bg-rose-500' },
  { header: 'bg-cyan-100 dark:bg-cyan-900/40', row: 'bg-cyan-50/50 dark:bg-cyan-950/15', dot: 'bg-cyan-500' },
]

function siteTone(index: number): SiteTone {
  return SITE_TONES[index % SITE_TONES.length]
}

function normalizeVendor(vendor: string): string {
  const v = vendor.trim()
  return v || t('admin.relayMonitor.noVendor')
}

// 套餐筛选（客户端过滤；overview 返回全量无分页）。
const planFilter = ref('')
const filteredOverview = computed(() => {
  return overview.value.filter((r) => matchesClientFilters(r))
})

const filteredChanges = computed(() => changes.value.filter((r) => matchesClientFilters(r)))

const siteFilterOptions = computed(() => uniqueSorted([
  ...overview.value.map((r) => r.site),
  ...changes.value.map((r) => r.site),
  ...monitors.value.map((m) => m.name),
]))

const vendorFilterOptions = computed(() => uniqueSorted([
  ...overview.value.map((r) => normalizeVendor(r.vendor)),
  ...changes.value.map((r) => normalizeVendor(r.vendor)),
  ...monitors.value.map((m) => normalizeVendor(m.vendor)),
]))

const systemFilterOptions = computed(() => uniqueSorted([
  ...overview.value.map((r) => r.system),
  ...changes.value.map((r) => r.system),
  ...monitors.value.map((m) => m.system),
]) as RelaySystem[])

const availablePlanFilters = computed(() => {
  const seen = new Set<string>()
  const out: PlanTier[] = []
  for (const row of [...overview.value, ...changes.value]) {
    const tier = planTier(row.group_name)
    if (!seen.has(tier.key)) {
      seen.add(tier.key)
      out.push(tier)
    }
  }
  return out
})

const overviewSites = computed<OverviewSite[]>(() => {
  const siteMap = new Map<string, RelayOverviewRow[]>()
  for (const row of filteredOverview.value) {
    const key = `${row.monitor_id}:${row.site}`
    const rows = siteMap.get(key) ?? []
    rows.push(row)
    siteMap.set(key, rows)
  }

  const sites = Array.from(siteMap.entries()).map(([key, rows]) => {
    const first = rows[0]
    const sortedRows = [...rows].sort(compareOverviewRows)
    return {
      key,
      site: first.site,
      system: first.system,
      vendor: first.vendor,
      rows: sortedRows,
      upCount: rows.filter((r) => r.has_change && r.direction === 'up').length,
      downCount: rows.filter((r) => r.has_change && r.direction === 'down').length,
      lastChangedAt: newestChangedAt(rows),
    }
  })
  // 收藏的站点置顶
  return sites.sort((a, b) => {
    const fa = isFav(a.site) ? 0 : 1
    const fb = isFav(b.site) ? 0 : 1
    return fa - fb || a.site.localeCompare(b.site)
  })
})

// ---- 比价视图：按 厂商·套餐 聚合，组内站点按当前倍率升序 ----
const PLAN_ORDER: Record<string, number> = {
  free: 0, plus: 1, pro: 2, max: 3, team: 4, ultra: 5, enterprise: 6, other: 9,
}
interface CompareGroup {
  key: string
  vendor: string
  tier: PlanTier
  rows: RelayOverviewRow[]
  minRate: number
  maxRate: number
}
function compareRowSorter(a: RelayOverviewRow, b: RelayOverviewRow): number {
  if (compareSort.value === 'rate_desc') {
    return b.current_rate - a.current_rate || a.site.localeCompare(b.site)
  }
  if (compareSort.value === 'change') {
    const da = a.has_change ? Math.abs(a.new_rate - a.old_rate) : -1
    const db = b.has_change ? Math.abs(b.new_rate - b.old_rate) : -1
    return db - da || a.current_rate - b.current_rate
  }
  return a.current_rate - b.current_rate || a.site.localeCompare(b.site)
}
const compareGroups = computed<CompareGroup[]>(() => {
  const map = new Map<string, RelayOverviewRow[]>()
  for (const r of filteredOverview.value) {
    const key = `${normalizeVendor(r.vendor)}__${planTier(r.group_name).key}`
    const arr = map.get(key)
    if (arr) arr.push(r)
    else map.set(key, [r])
  }
  const groups = Array.from(map.entries()).map(([key, rows]) => ({
    key,
    vendor: normalizeVendor(rows[0].vendor),
    tier: planTier(rows[0].group_name),
    rows: [...rows].sort(compareRowSorter),
    minRate: Math.min(...rows.map((r) => r.current_rate)),
    maxRate: Math.max(...rows.map((r) => r.current_rate), 0),
  }))
  groups.sort((a, b) =>
    a.vendor.localeCompare(b.vendor) ||
    (PLAN_ORDER[a.tier.key] ?? 9) - (PLAN_ORDER[b.tier.key] ?? 9))
  return groups
})

// 比价二级分组：按厂商归类，厂商下是各套餐档位（compareGroups 已按厂商+档位排序）。
interface VendorSection {
  vendor: string
  buckets: CompareGroup[]
}
const compareByVendor = computed<VendorSection[]>(() => {
  const map = new Map<string, CompareGroup[]>()
  for (const g of compareGroups.value) {
    const arr = map.get(g.vendor)
    if (arr) arr.push(g)
    else map.set(g.vendor, [g])
  }
  return Array.from(map.entries()).map(([vendor, buckets]) => ({ vendor, buckets }))
})

// 厂商折叠状态（localStorage 持久化）。
const COLLAPSE_KEY = 'relay_collapsed_vendors'
const collapsedVendors = ref<Set<string>>(new Set())
function loadCollapsed() {
  try {
    const raw = localStorage.getItem(COLLAPSE_KEY)
    if (raw) collapsedVendors.value = new Set(JSON.parse(raw) as string[])
  } catch { /* ignore */ }
}
function isVendorCollapsed(v: string): boolean {
  return collapsedVendors.value.has(v)
}
function toggleVendor(v: string) {
  const next = new Set(collapsedVendors.value)
  if (next.has(v)) next.delete(v)
  else next.add(v)
  collapsedVendors.value = next
  try { localStorage.setItem(COLLAPSE_KEY, JSON.stringify([...next])) } catch { /* ignore */ }
}

// 倍率对比条宽度（相对组内最高倍率；越便宜越短）。
function rateBarWidth(rate: number, maxRate: number): string {
  if (!maxRate || maxRate <= 0) return '6%'
  const pct = Math.max(6, Math.round((rate / maxRate) * 100))
  return `${pct}%`
}

// KPI：当前筛选下的站点数 / 分组数。
const groupCount = computed(() => filteredOverview.value.length)
const siteCount = computed(() => new Set(filteredOverview.value.map((r) => r.site)).size)

function matchesClientFilters(row: RelayOverviewRow | RelayRateChange): boolean {
  if (siteFilter.value && row.site !== siteFilter.value) return false
  if (vendorFilter.value && normalizeVendor(row.vendor) !== vendorFilter.value) return false
  if (systemFilter.value && row.system !== systemFilter.value) return false
  if (planFilter.value && planTier(row.group_name).key !== planFilter.value) return false
  return true
}

function clearFilters() {
  searchQuery.value = ''
  siteFilter.value = ''
  vendorFilter.value = ''
  systemFilter.value = ''
  planFilter.value = ''
  directionFilter.value = ''
  reloadChanges()
}

function uniqueSorted(values: string[]): string[] {
  return Array.from(new Set(values.map((v) => v.trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b))
}

function compareOverviewRows(a: RelayOverviewRow, b: RelayOverviewRow): number {
  if (a.has_change !== b.has_change) return a.has_change ? -1 : 1
  const vendorCmp = normalizeVendor(a.vendor).localeCompare(normalizeVendor(b.vendor))
  if (vendorCmp !== 0) return vendorCmp
  const planCmp = planTier(a.group_name).label.localeCompare(planTier(b.group_name).label)
  if (planCmp !== 0) return planCmp
  return a.group_name.localeCompare(b.group_name)
}

function newestChangedAt(rows: RelayOverviewRow[]): string | null {
  return rows
    .map((r) => r.changed_at)
    .filter((v): v is string => Boolean(v))
    .sort((a, b) => new Date(b).getTime() - new Date(a).getTime())[0] ?? null
}

function isEmail(s: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s.trim())
}

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

// 打开某分组的倍率历史折线（用 relay_rate_changes 数据构造时间序列）。
async function openHistory(row: { monitor_id: number; site: string; group_name: string; current_rate: number }) {
  history.open = true
  history.loading = true
  history.site = row.site
  history.group = row.group_name
  history.points = []
  try {
    const res = await relayMonitorAPI.listChanges({ monitor_id: row.monitor_id, page: 1, page_size: 200 })
    const groupChanges = res.items
      .filter((c) => c.group_name === row.group_name)
      .sort((a, b) => new Date(a.detected_at).getTime() - new Date(b.detected_at).getTime())
    const pts: { t: string; rate: number }[] = []
    if (groupChanges.length) {
      pts.push({ t: groupChanges[0].detected_at, rate: groupChanges[0].old_rate })
      for (const c of groupChanges) pts.push({ t: c.detected_at, rate: c.new_rate })
    }
    if (!pts.length || pts[pts.length - 1].rate !== row.current_rate) {
      pts.push({ t: new Date().toISOString(), rate: row.current_rate })
    }
    history.points = pts
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.loadError')))
  } finally {
    history.loading = false
  }
}

const historyChartData = computed(() => {
  if (history.points.length < 2) return null
  return {
    labels: history.points.map((p) => formatTime(p.t)),
    datasets: [
      {
        label: history.group,
        data: history.points.map((p) => p.rate),
        borderColor: '#6366f1',
        backgroundColor: 'rgba(99,102,241,0.15)',
        fill: true,
        stepped: true,
        pointRadius: 3,
        pointBackgroundColor: '#6366f1',
      },
    ],
  }
})

const historyChartOptions = computed(() => {
  const dark = document.documentElement.classList.contains('dark')
  const text = dark ? '#e5e7eb' : '#374151'
  const grid = dark ? '#374151' : '#e5e7eb'
  return {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: { callbacks: { label: (c: { raw: unknown }) => formatRate(Number(c.raw)) } },
    },
    scales: {
      x: { grid: { color: grid }, ticks: { color: text, font: { size: 10 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 8 } },
      y: { grid: { color: grid }, ticks: { color: text, font: { size: 10 }, callback: (v: string | number) => formatRate(Number(v)) }, beginAtZero: true },
    },
  }
})

async function loadOverview() {
  overviewLoading.value = true
  try {
    overview.value = await relayMonitorAPI.overview(searchQuery.value.trim() || undefined)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.relayMonitor.loadError')))
  } finally {
    overviewLoading.value = false
  }
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
  await loadOverview()
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
    await loadOverview()
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
    await loadOverview()
    await loadChanges()
    await loadSummary()
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
  form.auth_mode = 'password'
  form.auth_account = ''
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
  // 有邮箱=账号密码模式；无邮箱但 sub2api=Token 模式
  form.auth_mode = m.system === 'sub2api' && !m.auth_account ? 'token' : 'password'
  form.auth_account = m.auth_account
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
      auth_account: effectiveAuthAccount() || undefined,
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
      auth_account: effectiveAuthAccount(),
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
  loadFavorites()
  loadCollapsed()
  loadOverview()
  loadChanges()
  loadSummary()
  loadMonitors()
})
</script>
