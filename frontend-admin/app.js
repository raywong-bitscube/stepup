(function () {
  'use strict';

  const LS_TOKEN = 'stepup_admin_token';
  const LS_API = 'stepup_api_base';

  /** 与部署约定一致时可不配置 meta：`?api=` / localStorage / meta / 登录框端口 仍可覆盖。 */
  const PAGE_PORT_TO_API_PORT = { '7010': '7012', '7011': '7012' };

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
    if (p.startsWith('/admin')) return location.origin;
    const host = location.hostname;
    const port = location.port;
    const isLocal = host === 'localhost' || host === '127.0.0.1';
    if (isLocal && port === '3001') return 'http://localhost:8080';
    if (isLocal && port === '8080') return location.origin;
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

  function aiLogErrLine(e) {
    const p = String(e.error_phase || '').trim();
    const m = String(e.error_message || '').trim();
    if (!p && !m) return '—';
    const line = p && m ? p + ' · ' + m : p || m;
    return escapeHtml(line.length > 120 ? line.slice(0, 120) + '…' : line);
  }

  function aiLogMetaJSON(e) {
    try {
      return JSON.stringify({ request_meta: e.request_meta, response_meta: e.response_meta }, null, 2);
    } catch {
      return '';
    }
  }

  /** Full-width detail table for one log row (placed in colspan row below main row). */
  function renderAICallDetailPanel(e) {
    const req = String(e.request_body || '').trim();
    const res = String(e.response_body || '').trim();
    const meta = String(aiLogMetaJSON(e) || '').trim();
    const cell = (label, text) => {
      const t = String(text || '').trim();
      const inner = t ? escapeHtml(t) : '<span class="muted">—</span>';
      return `<tr><th scope="row">${escapeHtml(label)}</th><td class="ai-detail-td"><pre class="ai-detail-pre">${inner}</pre></td></tr>`;
    };
    return `<table class="data ai-log-detail-inner" role="presentation"><tbody>
      ${cell('请求 JSON', req)}
      ${cell('响应原文', res)}
      ${cell('结构化 Meta', meta)}
    </tbody></table>`;
  }

  const state = {
    token: localStorage.getItem(LS_TOKEN),
    view: 'dashboard',
    flash: null,
    catalog: { subjects: [], stages: [] },
    students: [],
    subjects: [],
    stages: [],
    models: [],
    prompts: [],
    audits: [],
    aiLogs: [],
    aiLogFilters: {
      limit: 50,
      offset: 0,
      ai_model_id: '',
      action: '',
      result_status: '',
      adapter_kind: '',
      from: '',
      to: '',
    },
  };

  async function api(path, opts = {}) {
    const url = apiBase() + path;
    const headers = Object.assign({}, opts.headers || {});
    if (state.token) headers['Authorization'] = 'Bearer ' + state.token;
    if (opts.jsonBody) {
      headers['Content-Type'] = 'application/json';
    }
    const init = {
      method: opts.method || 'GET',
      headers,
    };
    if (opts.jsonBody) init.body = JSON.stringify(opts.jsonBody);
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

  function setFlash(kind, msg) {
    state.flash = msg ? { kind, msg } : null;
  }

  async function loadCatalog() {
    try {
      const d = await api('/api/v1/catalog');
      state.catalog.subjects = d.subjects || [];
      state.catalog.stages = d.stages || [];
    } catch {
      state.catalog.subjects = [];
      state.catalog.stages = [];
    }
  }

  function mount(root) {
    if (!state.token) {
      root.innerHTML = renderLogin();
      bindLogin(root);
      return;
    }
    root.innerHTML = renderAppShell();
    bindApp(root);
    refreshView(root).catch((e) => setFlash('err', e.message || String(e)));
  }

  function renderLogin() {
    const flash = state.flash
      ? `<div class="flash ${state.flash.kind}">${escapeHtml(state.flash.msg)}</div>`
      : '';
    return `
      <div class="wrap" style="max-width:420px;margin-top:40px">
        <div class="card">
          <h1>StepUp 管理后台</h1>
          <p class="muted">登录后可管理学生、科目、阶段、AI 模型、Prompt 与审计日志。</p>
          ${flash}
          <div class="form-grid" style="margin-top:12px">
            <div>
              <label>API 根地址（可选）</label>
              <input type="text" id="apiBase" placeholder="留空则用自动推断" value="${escapeHtml(apiBase())}" />
              <p class="muted" style="font-size:12px;margin:4px 0 0">默认已按当前访问地址选好 API。一般<strong>不用改</strong>，直接输密码登录即可。仅当 API 不在默认端口时，可改为完整地址（如 <code>http://主机:7012</code>）或只填端口 <code>7012</code>。清空并登录表示去掉已保存的地址、重新用自动推断。</p>
            </div>
            <div>
              <label>用户名</label>
              <input type="text" id="user" value="admin" autocomplete="username" />
            </div>
            <div>
              <label>密码</label>
              <input type="password" id="pass" autocomplete="current-password" />
            </div>
            <button class="btn" type="button" id="btnLogin">登录</button>
          </div>
        </div>
      </div>`;
  }

  function bindLogin(root) {
    const b = root.querySelector('#btnLogin');
    b.addEventListener('click', async () => {
      const u = root.querySelector('#user').value.trim();
      const p = root.querySelector('#pass').value;
      let ab = root.querySelector('#apiBase').value.trim();
      if (/^:?(\d+)$/.test(ab)) {
        const built = apiBaseSameHostPort(ab);
        if (built) ab = built.replace(/\/$/, '');
      } else if (ab) {
        ab = ab.replace(/\/$/, '');
      }
      if (ab) localStorage.setItem(LS_API, ab);
      else localStorage.removeItem(LS_API);
      try {
        const data = await api('/api/v1/admin/auth/login', {
          method: 'POST',
          jsonBody: { username: u, password: p },
        });
        state.token = data.token;
        localStorage.setItem(LS_TOKEN, data.token);
        setFlash(null);
        mount(document.getElementById('app'));
      } catch (e) {
        setFlash('err', '登录失败：' + (e.data && e.data.code ? e.data.code : e.message));
        mount(document.getElementById('app'));
      }
    });
  }

  function renderAppShell() {
    const flash = state.flash
      ? `<div class="flash ${state.flash.kind}">${escapeHtml(state.flash.msg)}</div>`
      : '';
    const nav = [
      ['dashboard', '仪表盘'],
      ['students', '学生'],
      ['subjects', '科目'],
      ['stages', '阶段'],
      ['models', 'AI 模型'],
      ['prompts', 'Prompt'],
      ['audit', '审计日志'],
      ['ai_logs', 'AI 调用日志'],
    ];
    const menu = nav
      .map(
        ([k, lab]) =>
          `<button type="button" data-view="${k}" class="${state.view === k ? 'active' : ''}">${escapeHtml(
            lab
          )}</button>`
      )
      .join('');
    return `
      <div class="wrap">
        <div class="card toolbar">
          <div>
            <strong>StepUp 管理后台</strong>
            <span class="muted" style="margin-left:8px">API: ${escapeHtml(apiBase())}</span>
          </div>
          <div class="row" style="margin-top:0">
            <button type="button" class="btn secondary small" id="btnLogout">退出</button>
          </div>
        </div>
        ${flash}
        <div class="layout">
          <aside class="card menu" id="sideMenu">${menu}</aside>
          <section class="card" id="mainPane"><p class="muted">加载中…</p></section>
        </div>
      </div>`;
  }

  function bindApp(root) {
    root.querySelector('#btnLogout').addEventListener('click', async () => {
      try {
        await api('/api/v1/admin/auth/logout', { method: 'POST' });
      } catch (_) {}
      state.token = null;
      localStorage.removeItem(LS_TOKEN);
      mount(document.getElementById('app'));
    });
    root.querySelectorAll('#sideMenu button').forEach((btn) => {
      btn.addEventListener('click', () => {
        state.view = btn.getAttribute('data-view');
        state.flash = null;
        mount(document.getElementById('app'));
      });
    });
  }

  async function refreshView(root) {
    const pane = root.querySelector('#mainPane');
    if (!pane) return;
    await loadCatalog();
    try {
      if (state.view === 'dashboard') {
        const [st, sub] = await Promise.all([
          api('/api/v1/admin/students'),
          api('/api/v1/admin/subjects'),
        ]);
        state.students = st.items || [];
        state.subjects = sub.items || [];
        pane.innerHTML = renderDashboard();
        return;
      }
      if (state.view === 'students') {
        const st = await api('/api/v1/admin/students');
        state.students = st.items || [];
        pane.innerHTML = renderStudents();
        bindStudents(pane);
        return;
      }
      if (state.view === 'subjects') {
        const d = await api('/api/v1/admin/subjects');
        state.subjects = d.items || [];
        pane.innerHTML = renderSubjects();
        bindSubjects(pane);
        return;
      }
      if (state.view === 'stages') {
        const d = await api('/api/v1/admin/stages');
        state.stages = d.items || [];
        pane.innerHTML = renderStages();
        bindStages(pane);
        return;
      }
      if (state.view === 'models') {
        const d = await api('/api/v1/admin/ai-models');
        state.models = d.items || [];
        pane.innerHTML = renderModels();
        bindModels(pane);
        return;
      }
      if (state.view === 'prompts') {
        const d = await api('/api/v1/admin/prompts');
        state.prompts = d.items || [];
        pane.innerHTML = renderPrompts();
        bindPrompts(pane);
        return;
      }
      if (state.view === 'audit') {
        const d = await api('/api/v1/admin/audit-logs?limit=200');
        state.audits = d.items || [];
        pane.innerHTML = renderAudit();
        return;
      }
      if (state.view === 'ai_logs') {
        const q = new URLSearchParams();
        const f = state.aiLogFilters;
        q.set('limit', String(f.limit || 50));
        q.set('offset', String(f.offset || 0));
        if (f.ai_model_id) q.set('ai_model_id', f.ai_model_id);
        if (f.action) q.set('action', f.action);
        if (f.result_status) q.set('result_status', f.result_status);
        if (f.adapter_kind) q.set('adapter_kind', f.adapter_kind);
        if (f.from) q.set('from', f.from);
        if (f.to) q.set('to', f.to);
        const d = await api('/api/v1/admin/ai-call-logs?' + q.toString());
        state.aiLogs = d.items || [];
        pane.innerHTML = renderAICallLogs();
        bindAICallLogs(pane);
      }
    } catch (e) {
      pane.innerHTML = `<p class="muted">加载失败：${escapeHtml(e.data && e.data.code ? e.data.code : e.message)}</p>`;
    }
  }

  function renderDashboard() {
    return `
      <h2>仪表盘</h2>
      <p class="muted">快速总览（数据来自接口实时统计）。</p>
      <div class="kpis">
        <div class="kpi"><div class="label">学生数</div><div class="value">${state.students.length}</div></div>
        <div class="kpi"><div class="label">科目数</div><div class="value">${state.subjects.length}</div></div>
        <div class="kpi"><div class="label">当前视图</div><div class="value" style="font-size:16px">v0.1</div></div>
      </div>`;
  }

  function stageOptions(sel) {
    const xs = state.catalog.stages || [];
    if (!xs.length) {
      const name = '高中';
      return `<option value="${name}" ${sel === name ? 'selected' : ''}>${name}</option>`;
    }
    return xs
      .map((s) => `<option value="${escapeHtml(s.name)}" ${s.name === sel ? 'selected' : ''}>${escapeHtml(s.name)}</option>`)
      .join('');
  }

  function renderStudents() {
    const rows = state.students
      .map((s) => {
        const phone = s.phone || '—';
        const email = s.email || '—';
        return `<tr>
          <td>${s.id}</td><td>${escapeHtml(String(phone))}</td><td>${escapeHtml(String(email))}</td>
          <td>${escapeHtml(s.name)}</td><td>${escapeHtml(s.stage)}</td>
          <td>${s.status === 1 ? '<span class="badge on">active</span>' : '<span class="badge off">off</span>'}</td>
          <td>
            <button type="button" class="btn small" data-act="edit" data-id="${s.id}">编辑</button>
            <button type="button" class="btn secondary small" data-act="papers" data-id="${s.id}">试卷</button>
          </td>
        </tr>`;
      })
      .join('');
    return `
      <div class="toolbar">
        <h2 style="margin:0">学生</h2>
        <button type="button" class="btn" id="btnAddStudent">新建学生</button>
      </div>
      <p class="muted">创建后学生可使用手机号/邮箱登录学生端。</p>
      <table class="data"><thead><tr><th>ID</th><th>手机</th><th>邮箱</th><th>姓名</th><th>阶段</th><th>状态</th><th></th></tr></thead>
      <tbody>${rows || '<tr><td colspan="7">暂无数据</td></tr>'}</tbody></table>
      <div id="modalRoot"></div>`;
  }

  function bindStudents(pane) {
    pane.querySelector('#btnAddStudent').addEventListener('click', () => openStudentModal(null));
    pane.querySelectorAll('button[data-act=edit]').forEach((b) => {
      b.addEventListener('click', () => {
        const id = Number(b.getAttribute('data-id'));
        const st = state.students.find((x) => x.id === id);
        openStudentModal(st);
      });
    });
    pane.querySelectorAll('button[data-act=papers]').forEach((b) => {
      b.addEventListener('click', () => openPapersModal(Number(b.getAttribute('data-id'))));
    });
  }

  function openStudentModal(existing) {
    const root = document.getElementById('modalRoot');
    const isEdit = !!existing;
    root.innerHTML = `
      <div class="modal-backdrop" id="mdb">
        <div class="modal">
          <h3>${isEdit ? '编辑学生 #' + existing.id : '新建学生'}</h3>
          <div class="form-grid">
            ${isEdit ? '' : `<div><label>手机号或邮箱（登录标识）</label><input id="m_idf" /></div>
            <div><label>初始密码</label><input id="m_pw" type="password" /></div>`}
            <div><label>姓名</label><input id="m_name" value="${existing ? escapeHtml(existing.name) : ''}" /></div>
            <div><label>阶段</label><select id="m_stage">${stageOptions(existing ? existing.stage : '')}</select></div>
            ${
              isEdit
                ? `<div><label>状态 1=启用 0=停用</label><input id="m_stat" type="number" value="${existing.status}" /></div>
            <div><label>重置密码（可选）</label><input id="m_newpw" type="password" placeholder="留空不改" /></div>`
                : ''
            }
          </div>
          <div class="row">
            <button type="button" class="btn" id="m_ok">${isEdit ? '保存' : '创建'}</button>
            <button type="button" class="btn secondary" id="m_cancel">取消</button>
          </div>
        </div>
      </div>`;
    const close = () => {
      root.innerHTML = '';
    };
    root.querySelector('#m_cancel').addEventListener('click', close);
    root.querySelector('#mdb').addEventListener('click', (e) => {
      if (e.target.id === 'mdb') close();
    });
    root.querySelector('#m_ok').addEventListener('click', async () => {
      try {
        if (isEdit) {
          const body = {
            name: root.querySelector('#m_name').value.trim(),
            stage: root.querySelector('#m_stage').value.trim(),
            status: Number(root.querySelector('#m_stat').value),
          };
          const np = root.querySelector('#m_newpw').value;
          if (np) body.password = np;
          await api('/api/v1/admin/students/' + existing.id, { method: 'PATCH', jsonBody: body });
        } else {
          await api('/api/v1/admin/students', {
            method: 'POST',
            jsonBody: {
              identifier: root.querySelector('#m_idf').value.trim(),
              password: root.querySelector('#m_pw').value,
              name: root.querySelector('#m_name').value.trim(),
              stage: root.querySelector('#m_stage').value.trim(),
            },
          });
        }
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  async function openPapersModal(studentId) {
    const root = document.getElementById('modalRoot');
    root.innerHTML = `<div class="modal-backdrop" id="mdb2"><div class="modal wide"><h3>学生 #${studentId} 的试卷</h3><p id="pload" class="muted">加载中…</p><div id="plist"></div><button type="button" class="btn secondary" id="pcancel">关闭</button></div></div>`;
    const close = () => {
      root.innerHTML = '';
    };
    root.querySelector('#pcancel').addEventListener('click', close);
    root.querySelector('#mdb2').addEventListener('click', (e) => {
      if (e.target.id === 'mdb2') close();
    });
    try {
      const d = await api('/api/v1/admin/students/' + studentId + '/papers');
      const rows = (d.items || [])
        .map(
          (p) =>
            `<tr><td>${p.id}</td><td>${escapeHtml(p.subject)}</td><td>${escapeHtml(p.stage)}</td><td>${escapeHtml(
              p.file_name
            )}</td><td>
            <button type="button" class="btn small" data-pid="${p.id}">分析</button>
          </td></tr>`
        )
        .join('');
      root.querySelector('#pload').textContent = '';
      root.querySelector('#plist').innerHTML = `<table class="data"><thead><tr><th>ID</th><th>科目</th><th>阶段</th><th>文件</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="5">暂无试卷</td></tr>'}</tbody></table>`;
      root.querySelectorAll('[data-pid]').forEach((btn) => {
        btn.addEventListener('click', async () => {
          const pid = btn.getAttribute('data-pid');
          try {
            const an = await api('/api/v1/admin/students/' + studentId + '/papers/' + pid + '/analysis');
            const a = an.analysis;
            alert(
              '状态：' +
                a.status +
                '\n摘要：' +
                (a.summary || '') +
                '\n薄弱点：' +
                (a.weak_points || []).join(', ')
            );
          } catch (e) {
            alert('加载分析失败: ' + (e.data && e.data.code ? e.data.code : e.message));
          }
        });
      });
    } catch (e) {
      root.querySelector('#pload').textContent = '加载失败';
    }
  }

  function renderSubjects() {
    const rows = state.subjects
      .map(
        (s) =>
          `<tr><td>${s.id}</td><td>${escapeHtml(s.name)}</td><td>${escapeHtml(
            s.description || '—'
          )}</td><td>${s.status}</td>
        <td><button type="button" class="btn small" data-sid="${s.id}">编辑</button></td></tr>`
      )
      .join('');
    return `
      <div class="toolbar"><h2 style="margin:0">科目</h2><button type="button" class="btn" id="btnAddSub">新建</button></div>
      <table class="data"><thead><tr><th>ID</th><th>名称</th><th>说明</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="5">暂无</td></tr>'}</tbody></table><div id="modalRoot"></div>`;
  }

  function bindSubjects(pane) {
    const mr = pane.querySelector('#modalRoot');
    pane.querySelector('#btnAddSub').addEventListener('click', () => openSubjectModal(mr, null));
    pane.querySelectorAll('button[data-sid]').forEach((b) => {
      b.addEventListener('click', () => {
        const id = Number(b.getAttribute('data-sid'));
        openSubjectModal(mr, state.subjects.find((x) => x.id === id));
      });
    });
  }

  function openSubjectModal(mr, ex) {
    const edit = !!ex;
    mr.innerHTML = `
      <div class="modal-backdrop" id="bd"><div class="modal"><h3>${edit ? '编辑科目' : '新建科目'}</h3>
      <div class="form-grid">
        <div><label>名称</label><input id="sn" value="${edit ? escapeHtml(ex.name) : ''}" /></div>
        <div><label>说明</label><input id="sd" value="${edit && ex.description ? escapeHtml(ex.description) : ''}" /></div>
        ${edit ? `<div><label>状态</label><input id="ss" type="number" value="${ex.status}" /></div>` : ''}
      </div>
      <div class="row"><button type="button" class="btn" id="sok">保存</button><button type="button" class="btn secondary" id="sx">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#sx').addEventListener('click', close);
    mr.querySelector('#bd').addEventListener('click', (e) => {
      if (e.target.id === 'bd') close();
    });
    mr.querySelector('#sok').addEventListener('click', async () => {
      try {
        if (edit) {
          await api('/api/v1/admin/subjects/' + ex.id, {
            method: 'PATCH',
            jsonBody: {
              name: mr.querySelector('#sn').value.trim(),
              description: mr.querySelector('#sd').value,
              status: Number(mr.querySelector('#ss').value),
            },
          });
        } else {
          await api('/api/v1/admin/subjects', {
            method: 'POST',
            jsonBody: {
              name: mr.querySelector('#sn').value.trim(),
              description: mr.querySelector('#sd').value.trim(),
            },
          });
        }
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderStages() {
    const rows = state.stages
      .map(
        (s) =>
          `<tr><td>${s.id}</td><td>${escapeHtml(s.name)}</td><td>${escapeHtml(
            s.description || '—'
          )}</td><td>${s.status}</td>
        <td><button type="button" class="btn small" data-stid="${s.id}">编辑</button></td></tr>`
      )
      .join('');
    return `
      <div class="toolbar"><h2 style="margin:0">阶段</h2><button type="button" class="btn" id="btnAddStg">新建</button></div>
      <table class="data"><thead><tr><th>ID</th><th>名称</th><th>说明</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="5">暂无</td></tr>'}</tbody></table><div id="modalRoot"></div>`;
  }

  function bindStages(pane) {
    const mr = pane.querySelector('#modalRoot');
    pane.querySelector('#btnAddStg').addEventListener('click', () => openStageModal(mr, null));
    pane.querySelectorAll('button[data-stid]').forEach((b) => {
      b.addEventListener('click', () => {
        const id = Number(b.getAttribute('data-stid'));
        openStageModal(mr, state.stages.find((x) => x.id === id));
      });
    });
  }

  function openStageModal(mr, ex) {
    const edit = !!ex;
    mr.innerHTML = `
      <div class="modal-backdrop" id="bd"><div class="modal"><h3>${edit ? '编辑阶段' : '新建阶段'}</h3>
      <div class="form-grid">
        <div><label>名称</label><input id="sn" value="${edit ? escapeHtml(ex.name) : ''}" /></div>
        <div><label>说明</label><input id="sd" value="${edit && ex.description ? escapeHtml(ex.description) : ''}" /></div>
        ${edit ? `<div><label>状态</label><input id="ss" type="number" value="${ex.status}" /></div>` : ''}
      </div>
      <div class="row"><button type="button" class="btn" id="sok">保存</button><button type="button" class="btn secondary" id="sx">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#sx').addEventListener('click', close);
    mr.querySelector('#bd').addEventListener('click', (e) => {
      if (e.target.id === 'bd') close();
    });
    mr.querySelector('#sok').addEventListener('click', async () => {
      try {
        if (edit) {
          await api('/api/v1/admin/stages/' + ex.id, {
            method: 'PATCH',
            jsonBody: {
              name: mr.querySelector('#sn').value.trim(),
              description: mr.querySelector('#sd').value,
              status: Number(mr.querySelector('#ss').value),
            },
          });
        } else {
          await api('/api/v1/admin/stages', {
            method: 'POST',
            jsonBody: {
              name: mr.querySelector('#sn').value.trim(),
              description: mr.querySelector('#sd').value.trim(),
            },
          });
        }
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderModels() {
    const rows = state.models
      .map(
        (m) =>
          `<tr><td>${m.id}</td><td>${escapeHtml(m.name)}</td><td style="word-break:break-all">${escapeHtml(
            m.url
          )}</td><td>${escapeHtml(m.model != null && m.model !== '' ? m.model : m.app_key || '')}</td><td>${
            m.status === 1 ? '<span class="badge on">激活</span>' : '<span class="badge off">未激活</span>'
          }</td>
        <td>
          <button type="button" class="btn small" data-mid="${m.id}">编辑</button>
          ${
            m.status !== 1
              ? `<button type="button" class="btn secondary small" data-act="act" data-mid="${m.id}">激活</button>`
              : ''
          }
        </td></tr>`
      )
      .join('');
    return `
      <div class="toolbar"><h2 style="margin:0">AI 模型</h2><button type="button" class="btn" id="btnAddModel">新建</button></div>
      <p class="muted">同时仅允许一个激活模型；列表不显示密钥。</p>
      <table class="data"><thead><tr><th>ID</th><th>名称</th><th>URL</th><th>model</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="6">暂无</td></tr>'}</tbody></table><div id="modalRoot"></div>`;
  }

  function bindModels(pane) {
    const mr = pane.querySelector('#modalRoot');
    pane.querySelector('#btnAddModel').addEventListener('click', () => openModelModal(mr, null));
    pane.querySelectorAll('button[data-mid]').forEach((b) => {
      const id = Number(b.getAttribute('data-mid'));
      if (b.getAttribute('data-act') === 'act') {
        b.addEventListener('click', async () => {
          try {
            await api('/api/v1/admin/ai-models/' + id, { method: 'PATCH', jsonBody: { status: 1 } });
            mount(document.getElementById('app'));
          } catch (e) {
            alert(e.data && e.data.code ? e.data.code : e.message);
          }
        });
        return;
      }
      b.addEventListener('click', () => openModelModal(mr, state.models.find((x) => x.id === id)));
    });
  }

  function openModelModal(mr, ex) {
    const edit = !!ex;
    mr.innerHTML = `
      <div class="modal-backdrop" id="bd"><div class="modal wide"><h3>${edit ? '编辑模型' : '新建模型'}</h3>
      <div class="form-grid">
        <div><label>名称</label><input id="n" value="${edit ? escapeHtml(ex.name) : ''}" /></div>
        <div><label>URL</label><input id="u" value="${edit ? escapeHtml(ex.url) : ''}" /></div>
        <div><label>model（上游 chat model）</label><input id="k" value="${edit ? escapeHtml((ex.model != null && ex.model !== '' ? ex.model : ex.app_key) || '') : ''}" /></div>
        <div><label>app_secret（${edit ? '留空不改' : '必填'}）</label><input id="s" type="password" /></div>
        ${edit ? `<div><label>状态 1 激活</label><input id="st" type="number" value="${ex.status}" /></div>` : ''}
      </div>
      <div class="row"><button type="button" class="btn" id="ok">保存</button><button type="button" class="btn secondary" id="sx">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#sx').addEventListener('click', close);
    mr.querySelector('#bd').addEventListener('click', (e) => {
      if (e.target.id === 'bd') close();
    });
    mr.querySelector('#ok').addEventListener('click', async () => {
      try {
        const body = {
          name: mr.querySelector('#n').value.trim(),
          url: mr.querySelector('#u').value.trim(),
          model: mr.querySelector('#k').value.trim(),
        };
        const sec = mr.querySelector('#s').value;
        if (edit) {
          if (sec) body.app_secret = sec;
          body.status = Number(mr.querySelector('#st').value);
          await api('/api/v1/admin/ai-models/' + ex.id, { method: 'PATCH', jsonBody: body });
        } else {
          body.app_secret = sec;
          await api('/api/v1/admin/ai-models', { method: 'POST', jsonBody: body });
        }
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderPrompts() {
    const rows = state.prompts
      .map(
        (p) =>
          `<tr><td>${p.id}</td><td><code>${escapeHtml(p.key)}</code></td><td>${escapeHtml(
            (p.description || '—') + ''
          )}</td><td>${p.status}</td>
        <td><button type="button" class="btn small" data-pid="${p.id}">编辑</button></td></tr>`
      )
      .join('');
    return `
      <div class="toolbar"><h2 style="margin:0">Prompt 模板</h2></div>
      <p class="muted">仅可编辑内容与说明；系统预置模板由迁移/种子写入，不提供新增或删除。占位符：<code>%subject</code> <code>%stage</code> <code>%file_name</code>。有试卷图片时由接口以多模态形式附在消息里，由模型自行识图分析，无需用占位符塞 OCR 文本。</p>
      <table class="data"><thead><tr><th>ID</th><th>key</th><th>说明</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="5">暂无</td></tr>'}</tbody></table><div id="modalRoot"></div>`;
  }

  function bindPrompts(pane) {
    const mr = pane.querySelector('#modalRoot');
    pane.querySelectorAll('button[data-pid]').forEach((b) => {
      b.addEventListener('click', () => {
        const id = Number(b.getAttribute('data-pid'));
        openPromptModal(mr, state.prompts.find((x) => x.id === id));
      });
    });
  }

  function openPromptModal(mr, ex) {
    if (!ex) return;
    mr.innerHTML = `
      <div class="modal-backdrop" id="bd"><div class="modal wide"><h3>编辑 Prompt</h3>
      <div class="form-grid">
        <div><label>key（只读）</label><input id="k" value="${escapeHtml(ex.key)}" readonly /></div>
        <div><label>说明</label><input id="d" value="${ex.description ? escapeHtml(ex.description) : ''}" /></div>
        <div><label>内容</label><textarea id="c">${escapeHtml(ex.content)}</textarea></div>
        <div><label>状态</label><input id="st" type="number" value="${ex.status}" /></div>
      </div>
      <p class="muted" style="margin-top:8px">占位符：<code>%subject</code> <code>%stage</code> <code>%file_name</code></p>
      <div class="row"><button type="button" class="btn" id="ok">保存</button><button type="button" class="btn secondary" id="sx">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#sx').addEventListener('click', close);
    mr.querySelector('#bd').addEventListener('click', (e) => {
      if (e.target.id === 'bd') close();
    });
    mr.querySelector('#ok').addEventListener('click', async () => {
      try {
        await api('/api/v1/admin/prompts/' + ex.id, {
          method: 'PATCH',
          jsonBody: {
            description: mr.querySelector('#d').value,
            content: mr.querySelector('#c').value,
            status: Number(mr.querySelector('#st').value),
          },
        });
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderAICallLogs() {
    const f = state.aiLogFilters;
    const rows = state.aiLogs
      .flatMap((e) => {
        const id = e.id;
        const head = `<tr class="ai-log-main">
        <td>${id}</td>
        <td class="ai-wrap" style="font-size:12px">${escapeHtml(e.created_at || '')}</td>
        <td class="ai-wrap">${escapeHtml(e.model_name_snapshot || '')}</td>
        <td>${escapeHtml(e.action || '')}</td>
        <td class="ai-wrap">${escapeHtml(e.adapter_kind || '')}</td>
        <td>${escapeHtml(e.outcome || '')}</td>
        <td>${e.latency_ms != null ? e.latency_ms : '—'}</td>
        <td class="ai-wrap" style="font-size:12px">${aiLogErrLine(e)}</td>
        <td class="ai-wrap" style="font-size:12px">${escapeHtml(e.endpoint_host || '')}</td>
        <td class="ai-wrap">${escapeHtml(e.chat_model || '')}</td>
        <td>${e.fallback_to_mock ? '是' : '否'}</td>
        <td><button type="button" class="btn secondary small" data-ailog-toggle="${id}" aria-expanded="false">展开详情</button></td>
      </tr>`;
        const detail = `<tr class="ai-log-detail" data-ailog-detail="${id}" hidden><td colspan="12">${renderAICallDetailPanel(
          e
        )}</td></tr>`;
        return [head, detail];
      })
      .join('');
    return `
      <h2>AI 调用日志</h2>
      <p class="muted">展示脱敏后的出站 JSON 与上游响应（内联图片 base64 已替换为占位符）；密钥永不记录。筛选仍可用模型 id，列表中已省略模型/试卷/学生 id 与重复 HTTP 列。</p>
      <div class="form-grid" style="grid-template-columns:repeat(auto-fill,minmax(140px,1fr));max-width:1100px;align-items:end;margin-bottom:12px">
        <div>
          <label>筛选：模型 id</label>
          <input type="text" id="ailogModel" placeholder="ai_model.id" value="${escapeHtml(f.ai_model_id)}" />
        </div>
        <div>
          <label>动作</label>
          <input type="text" id="ailogAction" placeholder="paper_analyze" value="${escapeHtml(f.action)}" />
        </div>
        <div>
          <label>结果</label>
          <select id="ailogStatus">
            <option value="">全部</option>
            <option value="success" ${f.result_status === 'success' ? 'selected' : ''}>success</option>
            <option value="fallback_mock" ${f.result_status === 'fallback_mock' ? 'selected' : ''}>fallback_mock</option>
            <option value="mock_only" ${f.result_status === 'mock_only' ? 'selected' : ''}>mock_only</option>
          </select>
        </div>
        <div>
          <label>适配器</label>
          <input type="text" id="ailogAdp" placeholder="http_chat..." value="${escapeHtml(f.adapter_kind)}" />
        </div>
        <div>
          <label>开始日期</label>
          <input type="text" id="ailogFrom" placeholder="2026-04-01 或 RFC3339" value="${escapeHtml(f.from)}" />
        </div>
        <div>
          <label>结束日期</label>
          <input type="text" id="ailogTo" placeholder="2026-04-03" value="${escapeHtml(f.to)}" />
        </div>
        <div>
          <label>每页</label>
          <input type="number" id="ailogLimit" min="1" max="200" value="${Number(f.limit) || 50}" />
        </div>
        <div class="row" style="gap:8px">
          <button type="button" class="btn" id="ailogQuery">查询</button>
          <button type="button" class="btn secondary small" id="ailogPrev" ${f.offset <= 0 ? 'disabled' : ''}>上一页</button>
          <button type="button" class="btn secondary small" id="ailogNext">下一页</button>
        </div>
      </div>
      <div class="table-wrap-ai">
      <table class="data ai-logs">
        <thead><tr>
          <th>ID</th><th>时间</th><th>模型名</th><th>动作</th><th>适配器</th><th>状态</th><th>ms</th>
          <th>错误</th><th>Endpoint</th><th>chat</th><th>mock</th><th>详情</th>
        </tr></thead>
        <tbody>${rows || '<tr><td colspan="12">暂无</td></tr>'}</tbody>
      </table></div>`;
  }

  function bindAICallLogs(pane) {
    const readFilters = (offsetVal) => ({
      limit: Math.min(200, Math.max(1, Number(pane.querySelector('#ailogLimit').value) || 50)),
      offset: Math.max(0, offsetVal),
      ai_model_id: pane.querySelector('#ailogModel').value.trim(),
      action: pane.querySelector('#ailogAction').value.trim(),
      result_status: pane.querySelector('#ailogStatus').value,
      adapter_kind: pane.querySelector('#ailogAdp').value.trim(),
      from: pane.querySelector('#ailogFrom').value.trim(),
      to: pane.querySelector('#ailogTo').value.trim(),
    });
    pane.querySelector('#ailogQuery').addEventListener('click', () => {
      state.aiLogFilters = readFilters(0);
      mount(document.getElementById('app'));
    });
    pane.querySelector('#ailogPrev').addEventListener('click', () => {
      const lim = state.aiLogFilters.limit || 50;
      state.aiLogFilters = readFilters(Math.max(0, (state.aiLogFilters.offset || 0) - lim));
      mount(document.getElementById('app'));
    });
    pane.querySelector('#ailogNext').addEventListener('click', () => {
      const lim = state.aiLogFilters.limit || 50;
      state.aiLogFilters = readFilters((state.aiLogFilters.offset || 0) + lim);
      mount(document.getElementById('app'));
    });
    pane.querySelector('table.ai-logs tbody')?.addEventListener('click', (ev) => {
      const btn = ev.target.closest('button[data-ailog-toggle]');
      if (!btn) return;
      const id = btn.getAttribute('data-ailog-toggle');
      const detail = pane.querySelector(`tr.ai-log-detail[data-ailog-detail="${CSS.escape(String(id))}"]`);
      if (!detail) return;
      const open = !detail.hidden;
      detail.hidden = open;
      btn.setAttribute('aria-expanded', String(!open));
      btn.textContent = open ? '展开详情' : '收起详情';
    });
  }

  function renderAudit() {
    const rows = state.audits
      .map(
        (e) =>
          `<tr><td>${e.id}</td><td>${escapeHtml(e.user_type)}</td><td>${e.user_id ?? '—'}</td><td>${escapeHtml(
            e.action
          )}</td><td>${escapeHtml(e.entity_type)}</td><td>${e.entity_id ?? '—'}</td>
        <td><pre class="raw" style="max-height:120px;margin:0">${escapeHtml(JSON.stringify(e.snapshot))}</pre></td>
        <td>${escapeHtml(e.created_at || '')}</td></tr>`
      )
      .join('');
    return `
      <h2>审计日志</h2>
      <p class="muted">最近 200 条（脱敏快照）。</p>
      <div style="overflow-x:auto">
      <table class="data"><thead><tr><th>ID</th><th>用户类型</th><th>用户ID</th><th>动作</th><th>实体</th><th>实体ID</th><th>snapshot</th><th>时间</th></tr></thead><tbody>${rows ||
        '<tr><td colspan="8">暂无</td></tr>'}</tbody></table></div>`;
  }

  function boot() {
    mount(document.getElementById('app'));
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', boot);
  else boot();
})();
