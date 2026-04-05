(function () {
  'use strict';

  const LS_TOKEN = 'stepup_student_token';
  const LS_API = 'stepup_api_base';

  /** 与部署约定一致时可不配置 meta：`?api=` / localStorage / meta 仍可覆盖。 */
  const PAGE_PORT_TO_API_PORT = { '7010': '7012', '7011': '7012' };
  /** 与根目录 `.env.example` 中 `BACKEND_PORT` 默认一致；非常规映射请用 `?api=` 或 meta。 */
  const DEFAULT_HOST_BACKEND_PORT = '8080';

  function apiBaseSameHostPort(portRaw) {
    const pr = String(portRaw || '')
      .trim()
      .replace(/^:/, '');
    if (!/^\d+$/.test(pr)) return '';
    return location.protocol + '//' + location.hostname + ':' + pr;
  }

  function apiBase() {
    const q = new URLSearchParams(location.search).get('api');
    if (q) {
      const qt = q.trim();
      if (/^:?(\d+)$/.test(qt)) {
        const u = apiBaseSameHostPort(qt);
        if (u) return u.replace(/\/$/, '');
      }
      return qt.replace(/\/$/, '');
    }
    const s = localStorage.getItem(LS_API);
    if (s) return s.replace(/\/$/, '');
    const meta = document.querySelector('meta[name="stepup-api-base"]');
    if (meta) {
      const mc = (meta.getAttribute('content') || '').trim();
      if (mc) return mc.replace(/\/$/, '');
    }
    const metaPort = document.querySelector('meta[name="stepup-api-port"]');
    if (metaPort) {
      const u = apiBaseSameHostPort(metaPort.getAttribute('content'));
      if (u) return u.replace(/\/$/, '');
    }
    const apiPortHint = PAGE_PORT_TO_API_PORT[location.port || ''];
    if (apiPortHint) {
      const u = apiBaseSameHostPort(apiPortHint);
      if (u) return u.replace(/\/$/, '');
    }
    const p = location.pathname || '';
    if (p.startsWith('/student')) return location.origin;
    const host = location.hostname;
    const port = location.port;
    const isLocal = host === 'localhost' || host === '127.0.0.1';
    if (isLocal && port === '3000') return 'http://localhost:8080';
    if (isLocal && port === '8080') return location.origin;
    if (!isLocal && (port === '3000' || port === '3001')) {
      return (location.protocol + '//' + host + ':' + DEFAULT_HOST_BACKEND_PORT).replace(/\/$/, '');
    }
    if (!isLocal) return location.origin;
    return 'http://localhost:8080';
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function formatBytes(n) {
    if (n == null || n === '' || Number(n) <= 0) return '—';
    const x = Number(n);
    if (x < 1024) return x + ' B';
    if (x < 1024 * 1024) return (x / 1024).toFixed(1) + ' KB';
    return (x / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function formatWhen(raw) {
    if (raw == null || raw === '') return '—';
    try {
      const d = new Date(raw);
      if (isNaN(d.getTime())) return String(raw);
      return d.toLocaleString('zh-CN', { hour12: false });
    } catch {
      return String(raw);
    }
  }

  /** Per-subject student features. Extend here or later move to GET /api/v1/catalog.features. */
  const PAPER_ANALYZE = {
    id: 'paper_analyze',
    title: '试卷 AI 分析',
    desc: '上传 PDF，或最多 10 张试卷图片，生成摘要、薄弱点与改进计划',
    available: true,
  };

  function subjectFeatures(subjectName) {
    const map = {
      语文: [
        PAPER_ANALYZE,
        {
          id: 'essay_outline',
          title: '作文提纲练习',
          desc: '立意、结构与论据提纲训练（即将开放）',
          available: false,
        },
      ],
      英语: [
        PAPER_ANALYZE,
        {
          id: 'vocab',
          title: '背单词',
          desc: '词书与复习计划（即将开放）',
          available: false,
        },
      ],
      数学: [
        PAPER_ANALYZE,
        {
          id: 'topic_drill',
          title: '题型专项',
          desc: '按知识点巩固（即将开放）',
          available: false,
        },
      ],
      物理: [
        PAPER_ANALYZE,
        {
          id: 'lab_think',
          title: '实验与探究',
          desc: '实验思路与数据处理（即将开放）',
          available: false,
        },
      ],
    };
    return map[subjectName] || [PAPER_ANALYZE];
  }

  /** 登录后主界面异步加载代数；`mount` 递增，用于丢弃过期的 Promise 回调，避免失败后反复 `mount` 造成死循环。 */
  let mainLoadGen = 0;

  const state = {
    token: localStorage.getItem(LS_TOKEN),
    authTab: 'login',
    flash: null,
    catalog: { subjects: [], stages: [] },
    session: null,
    papers: [],
    selectedPaperId: null,
    uploading: false,
    hubSubject: null,
    hubFeature: null,
  };

  async function api(path, opts = {}) {
    const url = apiBase() + path;
    const headers = Object.assign({}, opts.headers || {});
    if (state.token && !opts.skipAuth) headers['Authorization'] = 'Bearer ' + state.token;
    if (opts.jsonBody) headers['Content-Type'] = 'application/json';
    const init = { method: opts.method || 'GET', headers };
    if (opts.jsonBody) init.body = JSON.stringify(opts.jsonBody);
    if (opts.form) init.body = opts.form;
    const res = await fetch(url, init);
    const text = await res.text();
    let data = null;
    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      data = { raw: text };
    }
    if (!res.ok) {
      const err = new Error(data && data.code ? data.code : 'HTTP_' + res.status);
      err.status = res.status;
      err.data = data;
      throw err;
    }
    return data;
  }

  function failedFetchHint(msg) {
    return msg === 'Failed to fetch'
      ? '（请确认 API 已启动；若学生页与 API 不同源，后端 `CORS_ALLOWED_ORIGINS` 须包含本页 Origin，如 `http://<IP>:7010` 或 `:3000`，登录页可见「API」地址。）'
      : '';
  }

  function showPapersLoadError(root, e, gen) {
    const msg = (e.data && e.data.code) || e.message || String(e);
    const hint = failedFetchHint(msg);
    const pane = root.querySelector('#paperPane');
    if (!pane) return;
    pane.innerHTML = `<p class="muted">试卷列表加载失败：${escapeHtml(msg + hint)}</p><button type="button" class="btn" id="btnRetryPapers" style="margin-top:10px">重试</button>`;
    pane.querySelector('#btnRetryPapers').addEventListener('click', async () => {
      if (gen !== mainLoadGen) return;
      pane.innerHTML = '<p class="muted">加载中…</p>';
      try {
        await refreshPapers(root);
      } catch (e2) {
        if (gen !== mainLoadGen) return;
        showPapersLoadError(root, e2, gen);
      }
    });
  }

  function showShellLoadError(root, e) {
    const msg = (e.data && e.data.code) || e.message || String(e);
    const hint = failedFetchHint(msg);
    const line = root.querySelector('#sessionLine');
    if (line) line.textContent = '无法加载工作台';
    const hub = root.querySelector('#hubCard');
    if (hub) {
      hub.innerHTML = `<p class="muted">数据加载失败，请检查网络与后端服务。</p>
        <p class="flash err" style="margin-top:10px">${escapeHtml(msg + hint)}</p>
        <button type="button" class="btn" id="btnRetryShell" style="margin-top:12px">重试</button>`;
      hub.querySelector('#btnRetryShell').addEventListener('click', () => {
        mainLoadGen++;
        startMainShell(root, mainLoadGen);
      });
    }
    const pane = root.querySelector('#paperPane');
    if (pane) pane.innerHTML = '<p class="muted">试卷列表未加载。</p>';
  }

  function startMainShell(root, gen) {
    const hubEl = root.querySelector('#hubCard');
    if (hubEl) hubEl.innerHTML = '<p class="muted">加载中…</p>';
    const paperPane = root.querySelector('#paperPane');
    if (paperPane) paperPane.innerHTML = '<p class="muted">加载试卷列表…</p>';

    Promise.all([loadSession(root), loadCatalog()])
      .then(async () => {
        if (gen !== mainLoadGen) return;
        try {
          const hub = root.querySelector('#hubCard');
          if (!hub) return;
          hub.innerHTML = renderHub();
          bindHub(root, hub);
          if (state.hubSubject && state.hubFeature === 'paper_analyze') {
            const fb = hub.querySelector('#featureBody');
            if (fb) bindUpload(root, fb);
          }
          try {
            await refreshPapers(root);
          } catch (e) {
            if (gen !== mainLoadGen) return;
            showPapersLoadError(root, e, gen);
          }
        } catch (e) {
          if (gen !== mainLoadGen) return;
          showShellLoadError(root, e);
        }
      })
      .catch((e) => {
        if (gen !== mainLoadGen) return;
        showShellLoadError(root, e);
      });
  }

  function mount(root) {
    const flash = state.flash
      ? `<div class="flash ${state.flash.kind}">${escapeHtml(state.flash.msg)}</div>`
      : '';
    if (!state.token) {
      state.session = null;
      root.innerHTML = `
        <div class="wrap">
          <div class="card">
            <h1>StepUp 学生端</h1>
            <p class="muted">手机号或邮箱注册/登录。按科目使用试卷分析、作文提纲等功能（持续扩展）。</p>
            <p class="muted" style="margin-top:8px">API：<strong>${escapeHtml(apiBase())}</strong>
              （<code>?api=</code> 或 <code>localStorage.stepup_api_base</code>）</p>
            ${flash}
            <div class="tabs">
              <button type="button" id="tabReg" class="${state.authTab === 'reg' ? 'on' : ''}">注册</button>
              <button type="button" id="tabLog" class="${state.authTab === 'login' ? 'on' : ''}">登录</button>
            </div>
            <div id="authBody"></div>
          </div>
        </div>`;
      bindAuthTabs(root);
      renderAuthBody(root.querySelector('#authBody'));
      return;
    }
    mainLoadGen++;
    const gen = mainLoadGen;
    root.innerHTML = `
      <div class="wrap">
        <div class="card toolbar">
          <div>
            <h1 style="margin:0;font-size:20px">StepUp 学习工作台</h1>
            <p class="muted" style="margin:4px 0 0;font-size:13px" id="sessionLine">加载会话…</p>
          </div>
          <button type="button" class="btn secondary" id="btnOut">退出</button>
        </div>
        ${flash}
        <div class="card" id="hubCard"><p class="muted">加载中…</p></div>
        <div class="card">
          <h2>我的试卷</h2>
          <p class="muted">含各科目上传记录；点击一条查看 AI 摘要与改进计划。</p>
          <div id="paperPane"></div>
        </div>
      </div>`;
    bindMain(root);
    startMainShell(root, gen);
  }

  async function loadSession(root) {
    try {
      const d = await api('/api/v1/student/auth/me');
      state.session = d;
      const line = root.querySelector('#sessionLine');
      if (line && d.user) {
        const ex = d.expires_at ? formatWhen(d.expires_at) : '—';
        line.textContent = '当前账号：' + (d.user.identifier || '') + ' · 会话至 ' + ex;
      }
    } catch {
      const line = root.querySelector('#sessionLine');
      if (line) line.textContent = '无法获取会话信息';
    }
  }

  function bindAuthTabs(root) {
    root.querySelector('#tabReg').addEventListener('click', () => {
      state.authTab = 'reg';
      state.flash = null;
      mount(document.getElementById('app'));
    });
    root.querySelector('#tabLog').addEventListener('click', () => {
      state.authTab = 'login';
      state.flash = null;
      mount(document.getElementById('app'));
    });
  }

  function renderAuthBody(host) {
    if (state.authTab === 'login') {
      host.innerHTML = `
        <div class="form-grid">
          <div><label>手机号或邮箱</label><input id="lid" autocomplete="username" /></div>
          <div><label>密码</label><input id="lpw" type="password" autocomplete="current-password" /></div>
          <button type="button" class="btn" id="bLogin">登录</button>
        </div>`;
      const go = async () => {
        try {
          const d = await api('/api/v1/student/auth/login', {
            method: 'POST',
            jsonBody: {
              identifier: host.querySelector('#lid').value.trim(),
              password: host.querySelector('#lpw').value,
            },
            skipAuth: true,
          });
          state.token = d.token;
          localStorage.setItem(LS_TOKEN, d.token);
          state.flash = null;
          mount(document.getElementById('app'));
        } catch (e) {
          state.flash = { kind: 'err', msg: '登录失败：' + (e.data && e.data.code ? e.data.code : e.message) };
          mount(document.getElementById('app'));
        }
      };
      host.querySelector('#bLogin').addEventListener('click', go);
      host.querySelector('#lpw').addEventListener('keydown', (ev) => {
        if (ev.key === 'Enter') go();
      });
      return;
    }
    host.innerHTML = `
      <p class="muted">① 发送验证码 → ② 校验 → ③ 设置密码（开发环境响应中会包含验证码）。</p>
      <div class="row">
        <div><label>手机号或邮箱</label><input id="rid" /></div>
        <div><label>&nbsp;</label><button type="button" class="btn secondary" id="bSend" style="margin-top:0;width:100%">发送验证码</button></div>
      </div>
      <div id="codeInfo" class="flash info" style="display:none"></div>
      <div class="row">
        <div><label>验证码</label><input id="rcode" /></div>
        <div><label>&nbsp;</label><button type="button" class="btn secondary" id="bVer" style="margin-top:0;width:100%">校验验证码</button></div>
      </div>
      <div class="row">
        <div><label>设置密码（至少 8 位建议）</label><input id="rpw" type="password" /></div>
        <div><label>&nbsp;</label><button type="button" class="btn" id="bSet" style="margin-top:0;width:100%">设置密码并创建账号</button></div>
      </div>`;
    host.querySelector('#bSend').addEventListener('click', async () => {
      const id = host.querySelector('#rid').value.trim();
      try {
        const d = await api('/api/v1/student/auth/send-code', {
          method: 'POST',
          jsonBody: { identifier: id },
          skipAuth: true,
        });
        const el = host.querySelector('#codeInfo');
        el.style.display = 'block';
        el.textContent = '验证码（仅开发可见）：' + (d.code || '—');
      } catch (e) {
        alert('发送失败：' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
    host.querySelector('#bVer').addEventListener('click', async () => {
      try {
        await api('/api/v1/student/auth/verify-code', {
          method: 'POST',
          jsonBody: {
            identifier: host.querySelector('#rid').value.trim(),
            code: host.querySelector('#rcode').value.trim(),
          },
          skipAuth: true,
        });
        host.querySelector('#codeInfo').style.display = 'block';
        host.querySelector('#codeInfo').textContent = '验证码已校验通过，请设置密码。';
      } catch (e) {
        alert('校验失败：' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
    host.querySelector('#bSet').addEventListener('click', async () => {
      try {
        await api('/api/v1/student/auth/set-password', {
          method: 'POST',
          jsonBody: {
            identifier: host.querySelector('#rid').value.trim(),
            password: host.querySelector('#rpw').value,
          },
          skipAuth: true,
        });
        state.flash = { kind: 'ok', msg: '注册成功，请切换到「登录」。' };
        state.authTab = 'login';
        mount(document.getElementById('app'));
      } catch (e) {
        alert('设置密码失败：' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  async function loadCatalog() {
    try {
      const d = await api('/api/v1/catalog', { skipAuth: true });
      state.catalog.subjects = d.subjects || [];
      state.catalog.stages = d.stages || [];
    } catch {
      state.catalog.subjects = [];
      state.catalog.stages = [];
    }
  }

  function renderHub() {
    if (!state.hubSubject) {
      const subs = state.catalog.subjects || [];
      const cards = subs.length
        ? subs
            .map(
              (s) =>
                `<button type="button" class="subject-card" data-sub="${escapeHtml(s.name)}"><span class="sc-name">${escapeHtml(
                  s.name
                )}</span><span class="sc-hint">查看功能</span></button>`
            )
            .join('')
        : '<p class="muted">暂无科目。请稍后重试或联系管理员配置 catalog。</p>';
      return `
        <h2>选择科目</h2>
        <p class="muted">先选科目，再进入具体功能。试卷分析支持 <strong>1 个 PDF</strong> 或 <strong>最多 10 张图片</strong>。</p>
        <div class="subject-grid">${cards}</div>`;
    }
    const feats = subjectFeatures(state.hubSubject);
    const featHtml = feats
      .map((f) => {
        const on = state.hubFeature === f.id ? ' on' : '';
        const soon = f.available ? '' : ' soon';
        return `<button type="button" class="feature-card${on}${soon}" data-fid="${escapeHtml(f.id)}" data-available="${
          f.available ? '1' : '0'
        }"><span class="fc-title">${escapeHtml(f.title)}</span><span class="fc-desc">${escapeHtml(f.desc)}</span></button>`;
      })
      .join('');
    let panel = '';
    if (state.hubFeature === 'paper_analyze') {
      panel = `<div class="feature-panel" id="featureBody">${renderUploadForm()}</div>`;
    }
    return `
      <div class="hub-nav">
        <button type="button" class="btn-link" id="btnBackSubjects">← 全部科目</button>
        <h2 style="margin:8px 0 0">${escapeHtml(state.hubSubject)}</h2>
        <p class="muted" style="margin:4px 0 0">选择学习功能</p>
      </div>
      <div class="feature-grid">${featHtml}</div>
      ${panel}`;
  }

  function bindHub(root, hubEl) {
    hubEl.querySelector('#btnBackSubjects')?.addEventListener('click', () => {
      state.hubSubject = null;
      state.hubFeature = null;
      mount(document.getElementById('app'));
    });
    hubEl.querySelectorAll('.subject-card').forEach((b) => {
      b.addEventListener('click', () => {
        state.hubSubject = b.getAttribute('data-sub');
        state.hubFeature = null;
        mount(document.getElementById('app'));
      });
    });
    hubEl.querySelectorAll('.feature-card').forEach((b) => {
      b.addEventListener('click', () => {
        if (b.getAttribute('data-available') === '0') {
          alert('该功能即将开放，敬请期待。');
          return;
        }
        state.hubFeature = b.getAttribute('data-fid');
        mount(document.getElementById('app'));
      });
    });
  }

  function collectUploadFiles(fileList) {
    const fs = Array.from(fileList || []);
    if (fs.length === 0) return null;
    if (fs.length > 10) {
      alert('最多选择 10 个文件。');
      return null;
    }
    const pdfs = fs.filter((f) => f.name.toLowerCase().endsWith('.pdf'));
    if (pdfs.length > 1) {
      alert('PDF 仅支持单个文件上传。');
      return null;
    }
    if (pdfs.length === 1 && fs.length > 1) {
      alert('上传 PDF 时不要同时选择其他文件。多图请只选图片。');
      return null;
    }
    return fs;
  }

  function optStages() {
    const xs = state.catalog.stages || [];
    if (!xs.length) return '<option value="高中">高中</option>';
    return xs.map((s) => `<option value="${escapeHtml(s.name)}">${escapeHtml(s.name)}</option>`).join('');
  }

  function renderUploadForm() {
    const sub = state.hubSubject ? escapeHtml(state.hubSubject) : '—';
    return `
      <h3 style="margin:0 0 8px">试卷 AI 分析</h3>
      <p class="muted" style="margin:0 0 12px">当前科目：<strong>${sub}</strong>。可选 <strong>1 个 PDF</strong>，或 <strong>最多 10 张图片</strong>（多图将一并送给模型识图分析）。单张图不超过 25MB。</p>
      <div class="row" style="grid-template-columns:1fr">
        <div><label>阶段</label><select id="stg">${optStages()}</select></div>
      </div>
      <div class="drop">
        <input type="file" id="file" multiple accept=".pdf,application/pdf,image/*" />
        <p class="muted" id="fileMeta" style="margin:8px 0 0">未选择文件</p>
      </div>
      <button type="button" class="btn" id="bUp">提交并开始分析</button>`;
  }

  function bindUpload(root, container) {
    const fileInput = container.querySelector('#file');
    const meta = container.querySelector('#fileMeta');
    const btn = container.querySelector('#bUp');
    if (!fileInput || !btn) return;
    fileInput.addEventListener('change', () => {
      const fs = fileInput.files;
      if (!fs || !fs.length) {
        meta.textContent = '未选择文件';
        return;
      }
      const names = Array.from(fs)
        .map((f) => f.name)
        .join('、');
      let total = 0;
      Array.from(fs).forEach((f) => {
        total += f.size;
      });
      meta.textContent = `已选 ${fs.length} 个文件 · 合计 ${formatBytes(total)} · ${names.length > 120 ? names.slice(0, 120) + '…' : names}`;
    });
    btn.addEventListener('click', async () => {
      if (!state.hubSubject) {
        alert('请先从科目入口进入。');
        return;
      }
      const stg = container.querySelector('#stg').value;
      const fs = collectUploadFiles(fileInput.files);
      if (!fs) return;
      if (state.uploading) return;
      state.uploading = true;
      btn.disabled = true;
      btn.textContent = '上传并分析中…';
      const fd = new FormData();
      fd.append('subject', state.hubSubject);
      fd.append('stage', stg);
      fs.forEach((f) => fd.append('files', f, f.name));
      try {
        const res = await api('/api/v1/student/papers', { method: 'POST', form: fd });
        const newId = res.paper && res.paper.id ? Number(res.paper.id) : null;
        state.flash = { kind: 'ok', msg: '上传成功（试卷 #' + (newId || '?') + '），已生成分析。' };
        if (newId) state.selectedPaperId = newId;
        state.uploading = false;
        mount(document.getElementById('app'));
      } catch (e) {
        state.uploading = false;
        btn.disabled = false;
        btn.textContent = '提交并开始分析';
        alert('上传失败：' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function bindMain(root) {
    root.querySelector('#btnOut').addEventListener('click', async () => {
      try {
        await api('/api/v1/student/auth/logout', { method: 'POST' });
      } catch (_) {}
      state.token = null;
      localStorage.removeItem(LS_TOKEN);
      state.selectedPaperId = null;
      state.session = null;
      state.hubSubject = null;
      state.hubFeature = null;
      mount(document.getElementById('app'));
    });
  }

  async function refreshPapers(root) {
    const d = await api('/api/v1/student/papers');
    state.papers = d.items || [];
    const pane = root.querySelector('#paperPane');
    if (!pane) return;
    const rows = state.papers
      .map(
        (p) =>
          `<div class="paper-item ${state.selectedPaperId === p.id ? 'on' : ''}" data-id="${p.id}">
          <div><strong>#${p.id}</strong> ${escapeHtml(p.subject)} · ${escapeHtml(p.stage)}<br/>
          <span class="muted" style="font-size:12px">${escapeHtml(p.file_name)} · ${formatBytes(
            p.file_size
          )} · ${formatWhen(p.created_at)}</span></div>
          <span class="muted">›</span>
        </div>`
      )
      .join('');
    pane.innerHTML = `<div class="paper-list">${rows || '<p class="muted">暂无试卷，请先上传。</p>'}</div><div id="detail"></div>`;
    pane.querySelectorAll('.paper-item').forEach((el) => {
      el.addEventListener('click', async () => {
        state.selectedPaperId = Number(el.getAttribute('data-id'));
        await showDetail(root);
      });
    });
    if (state.selectedPaperId) await showDetail(root);
  }

  async function showDetail(root) {
    const pane = root.querySelector('#detail');
    if (!pane || !state.selectedPaperId) return;
    pane.innerHTML = '<p class="muted">加载详情…</p>';
    try {
      const [an, pl] = await Promise.all([
        api('/api/v1/student/papers/' + state.selectedPaperId + '/analysis'),
        api('/api/v1/student/papers/' + state.selectedPaperId + '/plan'),
      ]);
      const a = an.analysis;
      const plan = (pl && pl.plan) || [];
      const statusLabel =
        a.status === 'completed' ? '已完成' : a.status === 'failed' ? '失败' : a.status === 'processing' ? '处理中' : a.status;
      pane.innerHTML = `
        <div class="detail">
          <h3 style="margin-top:0">试卷 #${state.selectedPaperId}</h3>
          <p class="meta-line"><span class="lbl">分析状态</span> ${escapeHtml(statusLabel)}</p>
          <h4>模型信息</h4>
          <pre class="block">${escapeHtml(JSON.stringify(a.ai_model_snapshot || {}, null, 2))}</pre>
          <h4>摘要</h4>
          <pre class="block">${escapeHtml(a.summary || '')}</pre>
          <h4>薄弱点</h4>
          <ul class="bullets">${(a.weak_points || []).map((x) => '<li>' + escapeHtml(x) + '</li>').join('') || '<li class="muted">—</li>'}</ul>
          <h4>改进计划</h4>
          <ol class="steps">${plan.map((x) => '<li>' + escapeHtml(x) + '</li>').join('') || '<li class="muted">—</li>'}</ol>
        </div>`;
    } catch (e) {
      pane.innerHTML =
        '<p class="muted">无法加载详情：' + escapeHtml(e.data && e.data.code ? e.data.code : e.message) + '</p>';
    }
    refreshPapersListOnly(root);
  }

  function refreshPapersListOnly(root) {
    const list = root.querySelector('#paperPane .paper-list');
    if (!list) return;
    list.querySelectorAll('.paper-item').forEach((el) => {
      const id = Number(el.getAttribute('data-id'));
      el.classList.toggle('on', id === state.selectedPaperId);
    });
  }

  function boot() {
    mount(document.getElementById('app'));
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', boot);
  else boot();
})();
