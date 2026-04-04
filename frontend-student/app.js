(function () {
  'use strict';

  const LS_TOKEN = 'stepup_student_token';
  const LS_API = 'stepup_api_base';

  function apiBase() {
    const q = new URLSearchParams(location.search).get('api');
    if (q) return q.replace(/\/$/, '');
    const s = localStorage.getItem(LS_API);
    if (s) return s.replace(/\/$/, '');
    const p = location.pathname || '';
    if (p.startsWith('/student')) return location.origin;
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

  const state = {
    token: localStorage.getItem(LS_TOKEN),
    authTab: 'login',
    flash: null,
    catalog: { subjects: [], stages: [] },
    session: null,
    papers: [],
    selectedPaperId: null,
    uploading: false,
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
            <p class="muted">手机号或邮箱注册/登录，上传试卷查看 AI 分析与改进计划（第一阶段）。</p>
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
        <div class="card" id="uploadCard"><p class="muted">加载中…</p></div>
        <div class="card">
          <h2>我的试卷</h2>
          <p class="muted">点击一条查看摘要、薄弱点与改进计划（数据已保存至服务端）。</p>
          <div id="paperPane"></div>
        </div>
      </div>`;
    bindMain(root);
    Promise.all([loadSession(root), loadCatalog()])
      .then(() => {
        root.querySelector('#uploadCard').innerHTML = renderUploadForm();
        bindUpload(root);
        return refreshPapers(root);
      })
      .catch((e) => {
        state.flash = { kind: 'err', msg: (e.data && e.data.code) || e.message || String(e) };
        mount(root);
      });
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

  function optSubjects() {
    const xs = state.catalog.subjects || [];
    if (!xs.length) return '<option value="物理">物理</option><option value="语文">语文</option>';
    return xs.map((s) => `<option value="${escapeHtml(s.name)}">${escapeHtml(s.name)}</option>`).join('');
  }

  function optStages() {
    const xs = state.catalog.stages || [];
    if (!xs.length) return '<option value="高中">高中</option>';
    return xs.map((s) => `<option value="${escapeHtml(s.name)}">${escapeHtml(s.name)}</option>`).join('');
  }

  function renderUploadForm() {
    return `
      <h2>上传试卷</h2>
      <p class="muted">选择科目与阶段，上传 <strong>PDF</strong> 或 <strong>图片</strong>。分析由当前后台配置的 AI 模型执行，结果写入数据库。</p>
      <div class="row">
        <div><label>科目</label><select id="sub">${optSubjects()}</select></div>
        <div><label>阶段</label><select id="stg">${optStages()}</select></div>
      </div>
      <div class="drop">
        <input type="file" id="file" accept=".pdf,application/pdf,image/*" />
        <p class="muted" id="fileMeta" style="margin:8px 0 0">未选择文件</p>
      </div>
      <button type="button" class="btn" id="bUp">提交并开始分析</button>`;
  }

  function bindUpload(root) {
    const fileInput = root.querySelector('#file');
    const meta = root.querySelector('#fileMeta');
    const btn = root.querySelector('#bUp');
    fileInput.addEventListener('change', () => {
      const f = fileInput.files[0];
      if (!f) {
        meta.textContent = '未选择文件';
        return;
      }
      meta.textContent = f.name + ' · ' + formatBytes(f.size);
    });
    btn.addEventListener('click', async () => {
      const sub = root.querySelector('#sub').value;
      const stg = root.querySelector('#stg').value;
      const f = fileInput.files[0];
      if (!f) {
        alert('请选择文件');
        return;
      }
      if (state.uploading) return;
      state.uploading = true;
      btn.disabled = true;
      btn.textContent = '上传并分析中…';
      const fd = new FormData();
      fd.append('subject', sub);
      fd.append('stage', stg);
      fd.append('file', f, f.name);
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
