(function () {
  'use strict';

  const LS_TOKEN = 'stepup_admin_token';
  const LS_API = 'stepup_api_base';

  function apiBase() {
    const q = new URLSearchParams(location.search).get('api');
    if (q) return q.replace(/\/$/, '');
    const s = localStorage.getItem(LS_API);
    if (s) return s.replace(/\/$/, '');
    return 'http://localhost:8080';
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
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
              <label>API 基础地址（可选，默认 localhost:8080）</label>
              <input type="text" id="apiBase" placeholder="http://localhost:8080" value="${escapeHtml(apiBase())}" />
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
      const ab = root.querySelector('#apiBase').value.trim();
      if (ab) localStorage.setItem(LS_API, ab.replace(/\/$/, ''));
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
          )}</td><td>${escapeHtml(m.app_key)}</td><td>${
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
      <table class="data"><thead><tr><th>ID</th><th>名称</th><th>URL</th><th>app_key</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
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
        <div><label>app_key（模型名等）</label><input id="k" value="${edit ? escapeHtml(ex.app_key) : ''}" /></div>
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
          app_key: mr.querySelector('#k').value.trim(),
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
      <div class="toolbar"><h2 style="margin:0">Prompt 模板</h2><button type="button" class="btn" id="btnAddPr">新建</button></div>
      <table class="data"><thead><tr><th>ID</th><th>key</th><th>说明</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="5">暂无</td></tr>'}</tbody></table><div id="modalRoot"></div>`;
  }

  function bindPrompts(pane) {
    const mr = pane.querySelector('#modalRoot');
    pane.querySelector('#btnAddPr').addEventListener('click', () => openPromptModal(mr, null));
    pane.querySelectorAll('button[data-pid]').forEach((b) => {
      b.addEventListener('click', () => {
        const id = Number(b.getAttribute('data-pid'));
        openPromptModal(mr, state.prompts.find((x) => x.id === id));
      });
    });
  }

  function openPromptModal(mr, ex) {
    const edit = !!ex;
    mr.innerHTML = `
      <div class="modal-backdrop" id="bd"><div class="modal wide"><h3>${edit ? '编辑 Prompt' : '新建 Prompt'}</h3>
      <div class="form-grid">
        <div><label>key</label><input id="k" value="${edit ? escapeHtml(ex.key) : ''}" ${edit ? 'readonly' : ''} /></div>
        <div><label>说明</label><input id="d" value="${edit && ex.description ? escapeHtml(ex.description) : ''}" /></div>
        <div><label>内容</label><textarea id="c">${edit ? escapeHtml(ex.content) : ''}</textarea></div>
        ${edit ? `<div><label>状态</label><input id="st" type="number" value="${ex.status}" /></div>` : ''}
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
        if (edit) {
          await api('/api/v1/admin/prompts/' + ex.id, {
            method: 'PATCH',
            jsonBody: {
              description: mr.querySelector('#d').value,
              content: mr.querySelector('#c').value,
              status: Number(mr.querySelector('#st').value),
            },
          });
        } else {
          await api('/api/v1/admin/prompts', {
            method: 'POST',
            jsonBody: {
              key: mr.querySelector('#k').value.trim(),
              description: mr.querySelector('#d').value.trim(),
              content: mr.querySelector('#c').value,
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
