/**
 * Slide Deck 渲染（schemaVersion 1）— 与 docs/core/slide_deck_design_v0.1_260403.md 一致。
 * 依赖（可选）：window.katex — 用于 LaTeX；无则退化为等宽文本。
 */
(function (global) {
  'use strict';

  const VALID_TEMPLATES = new Set([
    'cover-image',
    'title-body',
    'formula-focus',
    'split-left-right',
    'split-top-bottom',
    'quiz-center',
    'bullet-steps',
    'two-column-text',
  ]);

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  /** 极简 Markdown：换行、**粗体**、以 "- " 开头的行转为列表项（块级简单处理） */
  function renderSimpleMarkdown(text) {
    if (text == null) return '';
    const raw = String(text);
    const lines = raw.split(/\r?\n/);
    let inList = false;
    let html = '';
    function closeList() {
      if (inList) {
        html += '</ul>';
        inList = false;
      }
    }
    for (const line of lines) {
      const listM = /^-\s+(.+)$/.exec(line);
      if (listM) {
        if (!inList) {
          html += '<ul class="slide-md-ul">';
          inList = true;
        }
        html += '<li>' + inlineFormat(escapeHtml(listM[1])) + '</li>';
      } else {
        closeList();
        if (line.trim() === '') {
          html += '<br/>';
        } else {
          html += '<p class="slide-md-p">' + inlineFormat(escapeHtml(line)) + '</p>';
        }
      }
    }
    closeList();
    return html || '<p class="slide-md-p">—</p>';
  }

  function inlineFormat(escapedLine) {
    return escapedLine.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  }

  function renderLatex(tex, display) {
    const t = String(tex || '');
    if (!global.katex || !global.katex.renderToString) {
      return '<pre class="slide-latex-fallback">' + escapeHtml(t) + '</pre>';
    }
    try {
      return global.katex.renderToString(t, {
        displayMode: display !== 'inline',
        throwOnError: false,
        strict: 'ignore',
      });
    } catch {
      return '<pre class="slide-latex-fallback">' + escapeHtml(t) + '</pre>';
    }
  }

  function maxStepOnSlide(slide) {
    let m = 1;
    const els = slide.elements || [];
    for (let i = 0; i < els.length; i++) {
      const s = Number(els[i].step);
      if (!isNaN(s) && s > m) m = s;
    }
    return m;
  }

  function filterVisible(slide, currentStep) {
    return (slide.elements || []).filter((e) => {
      const s = Number(e.step);
      const step = isNaN(s) ? 1 : s;
      return step <= currentStep;
      });
  }

  function renderQuestion(el, slideId, qIndex, answers) {
    const mode = el.mode === 'multi' ? 'multi' : 'single';
    const data = el.data || {};
    const qtext = data.text || '';
    const opts = data.options || [];
    const name = 'q-' + slideId + '-' + qIndex;
    const selected = answers[name] != null ? answers[name] : mode === 'multi' ? [] : '';
    let body = '<div class="slide-question"><div class="slide-q-prompt slide-md">' + renderSimpleMarkdown(qtext) + '</div><div class="slide-q-options">';
    opts.forEach((opt) => {
      const oid = escapeHtml(String(opt.id != null ? opt.id : ''));
      const otext = opt.text != null ? String(opt.text) : '';
      const idAttr = name + '-' + oid;
      if (mode === 'multi') {
        const checked = Array.isArray(selected) && selected.indexOf(opt.id) >= 0 ? ' checked' : '';
        body +=
          '<label class="slide-q-opt"><input type="checkbox" name="' +
          escapeHtml(name) +
          '" value="' +
          oid +
          '"' +
          checked +
          '/> <span class="slide-q-opt-text slide-md">' +
          renderSimpleMarkdown(otext) +
          '</span></label>';
      } else {
        const checked = selected === opt.id ? ' checked' : '';
        body +=
          '<label class="slide-q-opt"><input type="radio" name="' +
          escapeHtml(name) +
          '" value="' +
          oid +
          '"' +
          checked +
          '/> <span class="slide-q-opt-text slide-md">' +
          renderSimpleMarkdown(otext) +
          '</span></label>';
      }
    });
    body += '</div></div>';
    return body;
  }

  function bindQuestions(hostEl, slide, answers, onAnswer) {
    const slideId = slide.id || 'slide';
    const els = filterVisible(slide, hostEl._currentStep || 1);
    let qi = 0;
    els.forEach((el) => {
      if (el.type !== 'question') return;
      const name = 'q-' + slideId + '-' + qi;
      qi++;
      const mode = el.mode === 'multi' ? 'multi' : 'single';
      const inputs = [];
      hostEl.querySelectorAll('input').forEach((inp) => {
        if (inp.name === name) inputs.push(inp);
      });
      inputs.forEach((inp) => {
        inp.addEventListener('change', () => {
          if (mode === 'multi') {
            const ids = [];
            hostEl.querySelectorAll('input').forEach((b) => {
              if (b.name === name && b.checked) ids.push(b.value);
            });
            answers[name] = ids;
          } else {
            answers[name] = inp.value;
          }
          if (typeof onAnswer === 'function') {
            onAnswer({
              slideId: slideId,
              questionKey: name,
              mode: mode,
              selected: answers[name],
            });
          }
        });
      });
    });
  }

  function renderElementHTML(el, slide, qCounterRef) {
    const role = el.role || 'body';
    const typ = el.type || 'text';
    if (typ === 'text') {
      return (
        '<div class="slide-el slide-type-text slide-role-' +
        escapeHtml(role) +
        '"><div class="slide-md">' +
        renderSimpleMarkdown(el.content) +
        '</div></div>'
      );
    }
    if (typ === 'latex') {
      const disp = el.display === 'inline' ? 'inline' : 'block';
      const inner = renderLatex(el.content, disp);
      return (
        '<div class="slide-el slide-type-latex slide-role-' +
        escapeHtml(role) +
        '">' +
        inner +
        '</div>'
      );
    }
    if (typ === 'image') {
      const src = el.src ? String(el.src) : '';
      const alt = el.alt != null ? String(el.alt) : '';
      const cap = el.caption != null ? String(el.caption) : '';
      let h =
        '<div class="slide-el slide-type-image slide-role-' +
        escapeHtml(role) +
        '"><img src="' +
        escapeHtml(src) +
        '" alt="' +
        escapeHtml(alt) +
        '" class="slide-img" loading="lazy"/>';
      if (cap) h += '<div class="slide-caption slide-md">' + renderSimpleMarkdown(cap) + '</div>';
      h += '</div>';
      return h;
    }
    if (typ === 'question') {
      const idx = qCounterRef.v++;
      return renderQuestion(el, slide.id || 'slide', idx, qCounterRef.answers || {});
    }
    return '<div class="slide-el slide-unknown">未知类型</div>';
  }

  function buildSlideInner(slide, currentStep, answers) {
    const tpl = slide.layoutTemplate || 'title-body';
    const safeTpl = VALID_TEMPLATES.has(tpl) ? tpl : 'title-body';
    const qRef = { v: 0, answers: answers };
    const visible = filterVisible(slide, currentStep);
    let inner = visible.map((el) => renderElementHTML(el, slide, qRef)).join('');

    const wrap =
      '<div class="slide-tpl slide-tpl-' +
      escapeHtml(safeTpl) +
      '" data-template="' +
      escapeHtml(safeTpl) +
      '">' +
      inner +
      '</div>';
    return wrap;
  }

  /**
   * @param {HTMLElement} container
   * @param {object} deck - 完整 deck JSON
   * @param {object} [options]
   * @param {number} [options.initialSlide]
   * @param {function} [options.onNavigate] ({ slideIndex, slideId, step, maxStep })
   * @param {function} [options.onAnswer] ({ slideId, questionKey, mode, selected })
   */
  function mount(container, deck, options) {
    if (!container || !deck || !Array.isArray(deck.slides)) {
      throw new Error('SlideDeckRenderer.mount: invalid args');
    }
    const schemaVersion = Number(deck.schemaVersion);
    if (schemaVersion !== 1) {
      container.innerHTML =
        '<p class="muted">不支持的 schemaVersion（需要 1）。</p>';
      return { destroy: function () {}, getContext: function () {} };
    }

    options = options || {};
    const theme = (deck.meta && deck.meta.theme) || 'light-default';
    let slideIndex = options.initialSlide != null ? Number(options.initialSlide) : 0;
    if (slideIndex < 0) slideIndex = 0;
    if (slideIndex >= deck.slides.length) slideIndex = Math.max(0, deck.slides.length - 1);

    let currentStep = 1;
    const answers = {};

    const host = document.createElement('div');
    host.className = 'slide-deck-root theme-' + String(theme).replace(/[^a-z0-9-]/gi, '-');
    host.innerHTML =
      '<div class="slide-stage-wrap"><div class="slide-stage" id="slideStage"></div></div>' +
      '<div class="slide-toolbar">' +
      '<span class="slide-progress" id="slideProg"></span>' +
      '<button type="button" class="btn secondary sm" id="btnSlidePrev">上一页</button>' +
      '<button type="button" class="btn secondary sm" id="btnStepNext">下一步</button>' +
      '<button type="button" class="btn sm" id="btnSlideNext">下一页</button>' +
      '</div>';

    container.innerHTML = '';
    container.appendChild(host);

    const stage = host.querySelector('#slideStage');
    const prog = host.querySelector('#slideProg');
    const btnPrev = host.querySelector('#btnSlidePrev');
    const btnStep = host.querySelector('#btnStepNext');
    const btnNext = host.querySelector('#btnSlideNext');

    function getSlide() {
      return deck.slides[slideIndex] || { elements: [], id: '' };
    }

    function paint() {
      const sl = getSlide();
      currentStep = Math.min(currentStep, Math.max(1, maxStepOnSlide(sl)));
      host._currentStep = currentStep;
      stage.innerHTML = buildSlideInner(sl, currentStep, answers);
      bindQuestions(host, sl, answers, options.onAnswer);
      const mx = maxStepOnSlide(sl);
      prog.textContent = '第 ' + (slideIndex + 1) + ' / ' + deck.slides.length + ' 页 · 步 ' + currentStep + ' / ' + mx;
      btnStep.disabled = currentStep >= mx && slideIndex >= deck.slides.length - 1;
      btnPrev.disabled = slideIndex <= 0 && currentStep <= 1;
      btnNext.disabled = slideIndex >= deck.slides.length - 1;
      if (typeof options.onNavigate === 'function') {
        options.onNavigate({
          slideIndex: slideIndex,
          slideId: sl.id || '',
          step: currentStep,
          maxStep: mx,
        });
      }
    }

    btnPrev.addEventListener('click', () => {
      const sl = getSlide();
      const mx = maxStepOnSlide(sl);
      if (currentStep > 1) {
        currentStep--;
        paint();
        return;
      }
      if (slideIndex > 0) {
        slideIndex--;
        currentStep = Math.max(1, maxStepOnSlide(getSlide()));
        paint();
      }
    });

    btnStep.addEventListener('click', () => {
      const sl = getSlide();
      const mx = maxStepOnSlide(sl);
      if (currentStep < mx) {
        currentStep++;
        paint();
        return;
      }
      if (slideIndex < deck.slides.length - 1) {
        slideIndex++;
        currentStep = 1;
        paint();
      }
    });

    btnNext.addEventListener('click', () => {
      if (slideIndex < deck.slides.length - 1) {
        slideIndex++;
        currentStep = 1;
        paint();
      }
    });

    paint();

    return {
      destroy: function () {
        if (host.parentNode) host.parentNode.removeChild(host);
      },
      getContext: function () {
        const sl = getSlide();
        return {
          deckTitle: (deck.meta && deck.meta.title) || '',
          slideIndex: slideIndex,
          slideId: sl.id || '',
          step: currentStep,
          maxStep: maxStepOnSlide(sl),
          answers: JSON.parse(JSON.stringify(answers)),
        };
      },
      nextStep: function () {
        btnStep.click();
      },
    };
  }

  global.SlideDeckRenderer = {
    mount: mount,
    renderSimpleMarkdown: renderSimpleMarkdown,
    VALID_TEMPLATES: VALID_TEMPLATES,
  };
})(typeof window !== 'undefined' ? window : this);
(function () {
  'use strict';

  const LS_TOKEN = 'stepup_admin_token';
  const LS_API = 'stepup_api_base';
  const LS_SLIDE_GEN_PROMPT_PREFIX = 'stepup_admin_slide_gen_prompt';

  function slideGenPromptStorageKey(sectionId) {
    return LS_SLIDE_GEN_PROMPT_PREFIX + '_' + String(sectionId);
  }

  function appendAdminSlideLog(preEl, text) {
    if (!preEl) return;
    preEl.textContent += '[' + new Date().toLocaleTimeString('zh-CN') + '] ' + text + '\n';
    preEl.scrollTop = preEl.scrollHeight;
  }

  function pickDeckForPreview(items) {
    if (!items || !items.length) return null;
    const active = items.find(function (x) {
      return x.deck_status === 'active';
    });
    return active || items[0];
  }

  function parseDeckContent(content) {
    if (content == null) return null;
    if (typeof content === 'object') return content;
    if (typeof content === 'string') {
      try {
        return JSON.parse(content);
      } catch (e) {
        return null;
      }
    }
    return null;
  }

  /** 与部署约定一致时可不配置 meta：`?api=` / localStorage / meta / 登录框端口 仍可覆盖。 */
  const PAGE_PORT_TO_API_PORT = { '7010': '7012', '7011': '7012' };
  /** 与根目录 `.env.example` 中 `BACKEND_PORT` 默认一致。 */
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
    if (p.startsWith('/admin')) return location.origin;
    const host = location.hostname;
    const port = location.port;
    const isLocal = host === 'localhost' || host === '127.0.0.1';
    if (isLocal && port === '3001') return 'http://localhost:8080';
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

  /** 状态 0/1：管理端单选（≤4 项用 radio） */
  function htmlAdminStatus10(nameAttr, currentValue, onLabel, offLabel) {
    const v = Number(currentValue) === 0 ? 0 : 1;
    const onL = onLabel || '激活';
    const offL = offLabel || '未激活';
    return `<div class="radio-group" role="group" aria-label="状态">
      <label class="radio-line"><input type="radio" name="${nameAttr}" value="1" ${v === 1 ? 'checked' : ''} /> ${escapeHtml(
      onL
    )}</label>
      <label class="radio-line"><input type="radio" name="${nameAttr}" value="0" ${v === 0 ? 'checked' : ''} /> ${escapeHtml(
      offL
    )}</label>
    </div>`;
  }

  function readAdminStatus10(container, nameAttr) {
    const el = container.querySelector(`input[name="${nameAttr}"]:checked`);
    if (!el) return 1;
    return Number(el.value);
  }

  /** 列表展示：业务 status 0/1（教材目录、科目、Prompt 等）。非 0/1 时原样标出（设计约定见 feature_design）。 */
  function formatAdminListStatus10(v) {
    if (v === null || v === undefined || v === '') {
      return '<span class="muted">—</span>';
    }
    const n = Number(v);
    if (!Number.isFinite(n)) {
      return '<span class="badge" title="非数字状态">' + escapeHtml(String(v)) + '</span>';
    }
    if (n === 1) return '<span class="badge on">激活</span>';
    if (n === 0) return '<span class="badge off">未激活</span>';
    return (
      '<span class="badge" title="非标准值（一般为 0=未激活，1=激活）">' + escapeHtml(String(n)) + '</span>'
    );
  }

  function tdAdminListStatus10(v) {
    const n = Number(v);
    const attr = Number.isFinite(n) ? n : 0;
    return '<td data-admin-status="' + attr + '">' + formatAdminListStatus10(v) + '</td>';
  }

  function tdSlideDeckSummary(count) {
    const c = Math.max(0, Math.floor(Number(count)) || 0);
    const inner =
      c <= 0
        ? '<span class="muted">无</span>'
        : '<span class="badge on" title="该节已有 ' +
          c +
          ' 套幻灯片记录（含草稿/生效/归档）">' +
          c +
          ' 套</span>';
    return '<td data-slide-deck-count="' + c + '">' + inner + '</td>';
  }

  function isAdminLoginPOST(path, opts) {
    const p = String(path || '').split('?')[0];
    const m = (opts.method || 'GET').toUpperCase();
    return p === '/api/v1/admin/auth/login' && m === 'POST';
  }

  /** 已在 api() 内清除会话并切换登录页时，后续 catch 勿再报错或写 pane。 */
  function authRedirectHandled(e) {
    return !!(e && e.authRedirectDone);
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
    const latSec =
      e.latency_ms != null && e.latency_ms !== ''
        ? (Number(e.latency_ms) / 1000).toFixed(3)
        : '';
    const errHtml = aiLogErrLine(e);
    const errCell =
      errHtml && errHtml !== '—' ? errHtml : '<span class="muted">—</span>';
    const mockTxt = e.fallback_to_mock ? '是' : '否';
    const cell = (label, text) => {
      const t = String(text || '').trim();
      const inner = t ? escapeHtml(t) : '<span class="muted">—</span>';
      return `<tr><th scope="row">${escapeHtml(label)}</th><td class="ai-detail-td"><pre class="ai-detail-pre">${inner}</pre></td></tr>`;
    };
    return `<table class="data ai-log-detail-inner" role="presentation"><tbody>
      ${cell('适配器', e.adapter_kind || '')}
      ${cell('用时(秒)', latSec)}
      <tr><th scope="row">错误</th><td class="ai-detail-td"><div class="ai-detail-text">${errCell}</div></td></tr>
      ${cell('Endpoint', e.endpoint_host || '')}
      ${cell('mock 回退', mockTxt)}
      ${cell(
        'student_id',
        e.student_id != null && e.student_id !== '' ? String(e.student_id) : ''
      )}
      ${cell('ref_table', e.ref_table || '')}
      ${cell('ref_id', e.ref_id != null && e.ref_id !== '' ? String(e.ref_id) : '')}
      ${cell('请求 JSON', req)}
      ${cell('响应原文', res)}
      ${cell('结构化 Meta', meta)}
    </tbody></table>`;
  }

  function formatAuditSnapshotJson(snap) {
    if (snap == null || snap === '') return '';
    try {
      const o = typeof snap === 'object' && snap !== null ? snap : JSON.parse(String(snap));
      return JSON.stringify(o, null, 2);
    } catch {
      return String(snap);
    }
  }

  function renderAuditDetailPanel(e) {
    const snapText = formatAuditSnapshotJson(e.snapshot);
    const cell = (label, text) => {
      const t = String(text || '').trim();
      const inner = t ? escapeHtml(t) : '<span class="muted">—</span>';
      return `<tr><th scope="row">${escapeHtml(label)}</th><td class="ai-detail-td"><pre class="ai-detail-pre">${inner}</pre></td></tr>`;
    };
    const ip = e.ip_address ? String(e.ip_address).trim() : '';
    const extra = e.created_by != null ? String(e.created_by) : '';
    return `<table class="data ai-log-detail-inner" role="presentation"><tbody>
      ${cell('IP', ip)}
      ${cell('记录人 ID', extra)}
      ${cell('snapshot', snapText)}
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
    auditFilters: { limit: 50, offset: 0 },
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
    /** 科目下教材目录：独立视图导航（有 textbook 的科目才从编辑弹窗进入） */
    subjectCatalog: null,
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
      if (
        res.status === 401 &&
        state.token &&
        !opts.skipAuthRedirect &&
        !isAdminLoginPOST(path, opts)
      ) {
        state.token = null;
        localStorage.removeItem(LS_TOKEN);
        state.flash = { kind: 'info', msg: '登录已失效或无权访问，请重新登录' };
        const appRoot = document.getElementById('app');
        if (appRoot) mount(appRoot);
        const err = new Error(data && data.code ? data.code : 'UNAUTHORIZED');
        err.status = 401;
        err.data = data;
        err.authRedirectDone = true;
        throw err;
      }
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
    } catch (e) {
      if (authRedirectHandled(e)) throw e;
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
    refreshView(root).catch((e) => {
      if (authRedirectHandled(e)) return;
      setFlash('err', e.message || String(e));
    });
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
          `<button type="button" data-view="${k}" class="${
            state.view === k || (state.view === 'subject_catalog' && k === 'subjects') ? 'active' : ''
          }">${escapeHtml(lab)}</button>`
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
    const side = root.querySelector('#sideMenu');
    if (side) {
      side.querySelectorAll('button').forEach((btn) => {
        btn.addEventListener('click', () => {
          state.view = btn.getAttribute('data-view');
          state.flash = null;
          state.subjectCatalog = null;
          mount(document.getElementById('app'));
        });
      });
    }
  }

  async function refreshView(root) {
    const pane = root.querySelector('#mainPane');
    if (!pane) return;
    await loadCatalog();
    try {
      if (state.view === 'subject_catalog') {
        const nav = state.subjectCatalog;
        if (!nav || !nav.subjectId) {
          state.view = 'subjects';
          state.subjectCatalog = null;
          mount(root);
          return;
        }
        if (nav.mode === 'textbooks') {
          const d = await api('/api/v1/admin/subjects/' + nav.subjectId + '/textbooks');
          const items = d.items || [];
          pane.innerHTML = renderTextbookCatalogPage(nav, items);
          bindTextbookCatalog(pane, nav);
          return;
        }
        if (nav.mode === 'chapters') {
          const d = await api('/api/v1/admin/textbooks/' + nav.textbookId + '/chapters');
          const items = d.items || [];
          pane.innerHTML = renderChapterCatalogPage(nav, items);
          bindChapterCatalog(pane, nav);
          return;
        }
        if (nav.mode === 'sections') {
          const d = await api('/api/v1/admin/chapters/' + nav.chapterId + '/sections');
          const items = d.items || [];
          pane.innerHTML = renderSectionCatalogPage(nav, items);
          bindSectionCatalog(pane, nav);
          return;
        }
        state.view = 'subjects';
        state.subjectCatalog = null;
        mount(root);
        return;
      }
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
        const af = state.auditFilters;
        const lim = af.limit || 50;
        const off = af.offset || 0;
        const d = await api('/api/v1/admin/audit-logs?limit=' + lim + '&offset=' + off);
        state.audits = d.items || [];
        pane.innerHTML = renderAudit();
        bindAuditLogs(pane);
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
        return;
      }
    } catch (e) {
      if (authRedirectHandled(e)) return;
      const msg = e.data && e.data.code ? e.data.code : e.message || String(e);
      const hint =
        msg === 'Failed to fetch'
          ? '（请确认 API 已启动且地址正确；若页面与 API 不同端口，后端需将当前页面的 Origin 加入 <code>CORS_ALLOWED_ORIGINS</code>，见工具栏「API: …」所示根地址。）'
          : '';
      pane.innerHTML = `<p class="muted">加载失败：${escapeHtml(msg)}${hint}</p>`;
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
          <td data-admin-status="${Number.isFinite(Number(s.status)) ? Number(s.status) : 0}">${formatAdminListStatus10(
            s.status
          )}</td>
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
                ? `<div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10(
                    'admStudentStatus',
                    existing.status,
                    '激活',
                    '未激活'
                  )}</div>
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
            status: readAdminStatus10(root, 'admStudentStatus'),
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
        if (authRedirectHandled(e)) return;
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
            if (authRedirectHandled(e)) return;
            alert('加载分析失败: ' + (e.data && e.data.code ? e.data.code : e.message));
          }
        });
      });
    } catch (e) {
      if (authRedirectHandled(e)) return;
      root.querySelector('#pload').textContent = '加载失败';
    }
  }

  function renderSubjects() {
    const rows = state.subjects
      .map(
        (s) =>
          `<tr><td>${s.id}</td><td>${escapeHtml(s.name)}</td><td>${escapeHtml(
            s.description || '—'
          )}</td>${tdAdminListStatus10(s.status)}
        <td class="row" style="gap:6px;flex-wrap:wrap"><button type="button" class="btn small" data-sid="${s.id}">编辑</button>${
          (s.textbook_count || 0) > 0
            ? `<button type="button" class="btn small secondary" data-catalog-sid="${s.id}">目录</button>`
            : ''
        }</td></tr>`
      )
      .join('');
    return `
      <div class="toolbar"><h2 style="margin:0">科目</h2><button type="button" class="btn" id="btnAddSub">新建</button></div>
      <p class="muted">某科目下已有教材时，列表中显示「目录」，可维护教材/章/节（仅编辑，无新增删除）。</p>
      <table class="data"><thead><tr><th>ID</th><th>名称</th><th>说明</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="5">暂无</td></tr>'}</tbody></table><div id="modalRoot"></div>`;
  }

  function bindSubjects(pane) {
    const mr = pane.querySelector('#modalRoot');
    pane.querySelector('#btnAddSub').addEventListener('click', () => openSubjectModal(mr, null));
    pane.querySelectorAll('button[data-catalog-sid]').forEach((b) => {
      b.addEventListener('click', () => {
        const id = Number(b.getAttribute('data-catalog-sid'));
        const sub = state.subjects.find((x) => x.id === id);
        mountCatalog({
          subjectId: id,
          subjectName: sub ? sub.name || '' : '',
          mode: 'textbooks',
        });
      });
    });
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
        ${
          edit
            ? `<div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10(
                'admSubjectStatus',
                ex.status,
                '激活',
                '未激活'
              )}</div>`
            : ''
        }
      </div>
      <div class="row" id="subActions"><button type="button" class="btn" id="sok">保存</button><button type="button" class="btn secondary" id="sx">取消</button></div>
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
              status: readAdminStatus10(mr, 'admSubjectStatus'),
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
        if (authRedirectHandled(e)) return;
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function mountCatalog(nav) {
    state.view = 'subject_catalog';
    state.subjectCatalog = nav;
    mount(document.getElementById('app'));
  }

  function renderTextbookCatalogPage(nav, items) {
    const rows = items
      .map(
        (t) =>
          `<tr><td>${t.id}</td><td>${escapeHtml(t.name)}</td><td>${escapeHtml(t.version)}</td><td>${escapeHtml(
            t.subject
          )}</td><td>${escapeHtml(t.category)}</td><td>${escapeHtml(
            (t.remarks || '').trim() || '—'
          )}</td>${tdAdminListStatus10(t.status)}
        <td class="row" style="gap:6px;flex-wrap:wrap">
          <button type="button" class="btn small" data-edit-tb="${t.id}">编辑</button>
          <button type="button" class="btn small secondary" data-chapters-tb="${t.id}">章节</button>
        </td></tr>`
      )
      .join('');
    return `
      <div class="toolbar row" style="justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
        <div>
          <button type="button" class="btn secondary small" id="catBackSubjects">← 科目列表</button>
          <span class="muted" style="margin-left:12px;font-weight:500">${escapeHtml(
            nav.subjectName || '科目'
          )} · 教材目录</span>
        </div>
      </div>
      <p class="muted" style="margin-top:8px">仅可编辑已有教材（书名、版本、学科展示名、备注、状态）；不提供新增或删除。<strong>类别</strong>只读。</p>
      <table class="data"><thead><tr><th>ID</th><th>名称</th><th>版本</th><th>学科</th><th>类别</th><th>备注</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="8">暂无教材</td></tr>'}</tbody></table>
      <div id="catalogModalRoot"></div>`;
  }

  function bindTextbookCatalog(pane, nav) {
    pane.querySelector('#catBackSubjects').addEventListener('click', () => {
      state.view = 'subjects';
      state.subjectCatalog = null;
      mount(document.getElementById('app'));
    });
    const mr = pane.querySelector('#catalogModalRoot');
    pane.querySelectorAll('button[data-edit-tb]').forEach((b) => {
      b.addEventListener('click', () => {
        const tr = b.closest('tr');
        const id = Number(b.getAttribute('data-edit-tb'));
        const cells = tr ? tr.querySelectorAll('td') : [];
        const rm = cells[5] ? cells[5].textContent.trim() : '';
        const stTb = tr.querySelector('td[data-admin-status]');
        const rec = {
          id,
          name: cells[1] ? cells[1].textContent.trim() : '',
          version: cells[2] ? cells[2].textContent.trim() : '',
          subject: cells[3] ? cells[3].textContent.trim() : '',
          remarks: rm === '—' ? '' : rm,
          status: stTb ? Number(stTb.getAttribute('data-admin-status')) || 1 : 1,
        };
        openTextbookEditModal(mr, rec, () => mountCatalog(Object.assign({}, nav, { mode: 'textbooks' })));
      });
    });
    pane.querySelectorAll('button[data-chapters-tb]').forEach((b) => {
      b.addEventListener('click', () => {
        const tr = b.closest('tr');
        const cells = tr ? tr.querySelectorAll('td') : [];
        const tid = Number(b.getAttribute('data-chapters-tb'));
        const nm = cells[1] ? cells[1].textContent.trim() : '';
        mountCatalog({
          subjectId: nav.subjectId,
          subjectName: nav.subjectName,
          mode: 'chapters',
          textbookId: tid,
          textbookName: nm,
        });
      });
    });
  }

  function openTextbookEditModal(mr, t, onSaved) {
    mr.innerHTML = `
      <div class="modal-backdrop" id="tbd"><div class="modal"><h3>编辑教材</h3>
      <div class="form-grid">
        <div><label>名称</label><input id="tbn" value="${escapeHtml(t.name)}" /></div>
        <div><label>版本</label><input id="tbv" value="${escapeHtml(t.version)}" /></div>
        <div><label>学科（展示）</label><input id="tbs" value="${escapeHtml(t.subject)}" /></div>
        <div style="grid-column:1/-1"><label>备注</label><input id="tbr" value="${escapeHtml(t.remarks)}" /></div>
        <div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10('admTbStatus', t.status, '激活', '未激活')}</div>
      </div>
      <div class="row"><button type="button" class="btn" id="tbok">保存</button><button type="button" class="btn secondary" id="tbx">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#tbx').addEventListener('click', close);
    mr.querySelector('#tbd').addEventListener('click', (e) => {
      if (e.target.id === 'tbd') close();
    });
    mr.querySelector('#tbok').addEventListener('click', async () => {
      try {
        await api('/api/v1/admin/textbooks/' + t.id, {
          method: 'PATCH',
          jsonBody: {
            name: mr.querySelector('#tbn').value.trim(),
            version: mr.querySelector('#tbv').value.trim(),
            subject: mr.querySelector('#tbs').value.trim(),
            remarks: mr.querySelector('#tbr').value,
            status: readAdminStatus10(mr, 'admTbStatus'),
          },
        });
        close();
        onSaved();
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderChapterCatalogPage(nav, items) {
    const rows = items
      .map(
        (c) =>
          `<tr><td>${c.id}</td><td>${c.number}</td><td>${escapeHtml(c.title)}</td><td>${escapeHtml(
            (c.full_title || '').trim() || '—'
          )}</td>${tdAdminListStatus10(c.status)}
        <td class="row" style="gap:6px;flex-wrap:wrap">
          <button type="button" class="btn small" data-edit-ch="${c.id}">编辑</button>
          <button type="button" class="btn small secondary" data-sects-ch="${c.id}">小节</button>
        </td></tr>`
      )
      .join('');
    return `
      <div class="toolbar row" style="justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
        <div>
          <button type="button" class="btn secondary small" id="catBackTextbooks">← 教材列表</button>
          <span class="muted" style="margin-left:12px">${escapeHtml(nav.subjectName || '')} › ${escapeHtml(
      nav.textbookName || ''
    )} · 章</span>
        </div>
      </div>
      <p class="muted" style="margin-top:8px">仅可编辑序号、标题、完整标题、状态；不提供新增或删除。</p>
      <table class="data"><thead><tr><th>ID</th><th>序号</th><th>标题</th><th>完整标题</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="6">暂无章节</td></tr>'}</tbody></table>
      <div id="catalogModalRoot"></div>`;
  }

  function bindChapterCatalog(pane, nav) {
    pane.querySelector('#catBackTextbooks').addEventListener('click', () => {
      mountCatalog({
        subjectId: nav.subjectId,
        subjectName: nav.subjectName,
        mode: 'textbooks',
      });
    });
    const mr = pane.querySelector('#catalogModalRoot');
    pane.querySelectorAll('button[data-edit-ch]').forEach((b) => {
      b.addEventListener('click', () => {
        const tr = b.closest('tr');
        const id = Number(b.getAttribute('data-edit-ch'));
        const cells = tr ? tr.querySelectorAll('td') : [];
        const ftRaw = cells[3] ? cells[3].textContent.trim() : '';
        const stCh = tr.querySelector('td[data-admin-status]');
        const rec = {
          id,
          number: cells[1] ? Number(cells[1].textContent) : 0,
          title: cells[2] ? cells[2].textContent.trim() : '',
          full_title: ftRaw === '—' ? '' : ftRaw,
          status: stCh ? Number(stCh.getAttribute('data-admin-status')) || 1 : 1,
        };
        openChapterEditModal(mr, rec, () =>
          mountCatalog({
            subjectId: nav.subjectId,
            subjectName: nav.subjectName,
            mode: 'chapters',
            textbookId: nav.textbookId,
            textbookName: nav.textbookName,
          })
        );
      });
    });
    pane.querySelectorAll('button[data-sects-ch]').forEach((b) => {
      b.addEventListener('click', () => {
        const tr = b.closest('tr');
        const cells = tr ? tr.querySelectorAll('td') : [];
        const cid = Number(b.getAttribute('data-sects-ch'));
        const ct = cells[2] ? cells[2].textContent.trim() : '';
        mountCatalog({
          subjectId: nav.subjectId,
          subjectName: nav.subjectName,
          mode: 'sections',
          textbookId: nav.textbookId,
          textbookName: nav.textbookName,
          chapterId: cid,
          chapterTitle: ct,
        });
      });
    });
  }

  function openChapterEditModal(mr, c, onSaved) {
    mr.innerHTML = `
      <div class="modal-backdrop" id="cbd"><div class="modal"><h3>编辑章</h3>
      <div class="form-grid">
        <div><label>序号</label><input id="chnu" type="number" min="0" value="${c.number}" /></div>
        <div style="grid-column:1/-1"><label>标题</label><input id="cht" value="${escapeHtml(c.title)}" /></div>
        <div style="grid-column:1/-1"><label>完整标题</label><input id="chft" value="${escapeHtml(
          c.full_title || ''
        )}" /></div>
        <div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10('admChStatus', c.status, '激活', '未激活')}</div>
      </div>
      <div class="row"><button type="button" class="btn" id="chok">保存</button><button type="button" class="btn secondary" id="chx">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#chx').addEventListener('click', close);
    mr.querySelector('#cbd').addEventListener('click', (e) => {
      if (e.target.id === 'cbd') close();
    });
    mr.querySelector('#chok').addEventListener('click', async () => {
      try {
        await api('/api/v1/admin/chapters/' + c.id, {
          method: 'PATCH',
          jsonBody: {
            number: Number(mr.querySelector('#chnu').value),
            title: mr.querySelector('#cht').value.trim(),
            full_title: mr.querySelector('#chft').value.trim(),
            status: readAdminStatus10(mr, 'admChStatus'),
          },
        });
        close();
        onSaved();
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderSectionCatalogPage(nav, items) {
    const rows = items
      .map(
        (s) =>
          `<tr><td>${s.id}</td><td>${s.number}</td><td>${escapeHtml(s.title)}</td><td>${escapeHtml(
            (s.full_title || '').trim() || '—'
          )}</td>${tdSlideDeckSummary(s.slide_deck_count)}${tdAdminListStatus10(s.status)}
        <td class="row" style="gap:6px;flex-wrap:wrap"><button type="button" class="btn small" data-edit-se="${s.id}">编辑</button><button type="button" class="btn small secondary" data-slide-gen-se="${s.id}">生成幻灯片</button><button type="button" class="btn small secondary" data-slide-preview-se="${s.id}">试播</button></td></tr>`
      )
      .join('');
    return `
      <div class="toolbar row" style="justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
        <div>
          <button type="button" class="btn secondary small" id="catBackChapters">← 章列表</button>
          <span class="muted" style="margin-left:12px">${escapeHtml(nav.textbookName || '')} › ${escapeHtml(
      nav.chapterTitle || ''
    )} · 节</span>
        </div>
      </div>
      <p class="muted" style="margin-top:8px">仅可编辑序号、标题、完整标题、状态；不提供新增或删除。</p>
      <table class="data"><thead><tr><th>ID</th><th>序号</th><th>标题</th><th>完整标题</th><th>幻灯片</th><th>状态</th><th></th></tr></thead><tbody>${rows ||
        '<tr><td colspan="7">暂无小节</td></tr>'}</tbody></table>
      <div id="catalogModalRoot"></div>`;
  }

  function bindSectionCatalog(pane, nav) {
    pane.querySelector('#catBackChapters').addEventListener('click', () => {
      mountCatalog({
        subjectId: nav.subjectId,
        subjectName: nav.subjectName,
        mode: 'chapters',
        textbookId: nav.textbookId,
        textbookName: nav.textbookName,
      });
    });
    const mr = pane.querySelector('#catalogModalRoot');
    pane.querySelectorAll('button[data-edit-se]').forEach((b) => {
      b.addEventListener('click', () => {
        const tr = b.closest('tr');
        const id = Number(b.getAttribute('data-edit-se'));
        const cells = tr ? tr.querySelectorAll('td') : [];
        const ftRaw = cells[3] ? cells[3].textContent.trim() : '';
        const stSe = tr.querySelector('td[data-admin-status]');
        const rec = {
          id,
          number: cells[1] ? Number(cells[1].textContent) : 0,
          title: cells[2] ? cells[2].textContent.trim() : '',
          full_title: ftRaw === '—' ? '' : ftRaw,
          status: stSe ? Number(stSe.getAttribute('data-admin-status')) || 1 : 1,
        };
        openSectionEditModal(mr, rec, () =>
          mountCatalog({
            subjectId: nav.subjectId,
            subjectName: nav.subjectName,
            mode: 'sections',
            textbookId: nav.textbookId,
            textbookName: nav.textbookName,
            chapterId: nav.chapterId,
            chapterTitle: nav.chapterTitle,
          })
        );
      });
    });
    pane.querySelectorAll('button[data-slide-gen-se]').forEach(function (b) {
      b.addEventListener('click', function () {
        const tr = b.closest('tr');
        const id = Number(b.getAttribute('data-slide-gen-se'));
        const title = tr && tr.querySelectorAll('td').length > 2 ? tr.querySelectorAll('td')[2].textContent.trim() : '';
        openSlideGenerateModal(mr, id, title, function () {
          mountCatalog({
            subjectId: nav.subjectId,
            subjectName: nav.subjectName,
            mode: 'sections',
            textbookId: nav.textbookId,
            textbookName: nav.textbookName,
            chapterId: nav.chapterId,
            chapterTitle: nav.chapterTitle,
          });
        });
      });
    });
    pane.querySelectorAll('button[data-slide-preview-se]').forEach(function (b) {
      b.addEventListener('click', function () {
        const id = Number(b.getAttribute('data-slide-preview-se'));
        openSlidePreviewAdmin(id);
      });
    });
  }

  function openSlideGenerateModal(mr, sectionId, sectionTitle, onDone) {
    const run = async function () {
      let defaultPrompt = '';
      try {
        const res = await api('/api/v1/admin/sections/' + sectionId + '/slide-generate/default-prompt');
        defaultPrompt = res.prompt || '';
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('加载默认提示词失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        return;
      }
      const cached = localStorage.getItem(slideGenPromptStorageKey(sectionId));
      const initial = cached != null && cached !== '' ? cached : defaultPrompt;

      mr.innerHTML = `
      <div class="modal-backdrop" id="sgd"><div class="modal" style="max-width:720px;width:95vw">
        <h3>生成幻灯片 · ${escapeHtml(sectionTitle || '小节 #' + sectionId)}</h3>
        <p class="muted" style="margin:8px 0 0">可编辑提示词。默认要求约 3～10 页、题目含标答与解析；生成成功后将写入幻灯片草稿并记入 AI 日志。</p>
        <div style="margin-top:12px"><label class="muted" style="display:block;margin-bottom:6px">Prompt</label>
        <textarea id="sgPrompt" rows="14" style="width:100%;box-sizing:border-box;font-family:ui-monospace,monospace;font-size:13px;padding:10px;border-radius:8px;border:1px solid #cbd5e1"></textarea></div>
        <div class="row" style="margin-top:12px;gap:8px;flex-wrap:wrap">
          <button type="button" class="btn" id="sgRun">生成</button>
          <button type="button" class="btn secondary" id="sgCancel">取消</button>
        </div>
      </div></div>`;
      mr.querySelector('#sgPrompt').value = initial;

      const close = function () {
        mr.innerHTML = '';
      };
      mr.querySelector('#sgCancel').addEventListener('click', close);
      mr.querySelector('#sgd').addEventListener('click', function (e) {
        if (e.target.id === 'sgd') close();
      });
      mr.querySelector('#sgRun').addEventListener('click', async function () {
        const ta = mr.querySelector('#sgPrompt');
        const prompt = ta ? ta.value : '';
        localStorage.setItem(slideGenPromptStorageKey(sectionId), prompt);
        try {
          mr.querySelector('#sgRun').disabled = true;
          await api('/api/v1/admin/sections/' + sectionId + '/slide-decks/generate-ai', {
            method: 'POST',
            jsonBody: { prompt: prompt },
          });
          alert('已生成幻灯片草稿（可稍后在试播中查看）。');
          close();
          if (typeof onDone === 'function') onDone();
        } catch (e) {
          if (authRedirectHandled(e)) return;
          const code = e.data && e.data.code ? e.data.code : e.message;
          const msg = e.data && e.data.message ? e.data.message : '';
          alert('生成失败: ' + code + (msg ? '\n' + msg : ''));
        } finally {
          const btn = mr.querySelector('#sgRun');
          if (btn) btn.disabled = false;
        }
      });
    };
    run();
  }

  function openSlidePreviewAdmin(sectionId) {
    const run = async function () {
      let list;
      try {
        list = await api('/api/v1/admin/sections/' + sectionId + '/slide-decks');
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('加载幻灯片列表失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        return;
      }
      const items = list.items || [];
      if (!items.length) {
        alert('幻灯片不存在：请先生成幻灯片。');
        return;
      }
      const pick = pickDeckForPreview(items);
      let deckRow;
      try {
        deckRow = await api('/api/v1/admin/slide-decks/' + pick.id);
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('加载幻灯片失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        return;
      }
      const deck = parseDeckContent(deckRow.content);
      if (!deck || !Array.isArray(deck.slides)) {
        alert('幻灯片内容无效或为空。');
        return;
      }
      if (typeof window.SlideDeckRenderer === 'undefined' || !window.SlideDeckRenderer.mount) {
        alert('幻灯片渲染器未加载。');
        return;
      }

      const overlay = document.createElement('div');
      overlay.className = 'admin-slide-preview-overlay';
      overlay.innerHTML =
        '<div class="admin-slide-preview-panel"><div class="toolbar row" style="justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px;margin-bottom:12px">' +
        '<strong>试播 · 小节 ' +
        sectionId +
        '</strong><button type="button" class="btn secondary" id="admSlidePrevClose">关闭</button></div>' +
        '<div class="admin-slide-preview-grid"><div id="admSlideStageHost"></div>' +
        '<div class="admin-slide-preview-log"><div class="muted" style="font-weight:600">动作日志</div>' +
        '<pre class="admin-slide-log-pre" id="admSlideLogPre"></pre></div></div></div>';

      document.body.appendChild(overlay);
      const host = overlay.querySelector('#admSlideStageHost');
      const logPre = overlay.querySelector('#admSlideLogPre');
      const deckTitle = (deck.meta && deck.meta.title) || deckRow.title || '';
      appendAdminSlideLog(logPre, '已加载 deck id=' + pick.id + ' · ' + deckTitle);

      let ctl = window.SlideDeckRenderer.mount(host, deck, {
        onNavigate: function (ctx) {
          appendAdminSlideLog(
            logPre,
            '翻页/步进 slideIndex=' +
              (ctx.slideIndex + 1) +
              ' slideId=' +
              (ctx.slideId || '—') +
              ' step=' +
              ctx.step +
              '/' +
              ctx.maxStep
          );
        },
        onAnswer: function (ctx) {
          appendAdminSlideLog(
            logPre,
            '作答 slideId=' + (ctx.slideId || '—') + ' key=' + ctx.questionKey + ' mode=' + ctx.mode + ' → ' + JSON.stringify(ctx.selected)
          );
        },
      });

      overlay.querySelector('#admSlidePrevClose').addEventListener('click', function () {
        if (ctl && typeof ctl.destroy === 'function') ctl.destroy();
        ctl = null;
        overlay.remove();
      });
      overlay.addEventListener('click', function (e) {
        if (e.target === overlay) {
          if (ctl && typeof ctl.destroy === 'function') ctl.destroy();
          ctl = null;
          overlay.remove();
        }
      });
    };
    run();
  }

  function openSectionEditModal(mr, srow, onSaved) {
    mr.innerHTML = `
      <div class="modal-backdrop" id="sbd"><div class="modal"><h3>编辑节</h3>
      <div class="form-grid">
        <div><label>序号</label><input id="senu" type="number" min="0" value="${srow.number}" /></div>
        <div style="grid-column:1/-1"><label>标题</label><input id="set" value="${escapeHtml(srow.title)}" /></div>
        <div style="grid-column:1/-1"><label>完整标题</label><input id="seft" value="${escapeHtml(
          srow.full_title || ''
        )}" /></div>
        <div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10('admSeStatus', srow.status, '激活', '未激活')}</div>
      </div>
      <div class="row"><button type="button" class="btn" id="seok">保存</button><button type="button" class="btn secondary" id="sex">取消</button></div>
      </div></div>`;
    const close = () => {
      mr.innerHTML = '';
    };
    mr.querySelector('#sex').addEventListener('click', close);
    mr.querySelector('#sbd').addEventListener('click', (e) => {
      if (e.target.id === 'sbd') close();
    });
    mr.querySelector('#seok').addEventListener('click', async () => {
      try {
        await api('/api/v1/admin/sections/' + srow.id, {
          method: 'PATCH',
          jsonBody: {
            number: Number(mr.querySelector('#senu').value),
            title: mr.querySelector('#set').value.trim(),
            full_title: mr.querySelector('#seft').value.trim(),
            status: readAdminStatus10(mr, 'admSeStatus'),
          },
        });
        close();
        onSaved();
      } catch (e) {
        if (authRedirectHandled(e)) return;
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
          )}</td>${tdAdminListStatus10(s.status)}
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
        ${
          edit
            ? `<div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10(
                'admStageStatus',
                ex.status,
                '激活',
                '未激活'
              )}</div>`
            : ''
        }
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
              status: readAdminStatus10(mr, 'admStageStatus'),
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
        if (authRedirectHandled(e)) return;
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
          )}</td><td>${escapeHtml(m.model != null && m.model !== '' ? m.model : m.app_key || '')}</td>${tdAdminListStatus10(m.status)}
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
            if (authRedirectHandled(e)) return;
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
        ${
          edit
            ? `<div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10(
                'admModelStatus',
                ex.status,
                '激活',
                '未激活'
              )}</div>`
            : ''
        }
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
          body.status = readAdminStatus10(mr, 'admModelStatus');
          await api('/api/v1/admin/ai-models/' + ex.id, { method: 'PATCH', jsonBody: body });
        } else {
          body.app_secret = sec;
          await api('/api/v1/admin/ai-models', { method: 'POST', jsonBody: body });
        }
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        if (authRedirectHandled(e)) return;
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
          )}</td>${tdAdminListStatus10(p.status)}
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
        <div style="grid-column:1/-1"><label>状态</label>${htmlAdminStatus10(
          'admPromptStatus',
          ex.status,
          '启用',
          '停用'
        )}</div>
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
            status: readAdminStatus10(mr, 'admPromptStatus'),
          },
        });
        close();
        mount(document.getElementById('app'));
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    });
  }

  function renderAICallLogs() {
    const f = state.aiLogFilters;
    const lim = Number(f.limit) || 50;
    const nextDisabled = state.aiLogs.length < lim;
    const rows = state.aiLogs
      .flatMap((e) => {
        const id = e.id;
        const head = `<tr class="ai-log-main" data-ailog-id="${id}" role="button" tabindex="0" aria-expanded="false" title="点击展开/收起详情">
        <td>${id}</td>
        <td class="ai-wrap" style="font-size:12px">${escapeHtml(e.created_at || '')}</td>
        <td class="ai-wrap">${escapeHtml(e.model_name_snapshot || '')}</td>
        <td>${escapeHtml(e.action || '')}</td>
        <td>${escapeHtml(e.outcome || '')}</td>
        <td class="ai-wrap">${escapeHtml(e.chat_model || '')}</td>
      </tr>`;
        const detail = `<tr class="ai-log-detail" data-ailog-detail="${id}" hidden><td colspan="6">${renderAICallDetailPanel(
          e
        )}</td></tr>`;
        return [head, detail];
      })
      .join('');
    return `
      <h2>AI 调用日志</h2>
      <p class="muted">脱敏请求/响应见展开区；适配器、用时(秒)、错误、Endpoint、mock 回退亦在展开区。点击行展开/收起。</p>
      <div class="ai-log-bar">
        <div class="ai-log-filters">
          <div class="ai-log-filter-grid">
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
          <input type="number" id="ailogLimit" min="1" max="200" value="${lim}" />
        </div>
          </div>
        </div>
        <div class="ai-log-pager">
          <button type="button" class="btn" id="ailogQuery">查询</button>
          <button type="button" class="btn secondary small" id="ailogPrev" ${f.offset <= 0 ? 'disabled' : ''}>上一页</button>
          <button type="button" class="btn secondary small" id="ailogNext" ${nextDisabled ? 'disabled' : ''}>下一页</button>
        </div>
      </div>
      <div class="table-wrap-ai">
      <table class="data ai-logs">
        <thead><tr>
          <th>ID</th><th>时间</th><th>模型名</th><th>动作</th><th>状态</th><th>chat</th>
        </tr></thead>
        <tbody>${rows || '<tr><td colspan="6">暂无</td></tr>'}</tbody>
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
    const toggleAiRow = (mainTr) => {
      const id = mainTr.getAttribute('data-ailog-id');
      if (!id) return;
      const detail = pane.querySelector(`tr.ai-log-detail[data-ailog-detail="${CSS.escape(String(id))}"]`);
      if (!detail) return;
      const open = !detail.hidden;
      detail.hidden = open;
      mainTr.setAttribute('aria-expanded', String(!open));
      mainTr.classList.toggle('ai-log-open', !open);
      try {
        mainTr.focus({ preventScroll: true });
      } catch (_) {}
    };
    pane.querySelector('table.ai-logs tbody')?.addEventListener('click', (ev) => {
      const mainTr = ev.target.closest('tr.ai-log-main');
      if (!mainTr) return;
      toggleAiRow(mainTr);
    });
    pane.querySelector('table.ai-logs tbody')?.addEventListener('keydown', (ev) => {
      if (ev.key !== 'Enter' && ev.key !== ' ') return;
      const mainTr = ev.target.closest('tr.ai-log-main');
      if (!mainTr) return;
      ev.preventDefault();
      toggleAiRow(mainTr);
    });
  }

  function renderAudit() {
    const f = state.auditFilters;
    const lim = Number(f.limit) || 50;
    const nextDisabled = state.audits.length < lim;
    const rows = state.audits
      .flatMap((e) => {
        const id = e.id;
        const head = `<tr class="audit-log-main" data-audit-id="${id}" role="button" tabindex="0" aria-expanded="false" title="点击展开/收起详情">
        <td>${id}</td>
        <td>${escapeHtml(e.user_type)}</td>
        <td>${e.user_id ?? '—'}</td>
        <td>${escapeHtml(e.action)}</td>
        <td>${escapeHtml(e.entity_type)}</td>
        <td>${e.entity_id ?? '—'}</td>
        <td>${escapeHtml(e.created_at || '')}</td>
      </tr>`;
        const detail = `<tr class="audit-log-detail" data-audit-detail="${id}" hidden><td colspan="7">${renderAuditDetailPanel(
          e
        )}</td></tr>`;
        return [head, detail];
      })
      .join('');
    return `
      <h2>审计日志</h2>
      <p class="muted">脱敏快照在展开区；点击行展开/收起。</p>
      <div class="ai-log-bar">
        <div class="ai-log-filters">
          <div class="ai-log-filter-grid">
            <div>
              <label>每页</label>
              <input type="number" id="auditLimit" min="1" max="500" value="${lim}" />
            </div>
          </div>
        </div>
        <div class="ai-log-pager">
          <button type="button" class="btn" id="auditQuery">查询</button>
          <button type="button" class="btn secondary small" id="auditPrev" ${f.offset <= 0 ? 'disabled' : ''}>上一页</button>
          <button type="button" class="btn secondary small" id="auditNext" ${nextDisabled ? 'disabled' : ''}>下一页</button>
        </div>
      </div>
      <div class="table-wrap-ai">
      <table class="data audit-logs">
        <thead><tr><th>ID</th><th>用户类型</th><th>用户ID</th><th>动作</th><th>实体</th><th>实体ID</th><th>时间</th></tr></thead>
        <tbody>${rows || '<tr><td colspan="7">暂无</td></tr>'}</tbody>
      </table></div>`;
  }

  function bindAuditLogs(pane) {
    const readAudit = (offsetVal) => ({
      limit: Math.min(500, Math.max(1, Number(pane.querySelector('#auditLimit').value) || 50)),
      offset: Math.max(0, offsetVal),
    });
    pane.querySelector('#auditQuery').addEventListener('click', () => {
      state.auditFilters = readAudit(0);
      mount(document.getElementById('app'));
    });
    pane.querySelector('#auditPrev').addEventListener('click', () => {
      const lim = state.auditFilters.limit || 50;
      state.auditFilters = readAudit(Math.max(0, (state.auditFilters.offset || 0) - lim));
      mount(document.getElementById('app'));
    });
    pane.querySelector('#auditNext').addEventListener('click', () => {
      const lim = state.auditFilters.limit || 50;
      state.auditFilters = readAudit((state.auditFilters.offset || 0) + lim);
      mount(document.getElementById('app'));
    });
    const toggleAudit = (mainTr) => {
      const id = mainTr.getAttribute('data-audit-id');
      if (!id) return;
      const detail = pane.querySelector(`tr.audit-log-detail[data-audit-detail="${CSS.escape(String(id))}"]`);
      if (!detail) return;
      const open = !detail.hidden;
      detail.hidden = open;
      mainTr.setAttribute('aria-expanded', String(!open));
      mainTr.classList.toggle('ai-log-open', !open);
      try {
        mainTr.focus({ preventScroll: true });
      } catch (_) {}
    };
    pane.querySelector('table.audit-logs tbody')?.addEventListener('click', (ev) => {
      const tr = ev.target.closest('tr.audit-log-main');
      if (!tr) return;
      toggleAudit(tr);
    });
    pane.querySelector('table.audit-logs tbody')?.addEventListener('keydown', (ev) => {
      if (ev.key !== 'Enter' && ev.key !== ' ') return;
      const tr = ev.target.closest('tr.audit-log-main');
      if (!tr) return;
      ev.preventDefault();
      toggleAudit(tr);
    });
  }

  function boot() {
    mount(document.getElementById('app'));
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', boot);
  else boot();
})();
