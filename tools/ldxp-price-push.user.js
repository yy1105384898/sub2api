// ==UserScript==
// @name         链动小铺价格推送 (sub2api card-monitor)
// @namespace    sub2api-card-monitor
// @version      1.0
// @description  在浏览器内抓取链动小铺货源(天然过阿里云 acw_sc__v2 反爬)并推送到 sub2api 发卡平台监控。
// @match        https://pay.ldxp.cn/*
// @match        https://www.ldxp.cn/*
// @grant        GM_xmlhttpRequest
// @connect      *
// @run-at       document-idle
// ==/UserScript==
//
// 说明：管理后台「发卡平台监控 → 监控平台 → 推送脚本」会生成已填好密钥的版本，
// 直接复制/下载安装即可。本文件是版本化的参考模板，使用前请替换下面两个常量：
//   INGEST_URL  你的 sub2api 推送地址，如 https://yysubapi.yangyangnj.top/api/v1/card-ingest
//   KEY         该监控的 ingest_key（后台「推送脚本」里复制）
// 并把 @connect 改成你的 sub2api 域名。
(function () {
  'use strict';
  var INGEST_URL = '__INGEST_URL__';
  var KEY = '__INGEST_KEY__';
  var PAGES = 5;                       // 每轮抓取页数
  var INTERVAL_MS = 5 * 60 * 1000;     // 抓取间隔
  var API = '/merchantApi/MyParent/searchGoodsList';

  function getToken() {
    var keys = ['auth-token', 'Merchant-Token', 'merchant-token', 'token', 'Authorization'];
    for (var i = 0; i < keys.length; i++) {
      var v = localStorage.getItem(keys[i]);
      if (v) {
        try { var p = JSON.parse(v); return p.value || p.token || p.access_token || v; }
        catch (e) { return v; }
      }
    }
    return '';
  }

  async function fetchPage(token, page) {
    var resp = await fetch(API, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json;charset=UTF-8',
        'Accept': 'application/json, text/plain, */*',
        'Merchant-Token': token
      },
      body: JSON.stringify({ current: page, pageSize: 50, name: '', goods_type: '', keywords: '' })
    });
    var data = await resp.json();
    if (data.code !== 1) throw new Error(data.msg || data.message || ('code ' + data.code));
    return (data.data && data.data.list) || [];
  }

  function push(products) {
    return new Promise(function (resolve, reject) {
      GM_xmlhttpRequest({
        method: 'POST',
        url: INGEST_URL,
        headers: { 'Content-Type': 'application/json' },
        data: JSON.stringify({ key: KEY, products: products }),
        onload: function (r) {
          (r.status >= 200 && r.status < 300)
            ? resolve(r)
            : reject(new Error('ingest HTTP ' + r.status + ' ' + r.responseText));
        },
        onerror: function () { reject(new Error('ingest network error')); }
      });
    });
  }

  async function run() {
    try {
      var token = getToken();
      if (!token) { console.warn('[ldxp-push] 未登录链动后台，跳过'); return; }
      var all = [];
      for (var p = 1; p <= PAGES; p++) {
        var list = await fetchPage(token, p);
        all = all.concat(list);
        if (list.length < 50) break;
      }
      if (all.length) { await push(all); console.log('[ldxp-push] 已推送 ' + all.length + ' 个商品'); }
    } catch (e) {
      console.error('[ldxp-push] 失败', e);
    }
  }

  setTimeout(run, 4000);
  setInterval(run, INTERVAL_MS);
})();
