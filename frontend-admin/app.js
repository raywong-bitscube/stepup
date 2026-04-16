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

  /** 将 JSON/模型输出中的字面量实体先还原，再走 escapeHtml。多轮收敛以支持 &amp;gt; 等双重转义。 */
  function decodeHtmlEntities(s) {
    if (s == null || s === '') return '';
    let t = String(s);
    for (let pass = 0; pass < 10; pass++) {
      const before = t;
      t = t.replace(/&#x([0-9a-fA-F]+);/g, function (_, hex) {
        const c = parseInt(hex, 16);
        return c >= 0 && c <= 0x10ffff ? String.fromCodePoint(c) : _;
      });
      t = t.replace(/&#(\d+);/g, function (_, dec) {
        const c = parseInt(dec, 10);
        return c >= 0 && c <= 0x10ffff ? String.fromCodePoint(c) : _;
      });
      t = t.replace(/&lt;/gi, '<');
      t = t.replace(/&gt;/gi, '>');
      t = t.replace(/&quot;/gi, '"');
      t = t.replace(/&#0*39;/g, "'");
      t = t.replace(/&apos;/gi, "'");
      t = t.replace(/&nbsp;/g, ' ');
      t = t.replace(/&amp;/g, '&');
      if (t === before) break;
    }
    return t;
  }

  /** 极简 Markdown：换行、**粗体**、以 "- " 开头的行转为列表项（块级简单处理） */
  function renderSimpleMarkdown(text) {
    if (text == null) return '';
    const raw = decodeHtmlEntities(String(text));
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
        html += '<li>' + renderMarkdownSegment(escapeHtml(listM[1])) + '</li>';
      } else {
        closeList();
        if (line.trim() === '') {
          html += '<br/>';
        } else {
          html += '<p class="slide-md-p">' + renderMarkdownSegment(escapeHtml(line)) + '</p>';
        }
      }
    }
    closeList();
    return html || '<p class="slide-md-p">—</p>';
  }

  function inlineFormat(escapedLine) {
    return escapedLine.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  }

  /** 已 escapeHtml 的片段：$$块级公式$$，再 $行内$，最后 **粗体**（仅作用于非公式段）。 */
  function renderMarkdownSegment(escapedText) {
    if (escapedText == null || escapedText === '') return '';
    if (escapedText.indexOf('$$') >= 0) {
      const segs = escapedText.split('$$');
      let out = '';
      for (let j = 0; j < segs.length; j++) {
        if (j % 2 === 0) {
          out += injectInlineDollarMath(segs[j]);
        } else {
          const tex = segs[j].trim();
          out += '<div class="slide-math-block">' + renderLatex(tex, false) + '</div>';
        }
      }
      return out;
    }
    return injectInlineDollarMath(escapedText);
  }

  function injectInlineDollarMath(escapedText) {
    if (escapedText.indexOf('$') < 0) return inlineFormat(escapedText);
    const parts = escapedText.split('$');
    if (parts.length === 1) return inlineFormat(escapedText);
    let html = '';
    for (let i = 0; i < parts.length; i++) {
      if (i % 2 === 0) {
        html += inlineFormat(parts[i]);
      } else {
        const tex = parts[i].trim();
        if (tex === '') html += '$';
        else
          html += '<span class="slide-math-inline">' + renderLatex(tex, 'inline') + '</span>';
      }
    }
    return html;
  }

  function renderLatex(tex, display) {
    const t = decodeHtmlEntities(String(tex || ''));
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
      const alt = el.alt != null ? decodeHtmlEntities(String(el.alt)) : '';
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

  function assetURL(raw) {
    const t = String(raw || '').trim();
    if (!t) return '';
    if (/^https?:\/\//i.test(t) || t.startsWith('data:') || t.startsWith('blob:')) return t;
    if (t.startsWith('/')) {
      const base = apiBase().replace(/\/$/, '');
      return base + t;
    }
    return t;
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

  /**
   * 长耗时请求 UI（与学生端「生成中…」「识别中…」一致）：主按钮改文案并禁用，锁定表单控件，禁用取消，backdrop 标记 data-busy（点击遮罩不误关）。
   */
  function setAdminModalLongTask(modalContainer, busy, o) {
    o = o || {};
    const idleLabel = o.idleLabel || '确定';
    const busyLabel = o.busyLabel || '处理中…';
    const bd = o.backdropSel ? modalContainer.querySelector(o.backdropSel) : modalContainer.querySelector('.modal-backdrop');
    const submit = o.submitSel ? modalContainer.querySelector(o.submitSel) : null;
    const cancel = o.cancelSel ? modalContainer.querySelector(o.cancelSel) : null;
    const lock = (o.lockSelectors || []).map(function (sel) {
      return modalContainer.querySelector(sel);
    }).filter(Boolean);
    if (busy) {
      if (bd) bd.setAttribute('data-busy', '1');
      if (submit) {
        if (!submit.dataset.adminIdleLabel) {
          submit.dataset.adminIdleLabel = (submit.textContent || '').trim() || idleLabel;
        }
        submit.textContent = busyLabel;
        submit.disabled = true;
      }
      if (cancel) cancel.disabled = true;
      lock.forEach(function (el) {
        el.readOnly = true;
        el.setAttribute('aria-busy', 'true');
      });
    } else {
      if (bd) bd.removeAttribute('data-busy');
      if (submit) {
        submit.textContent = submit.dataset.adminIdleLabel || idleLabel;
        submit.disabled = false;
        delete submit.dataset.adminIdleLabel;
      }
      if (cancel) cancel.disabled = false;
      lock.forEach(function (el) {
        el.readOnly = false;
        el.removeAttribute('aria-busy');
      });
    }
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
        'sys_user_id',
        e.sys_user_id != null && e.sys_user_id !== '' ? String(e.sys_user_id) : ''
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
    examPapers: [],
    examSourceDetailPaperID: 0,
    aiLogFilters: {
      limit: 50,
      offset: 0,
      ai_provider_model_id: '',
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
    if (opts.body != null) init.body = opts.body;
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
    const doLogin = async () => {
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
    };
    b.addEventListener('click', doLogin);
    root.querySelector('#pass')?.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter') return;
      e.preventDefault();
      doLogin();
    });
    root.querySelector('#user')?.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter') return;
      e.preventDefault();
      doLogin();
    });
  }

  function renderAppShell() {
    const flash = state.flash
      ? `<div class="flash ${state.flash.kind}">${escapeHtml(state.flash.msg)}</div>`
      : '';
    const nav = [
      ['dashboard', '仪表盘'],
      ['students', '学生'],
      ['exam_source', '试卷库'],
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
      if (state.view === 'exam_source') {
        const d = await api('/api/v1/admin/exam-source/papers');
        state.examPapers = d.items || [];
        pane.innerHTML = renderExamSourcePapers();
        bindExamSourcePapers(pane);
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
        if (f.ai_provider_model_id) q.set('ai_provider_model_id', f.ai_provider_model_id);
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
        <td class="td-actions"><span class="td-actions-inner"><button type="button" class="btn small" data-sid="${s.id}">编辑</button>${
          (s.textbook_count || 0) > 0
            ? `<button type="button" class="btn small secondary" data-catalog-sid="${s.id}">目录</button>`
            : ''
        }</span></td></tr>`
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
        <td class="td-actions"><span class="td-actions-inner">
          <button type="button" class="btn small" data-edit-tb="${t.id}">编辑</button>
          <button type="button" class="btn small secondary" data-chapters-tb="${t.id}">章节</button>
        </span></td></tr>`
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
        <td class="td-actions"><span class="td-actions-inner">
          <button type="button" class="btn small" data-edit-ch="${c.id}">编辑</button>
          <button type="button" class="btn small secondary" data-sects-ch="${c.id}">小节</button>
        </span></td></tr>`
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
        <td class="td-actions"><span class="td-actions-inner"><button type="button" class="btn small" data-edit-se="${s.id}">编辑</button><button type="button" class="btn small secondary" data-slide-gen-se="${s.id}">生成幻灯片</button><button type="button" class="btn small secondary" data-slide-preview-se="${s.id}">试播</button></span></td></tr>`
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
        <p class="muted" style="margin:8px 0 0">可编辑提示词。默认约 10～20 页、多例题与讲解；题目须含标答与解析。生成可能需数十秒，请点击后稍候。</p>
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
        if (e.target.id === 'sgd') {
          if (mr.querySelector('#sgd') && mr.querySelector('#sgd').getAttribute('data-busy') === '1') return;
          close();
        }
      });
      mr.querySelector('#sgRun').addEventListener('click', async function () {
        const ta = mr.querySelector('#sgPrompt');
        const prompt = ta ? ta.value : '';
        localStorage.setItem(slideGenPromptStorageKey(sectionId), prompt);
        const longTaskOpt = {
          backdropSel: '#sgd',
          submitSel: '#sgRun',
          cancelSel: '#sgCancel',
          lockSelectors: ['#sgPrompt'],
          idleLabel: '生成',
          busyLabel: '生成中…',
        };
        try {
          setAdminModalLongTask(mr, true, longTaskOpt);
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
          if (mr.querySelector('#sgRun')) setAdminModalLongTask(mr, false, longTaskOpt);
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
          <input type="text" id="ailogModel" placeholder="ai_provider_model.id" value="${escapeHtml(f.ai_provider_model_id)}" />
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
      ai_provider_model_id: pane.querySelector('#ailogModel').value.trim(),
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

  function renderExamSourceAnalyzeGroups(analyzed) {
    const groups = analyzed && Array.isArray(analyzed.groups) ? analyzed.groups : [];
    if (!groups.length) {
      return '<p class="muted" style="margin:0">大题分组：本次识别未返回分组（不影响建卷；有模型时可显示卷面大题说明）。</p>';
    }
    const blocks = groups
      .map((g) => {
        const ord = g.group_order != null ? g.group_order : '—';
        const sk = escapeHtml(String(g.system_kind || ''));
        const tl = g.title_label ? ' · ' + escapeHtml(String(g.title_label)) : '';
        const desc = escapeHtml(String(g.description_text || ''));
        return `<div class="es-analyze-group"><span class="es-analyze-group-head"><strong>大题 ${ord}</strong> <span class="muted">${sk}</span>${tl}</span><div class="es-analyze-group-desc">${desc || '—'}</div></div>`;
      })
      .join('');
    return `<label>识别到的大题分组（只读）</label><div class="es-analyze-groups">${blocks}</div>`;
  }

  function renderExamSourcePapers() {
    if (Number(state.examSourceDetailPaperID || 0) > 0) {
      return `
      <div class="toolbar">
        <h2 style="margin:0">试卷详情</h2>
        <div class="row" style="margin:0">
          <button type="button" class="btn secondary" id="btnExamBackList">返回试卷列表</button>
        </div>
      </div>
      <div id="detailRoot"></div>`;
    }
    const rows = (state.examPapers || [])
      .map((p) => {
        const total = p.total_score == null ? '—' : p.total_score;
        const yr = p.exam_year == null ? '—' : p.exam_year;
        return `<tr>
          <td>${p.id}</td>
          <td><a href="#" data-es-open="${p.id}">${escapeHtml(p.title || '')}</a></td>
          <td>${yr}</td>
          <td>${escapeHtml(p.term || '—')}</td>
          <td>${p.k12_subject_id || '—'}</td>
          <td>${escapeHtml(String(total))}</td>
          ${tdAdminListStatus10(p.status)}
        </tr>`;
      })
      .join('');
    return `
      <div class="toolbar">
        <h2 style="margin:0">试卷库（exam_source）</h2>
        <div class="row" style="margin:0">
          <button type="button" class="btn secondary" id="btnExamUpload">多图上传建卷</button>
        </div>
      </div>
      <p class="muted">支持浏览整卷信息，并通过“两阶段上传（先分析、再补全）”创建试卷及页面子记录。</p>
      <table class="data">
        <thead><tr><th>ID</th><th>标题</th><th>年份</th><th>学期</th><th>学科ID</th><th>总分</th><th>状态</th></tr></thead>
        <tbody>${rows || '<tr><td colspan="7">暂无数据</td></tr>'}</tbody>
      </table>
      <div id="detailRoot" style="margin-top:14px"></div>`;
  }

  function bindExamSourcePapers(pane) {
    const detailRoot = pane.querySelector('#detailRoot');
    pane.querySelector('#btnExamUpload')?.addEventListener('click', () => openExamSourceUploadModal(detailRoot));
    pane.querySelector('#btnExamBackList')?.addEventListener('click', () => {
      state.examSourceDetailPaperID = 0;
      mount(document.getElementById('app'));
    });
    pane.querySelectorAll('a[data-es-open]').forEach((a) => {
      a.addEventListener('click', (e) => {
        e.preventDefault();
        const id = Number(a.getAttribute('data-es-open'));
        if (!id) return;
        state.examSourceDetailPaperID = id;
        mount(document.getElementById('app'));
      });
    });
    const detailPaperID = Number(state.examSourceDetailPaperID || 0);
    if (detailRoot && detailPaperID > 0) {
      openExamSourceDetailModal(detailRoot, detailPaperID, {
        closeLabel: '返回试卷列表',
        onClose: () => {
          state.examSourceDetailPaperID = 0;
          mount(document.getElementById('app'));
        },
      });
    }
  }

  function openExamSourceUploadModal(mr) {
    const close = () => {
      mr.innerHTML = '';
    };
    let selectedFiles = [];
    let analyzed = null;
    let bboxOpt = { debugBBox: false, disableInset: false, disableNextClamp: false };

    const appendBBoxOptions = (fd, opt) => {
      if (!fd || !opt) return;
      if (opt.debugBBox) fd.append('debug_bbox', '1');
      if (opt.disableInset) fd.append('bbox_disable_inset', '1');
      if (opt.disableNextClamp) fd.append('bbox_disable_next_clamp', '1');
    };

    const bindBackdropClose = () => {
      mr.querySelector('#esUpBd')?.addEventListener('click', (e) => {
        if (e.target.id === 'esUpBd') close();
      });
      mr.querySelector('#esuCancel')?.addEventListener('click', close);
    };

    const renderStepAnalyze = () => {
      mr.innerHTML = `
      <div class="modal-backdrop" id="esUpBd"><div class="modal" style="max-width:760px;width:95vw">
        <h3>多图上传建卷（第 1 步：上传并分析）</h3>
        <p class="muted">先上传整卷图片，系统自动分析题号与试卷信息；下一步可人工补全和修改后再正式建卷。</p>
        <div class="form-grid">
          <div style="grid-column:1/-1"><label>标题提示（可选）</label><input id="esuHintTitle" placeholder="可留空，系统会自动识别标题" /></div>
          <div style="grid-column:1/-1"><label>图片文件（可多选） *</label><input id="esuFiles" type="file" multiple accept="image/*" /></div>
          <div style="grid-column:1/-1">
            <label style="margin-bottom:6px">bbox 调试/开关（仅用于排查）</label>
            <div class="row" style="margin:0;gap:12px;flex-wrap:wrap">
              <label style="margin:0"><input id="esuDbgBBox" type="checkbox" ${bboxOpt.debugBBox ? 'checked' : ''} /> 返回调试信息</label>
              <label style="margin:0"><input id="esuNoInset" type="checkbox" ${bboxOpt.disableInset ? 'checked' : ''} /> 关闭 inset 收缩</label>
              <label style="margin:0"><input id="esuNoClamp" type="checkbox" ${bboxOpt.disableNextClamp ? 'checked' : ''} /> 关闭邻题底边截断</label>
            </div>
          </div>
        </div>
        <div class="row"><button type="button" class="btn" id="esuAnalyze">上传并分析</button><button type="button" class="btn secondary" id="esuCancel">取消</button></div>
      </div></div>`;
      bindBackdropClose();
      mr.querySelector('#esuAnalyze')?.addEventListener('click', async () => {
        const filesInput = mr.querySelector('#esuFiles');
        const files = filesInput && filesInput.files ? Array.from(filesInput.files) : [];
        if (!files.length) {
          alert('请至少选择一张图片');
          return;
        }
        selectedFiles = files;
        const fd = new FormData();
        const hintTitle = (mr.querySelector('#esuHintTitle').value || '').trim();
        if (hintTitle) fd.append('title', hintTitle);
        bboxOpt = {
          debugBBox: !!mr.querySelector('#esuDbgBBox')?.checked,
          disableInset: !!mr.querySelector('#esuNoInset')?.checked,
          disableNextClamp: !!mr.querySelector('#esuNoClamp')?.checked,
        };
        appendBBoxOptions(fd, bboxOpt);
        selectedFiles.forEach((f) => fd.append('images', f));
        setAdminModalLongTask(mr, true, {
          backdropSel: '#esUpBd',
          submitSel: '#esuAnalyze',
          cancelSel: '#esuCancel',
          idleLabel: '上传并分析',
          busyLabel: '分析中…',
          lockSelectors: ['#esuHintTitle', '#esuFiles'],
        });
        try {
          analyzed = await api('/api/v1/admin/exam-source/papers/upload-analyze', { method: 'POST', body: fd });
          renderStepSubmit();
        } catch (e) {
          if (authRedirectHandled(e)) return;
          alert('分析失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        } finally {
          setAdminModalLongTask(mr, false, {
            backdropSel: '#esUpBd',
            submitSel: '#esuAnalyze',
            cancelSel: '#esuCancel',
            idleLabel: '上传并分析',
            busyLabel: '分析中…',
            lockSelectors: ['#esuHintTitle', '#esuFiles'],
          });
        }
      });
    };

    const renderStepSubmit = () => {
      const s = (analyzed && analyzed.suggested) || {};
      const subjList = Array.isArray(state.catalog && state.catalog.subjects) ? state.catalog.subjects : [];
      const subjOpt = ['<option value="">请选择学科</option>']
        .concat(
          subjList.map((x) => {
            const id = Number(x.id || 0);
            const nm = String(x.name || x.title || ('学科#' + id));
            const selected = Number(s.k12_subject_id || 0) === id ? 'selected' : '';
            return `<option value="${id}" ${selected}>${escapeHtml(nm)} (#${id})</option>`;
          })
        )
        .join('');
      const qnos = Array.isArray(analyzed && analyzed.question_nos) ? analyzed.question_nos.join(',') : '';
      const suggestedSubject = s.suggested_subject ? `（识别建议：${escapeHtml(String(s.suggested_subject))}）` : '';
      const debugBlock =
        analyzed && analyzed.debug
          ? `<details style="grid-column:1/-1"><summary>bbox 调试信息（页面尺寸 / AI框 / 最终框）</summary><pre class="raw" style="max-height:280px;overflow:auto">${escapeHtml(
              JSON.stringify(analyzed.debug, null, 2)
            )}</pre></details>`
          : '';
      mr.innerHTML = `
      <div class="modal-backdrop" id="esUpBd"><div class="modal" style="max-width:820px;width:96vw">
        <h3>多图上传建卷（第 2 步：确认并补全）</h3>
        <p class="muted">识别结果已回填，你可以修改任意字段后再提交建卷。</p>
        <div class="form-grid">
          <div><label>标题 *</label><input id="esuTitle" value="${escapeHtml(String(s.title || ''))}" /></div>
          <div><label>学科 *</label><select id="esuSubj">${subjOpt}</select></div>
          <div><label>年级ID（可选）</label><input id="esuGrade" type="number" min="1" value="${s.k12_grade_id || ''}" /></div>
          <div><label>年份</label><input id="esuYear" type="number" min="2000" max="2100" value="${s.exam_year || ''}" /></div>
          <div><label>学期</label><input id="esuTerm" placeholder="如：二模" value="${escapeHtml(String(s.term || ''))}" /></div>
          <div><label>总分</label><input id="esuScore" value="${escapeHtml(String(s.total_score || ''))}" /></div>
          <div><label>时长(分钟)</label><input id="esuDuration" type="number" min="1" value="${s.duration_minutes || ''}" /></div>
          <div><label>地区</label><input id="esuRegion" value="${escapeHtml(String(s.source_region || ''))}" /></div>
          <div><label>学校</label><input id="esuSchool" value="${escapeHtml(String(s.source_school || ''))}" /></div>
          <div><label>年级标签</label><input id="esuGradeLabel" value="${escapeHtml(String(s.grade_label || ''))}" /></div>
          <div><label>试卷类型</label><input id="esuPaperType" value="${escapeHtml(String(s.paper_type || 'mock_exam'))}" /></div>
          <div style="grid-column:1/-1"><label>题号列表（可改）</label><input id="esuQNos" placeholder="1,2,3,4,5..." value="${escapeHtml(qnos)}" /></div>
          <div style="grid-column:1/-1">
            <label style="margin-bottom:6px">bbox 调试/开关（建卷时会继续生效）</label>
            <div class="row" style="margin:0;gap:12px;flex-wrap:wrap">
              <label style="margin:0"><input id="esuDbgBBox2" type="checkbox" ${bboxOpt.debugBBox ? 'checked' : ''} /> 返回调试信息</label>
              <label style="margin:0"><input id="esuNoInset2" type="checkbox" ${bboxOpt.disableInset ? 'checked' : ''} /> 关闭 inset 收缩</label>
              <label style="margin:0"><input id="esuNoClamp2" type="checkbox" ${bboxOpt.disableNextClamp ? 'checked' : ''} /> 关闭邻题底边截断</label>
            </div>
          </div>
          <div style="grid-column:1/-1" id="esuGroupsWrap">${renderExamSourceAnalyzeGroups(analyzed)}</div>
          ${debugBlock}
          <div style="grid-column:1/-1"><p class="muted" style="margin:0">已上传图片数：${selectedFiles.length} ${suggestedSubject}</p></div>
        </div>
        <div class="row">
          <button type="button" class="btn secondary" id="esuBack">返回上一步</button>
          <button type="button" class="btn" id="esuSubmit">确认建卷</button>
          <button type="button" class="btn secondary" id="esuCancel">取消</button>
        </div>
      </div></div>`;
      bindBackdropClose();
      mr.querySelector('#esuBack')?.addEventListener('click', renderStepAnalyze);
      mr.querySelector('#esuSubmit')?.addEventListener('click', async () => {
        if (!selectedFiles.length) {
          alert('缺少上传图片，请返回上一步重新上传');
          return;
        }
        const title = (mr.querySelector('#esuTitle').value || '').trim();
        const subj = Number(mr.querySelector('#esuSubj').value || 0);
        if (!title || !subj) {
          alert('请至少填写标题并选择学科');
          return;
        }
        const fd = new FormData();
        fd.append('title', title);
        fd.append('k12_subject_id', String(subj));
        const putOpt = (k, v) => {
          const t = String(v || '').trim();
          if (t) fd.append(k, t);
        };
        putOpt('k12_grade_id', mr.querySelector('#esuGrade').value);
        putOpt('exam_year', mr.querySelector('#esuYear').value);
        putOpt('term', mr.querySelector('#esuTerm').value);
        putOpt('total_score', mr.querySelector('#esuScore').value);
        putOpt('duration_minutes', mr.querySelector('#esuDuration').value);
        putOpt('source_region', mr.querySelector('#esuRegion').value);
        putOpt('source_school', mr.querySelector('#esuSchool').value);
        putOpt('grade_label', mr.querySelector('#esuGradeLabel').value);
        putOpt('paper_type', mr.querySelector('#esuPaperType').value);
        putOpt('question_nos', mr.querySelector('#esuQNos').value);
        bboxOpt = {
          debugBBox: !!mr.querySelector('#esuDbgBBox2')?.checked,
          disableInset: !!mr.querySelector('#esuNoInset2')?.checked,
          disableNextClamp: !!mr.querySelector('#esuNoClamp2')?.checked,
        };
        appendBBoxOptions(fd, bboxOpt);
        selectedFiles.forEach((f) => fd.append('images', f));
        setAdminModalLongTask(mr, true, {
          backdropSel: '#esUpBd',
          submitSel: '#esuSubmit',
          cancelSel: '#esuCancel',
          idleLabel: '确认建卷',
          busyLabel: '建卷中…',
          lockSelectors: [
            '#esuTitle',
            '#esuSubj',
            '#esuGrade',
            '#esuYear',
            '#esuTerm',
            '#esuScore',
            '#esuDuration',
            '#esuRegion',
            '#esuSchool',
            '#esuGradeLabel',
            '#esuPaperType',
            '#esuQNos',
          ],
        });
        const backBtn = mr.querySelector('#esuBack');
        if (backBtn) backBtn.disabled = true;
        try {
          await api('/api/v1/admin/exam-source/papers/upload', { method: 'POST', body: fd });
          close();
          mount(document.getElementById('app'));
        } catch (e) {
          if (authRedirectHandled(e)) return;
          alert('建卷失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        } finally {
          setAdminModalLongTask(mr, false, {
            backdropSel: '#esUpBd',
            submitSel: '#esuSubmit',
            cancelSel: '#esuCancel',
            idleLabel: '确认建卷',
            busyLabel: '建卷中…',
            lockSelectors: [
              '#esuTitle',
              '#esuSubj',
              '#esuGrade',
              '#esuYear',
              '#esuTerm',
              '#esuScore',
              '#esuDuration',
              '#esuRegion',
              '#esuSchool',
              '#esuGradeLabel',
              '#esuPaperType',
              '#esuQNos',
            ],
          });
          if (backBtn) backBtn.disabled = false;
        }
      });
    };

    renderStepAnalyze();
  }

  function openExamSourceDetailModal(host, paperId, opts) {
    const options = opts || {};
    const closeLabel = String(options.closeLabel || '收起');
    const close =
      typeof options.onClose === 'function'
        ? options.onClose
        : () => {
            host.innerHTML = '';
          };
    const openImageLightbox = (imgURL) => {
      const u = String(imgURL || '').trim();
      if (!u) return;
      const mask = document.createElement('div');
      mask.className = 'es-image-lightbox';
      mask.innerHTML = `<div class="es-image-lightbox-inner"><img src="${escapeHtml(u)}" alt="preview" class="es-image-lightbox-img" /></div>`;
      mask.addEventListener('click', (e) => {
        if (e.target === mask) mask.remove();
      });
      host.appendChild(mask);
    };

    const load = async () => {
      host.innerHTML = `<div class="card es-inline-detail"><h3 style="margin:0">试卷详情 #${paperId}</h3><p class="muted">加载中…</p></div>`;
      try {
        const d = await api('/api/v1/admin/exam-source/papers/' + paperId);
        let preview = null;
        try {
          preview = await api('/api/v1/admin/exam-source/papers/' + paperId + '/recognition-preview');
        } catch {
          preview = null;
        }
        const p = d.paper || {};
        const pages = d.pages || [];
        const qs = d.questions || [];
        const qgroups = (d.question_groups || []).slice().sort((a, b) => (a.group_order || 0) - (b.group_order || 0));
        const pvQs = (preview && preview.questions) || [];
        const previewQByID = new Map();
        pvQs.forEach((x) => {
          if (x && x.id) previewQByID.set(Number(x.id), x);
        });
        const pageRows = pages
          .map(
            (x) =>
              `<tr><td>${x.page_no}</td><td>${x.file_id}</td><td>${
                x.public_url
                  ? `<button type="button" class="btn secondary small es-img-open" data-img="${escapeHtml(assetURL(x.public_url))}">查看</button>`
                  : '—'
              }</td></tr>`
          )
          .join('');

        const collectQuestionImageURLs = (pq) => {
          const out = [];
          const seen = new Set();
          const push = (u) => {
            const t = String(u || '').trim();
            if (!t) return;
            const url = assetURL(t);
            if (!url || seen.has(url)) return;
            seen.add(url);
            out.push(url);
          };
          if (pq && Array.isArray(pq.image_urls)) pq.image_urls.forEach(push);
          if (pq && Array.isArray(pq.stem_crop_public_urls)) pq.stem_crop_public_urls.forEach(push);
          if (pq && Array.isArray(pq.crop_public_urls)) pq.crop_public_urls.forEach(push);
          if (pq && Array.isArray(pq.stem_images)) {
            pq.stem_images.forEach((it) => {
              if (typeof it === 'string') push(it);
              else if (it && typeof it === 'object') push(it.public_url || it.url);
            });
          }
          push(pq && pq.stem_crop_public_url);
          return out;
        };

        const renderQuestionBlock = (q) => {
          const pq = previewQByID.get(Number(q.id)) || {};
          const imgURLs = collectQuestionImageURLs(pq);
          const imgGrid = imgURLs.length
            ? `<div class="es-question-images-grid">${imgURLs
                .map(
                  (u) =>
                    `<button type="button" class="es-question-image-item es-img-open" data-img="${escapeHtml(u)}"><img src="${escapeHtml(
                      u
                    )}" alt="question-image" class="es-question-image" /></button>`
                )
                .join('')}</div>`
            : '<div class="es-question-empty">暂无题图</div>';
          return `<article class="es-question-item">
              <p class="es-question-stem">[${escapeHtml(q.question_no || '—')}] ${escapeHtml(q.stem_text || '—')}</p>
              <div class="es-question-section"><span class="es-question-label">图:</span><button type="button" class="btn secondary small es-q-bbox-btn" data-qid="${q.id}">校正bbox</button>${imgGrid}</div>
              <p class="es-question-section"><span class="es-question-label">答案:</span>${escapeHtml(q.answer_text || '—')}</p>
              <p class="es-question-section"><span class="es-question-label">解析:</span>${escapeHtml(q.explanation_text || '—')}</p>
            </article>`;
        };

        let questionBlocks = '';
        if (qgroups.length) {
          for (const g of qgroups) {
            const gid = Number(g.id);
            const sub = qs.filter((q) => Number(q.group_id) === gid);
            const headLine = `大题 ${g.group_order != null ? g.group_order : '—'} · ${escapeHtml(String(g.system_kind || ''))}${
              g.title_label ? ' · ' + escapeHtml(String(g.title_label)) : ''
            }`;
            const desc = escapeHtml(String(g.description_text || '—'));
            const qContent = sub.length ? sub.map(renderQuestionBlock).join('') : '<p class="muted">该大题下暂无题目记录</p>';
            questionBlocks += `<section class="es-qgroup"><h4 class="es-qgroup-title">${headLine}</h4><p class="muted es-qgroup-desc">${desc}</p>${qContent}</section>`;
          }
          const ungrouped = qs.filter((q) => !q.group_id);
          if (ungrouped.length) {
            questionBlocks += `<section class="es-qgroup"><h4 class="es-qgroup-title">未分组题目</h4>${ungrouped.map(renderQuestionBlock).join('')}</section>`;
          }
        } else {
          questionBlocks = qs.length ? qs.map(renderQuestionBlock).join('') : '<p class="muted">暂无题目</p>';
        }

        host.innerHTML = `
          <div class="card es-inline-detail">
            <div class="toolbar"><h3 style="margin:0">试卷详情 #${paperId}</h3><div class="row" style="margin:0"><button type="button" class="btn secondary" id="esDtlPurge">彻底删除</button><button type="button" class="btn secondary" id="esDtlClose">${escapeHtml(closeLabel)}</button></div></div>
            <p class="muted">标题：${escapeHtml(p.title || '')}｜学科ID：${p.k12_subject_id || '—'}｜页数：${p.page_count || 0}｜题数：${p.question_count || 0}</p>
            <h4>试卷原图</h4>
            <table class="data"><thead><tr><th>页码</th><th>文件ID</th><th>图片</th></tr></thead><tbody>${pageRows || '<tr><td colspan="3">暂无页面</td></tr>'}</tbody></table>
            <h4 style="margin-top:14px">题目</h4>
            <p class="muted" style="margin-top:4px">按卷面大题分组展示说明与题目，题目内容与图片按详情形式完整展示。</p>
            ${questionBlocks}
          </div>`;

        host.querySelector('#esDtlPurge')?.addEventListener('click', async () => {
          if (!confirm('将彻底删除该试卷及其关联数据（不可恢复），确认继续？')) return;
          if (!confirm(`再次确认：彻底删除试卷 #${paperId} ?`)) return;
          const btn = host.querySelector('#esDtlPurge');
          if (btn) btn.disabled = true;
          try {
            await api('/api/v1/admin/exam-source/papers/' + paperId + '/purge', { method: 'DELETE' });
            close();
            mount(document.getElementById('app'));
          } catch (e) {
            if (authRedirectHandled(e)) return;
            alert('删除失败: ' + (e.data && e.data.code ? e.data.code : e.message));
          } finally {
            if (btn) btn.disabled = false;
          }
        });
        host.querySelector('#esDtlClose')?.addEventListener('click', close);
        host.querySelectorAll('.es-q-bbox-btn').forEach((btn) => {
          btn.addEventListener('click', () => {
            const qid = Number(btn.getAttribute('data-qid') || 0);
            openExamSourceBboxModal(host, paperId, qid);
          });
        });
        host.querySelectorAll('.es-img-open').forEach((btn) => {
          btn.addEventListener('click', () => {
            const u = btn.getAttribute('data-img');
            if (!u) return;
            openImageLightbox(u);
          });
        });
      } catch (e) {
        if (authRedirectHandled(e)) return;
        host.innerHTML = `<div class="card es-inline-detail"><h3 style="margin:0">试卷详情 #${paperId}</h3><p class="muted">加载失败</p><div class="row"><button type="button" class="btn secondary" id="esDtlClose">${escapeHtml(
          closeLabel
        )}</button></div></div>`;
        host.querySelector('#esDtlClose')?.addEventListener('click', close);
        alert('加载详情失败: ' + (e.data && e.data.code ? e.data.code : e.message));
      }
    };
    load();
  }

  function openExamSourceBboxModal(mr, paperId, initialQuestionID) {
    const goBack = () => {
      openExamSourceDetailModal(mr, paperId);
    };
    const shell = (inner, busy) => `
      <div class="modal-backdrop" id="esBboxBd" data-busy="${busy ? '1' : '0'}"><div class="modal xwide" style="width:98vw;max-height:92vh">
        <div class="toolbar">
          <h3 style="margin:0">识别预览 / 题干 bbox 校正 — 试卷 #${paperId}</h3>
          <div class="row" style="margin:0"><button type="button" class="btn secondary" id="esBboxClose">返回</button></div>
        </div>
        ${inner}
      </div></div>`;
    mr.innerHTML = shell('<p class="muted">加载中…</p>', true);
    mr.querySelector('#esBboxClose')?.addEventListener('click', goBack);
    mr.querySelector('#esBboxBd')?.addEventListener('click', (e) => {
      if (e.target.id === 'esBboxBd') goBack();
    });

    (async () => {
      let data;
      try {
        data = await api('/api/v1/admin/exam-source/papers/' + paperId + '/recognition-preview');
      } catch (e) {
        if (authRedirectHandled(e)) return;
        alert('加载预览失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        goBack();
        return;
      }
      const pages = data.pages || [];
      const questions = data.questions || [];
      if (!pages.length) {
        mr.innerHTML = shell(
          '<p class="muted">该试卷暂无页面图片，请先上传建卷。</p><div class="row"><button type="button" class="btn secondary" id="esBboxClose2">返回</button></div>',
          false
        );
        mr.querySelector('#esBboxClose2')?.addEventListener('click', goBack);
        mr.querySelector('#esBboxBd')?.addEventListener('click', (e) => {
          if (e.target.id === 'esBboxBd') goBack();
        });
        return;
      }
      if (!questions.length) {
        mr.innerHTML = shell(
          '<p class="muted">暂无题目记录。可先上传并填写题号列表生成题目，再校正 bbox。</p><div class="row"><button type="button" class="btn secondary" id="esBboxClose2">返回</button></div>',
          false
        );
        mr.querySelector('#esBboxClose2')?.addEventListener('click', goBack);
        mr.querySelector('#esBboxBd')?.addEventListener('click', (e) => {
          if (e.target.id === 'esBboxBd') goBack();
        });
        return;
      }

      const pageOptions = pages
        .map((p) => `<option value="${p.page_no}">第 ${p.page_no} 页</option>`)
        .join('');
      const qOptions = questions
        .map((q) => `<option value="${q.id}">#${escapeHtml(q.question_no || '')} (id ${q.id})</option>`)
        .join('');

      mr.innerHTML = shell(
        `
        <p class="muted">在整页图上查看/修改题干区域（归一化坐标 0～1）。保存后会重新裁剪题干图并更新记录。</p>
        <div class="es-bbox-grid">
          <div class="es-bbox-stage-wrap">
            <div class="es-bbox-stage" id="esBboxStage">
              <img id="esBboxImg" alt="page" />
              <div id="esBboxOverlay" class="es-bbox-overlay"></div>
            </div>
          </div>
          <div>
            <div><label>题目</label><select id="esBboxQ">${qOptions}</select></div>
            <div style="margin-top:10px"><label>页面</label><select id="esBboxPage">${pageOptions}</select></div>
            <div style="margin-top:10px;display:grid;grid-template-columns:1fr 1fr;gap:8px">
              <div><label>x</label><input id="esBboxX" type="number" step="0.001" min="0" max="1" /></div>
              <div><label>y</label><input id="esBboxY" type="number" step="0.001" min="0" max="1" /></div>
              <div><label>宽 w</label><input id="esBboxW" type="number" step="0.001" min="0" max="1" /></div>
              <div><label>高 h</label><input id="esBboxH" type="number" step="0.001" min="0" max="1" /></div>
            </div>
            <p class="muted" style="margin-top:10px;font-size:12px">裁剪预览（服务端保存后刷新）：<a id="esBboxCropLink" href="#" target="_blank" rel="noreferrer">—</a></p>
            <div class="row" style="margin-top:12px">
              <button type="button" class="btn" id="esBboxSave">保存本题</button>
            </div>
          </div>
        </div>`,
        false
      );

      mr.querySelector('#esBboxClose')?.addEventListener('click', goBack);
      mr.querySelector('#esBboxBd')?.addEventListener('click', (e) => {
        if (e.target.id === 'esBboxBd') goBack();
      });

      const selQ = mr.querySelector('#esBboxQ');
      const selPage = mr.querySelector('#esBboxPage');
      const img = mr.querySelector('#esBboxImg');
      const overlay = mr.querySelector('#esBboxOverlay');
      const inpX = mr.querySelector('#esBboxX');
      const inpY = mr.querySelector('#esBboxY');
      const inpW = mr.querySelector('#esBboxW');
      const inpH = mr.querySelector('#esBboxH');
      const cropLink = mr.querySelector('#esBboxCropLink');

      function getQ() {
        const id = Number(selQ.value);
        return questions.find((q) => q.id === id);
      }

      function pagePublicUrl(pageNo) {
        const p = pages.find((x) => x.page_no === pageNo);
        return p && p.public_url ? assetURL(p.public_url) : '';
      }

      function syncOverlay() {
        const x = Math.min(1, Math.max(0, parseFloat(inpX.value) || 0));
        const y = Math.min(1, Math.max(0, parseFloat(inpY.value) || 0));
        const w = Math.min(1, Math.max(0, parseFloat(inpW.value) || 0));
        const h = Math.min(1, Math.max(0, parseFloat(inpH.value) || 0));
        overlay.style.left = x * 100 + '%';
        overlay.style.top = y * 100 + '%';
        overlay.style.width = w * 100 + '%';
        overlay.style.height = h * 100 + '%';
      }

      function applyQuestion(q) {
        let pageNo = 1;
        if (q.stem_page_no != null && q.stem_page_no > 0) pageNo = q.stem_page_no;
        else if (q.page_from != null && q.page_from > 0) pageNo = q.page_from;
        selPage.value = String(pageNo);
        const bb = q.stem_bbox_norm && typeof q.stem_bbox_norm === 'object' ? q.stem_bbox_norm : null;
        if (bb && typeof bb.x === 'number') {
          inpX.value = String(bb.x);
          inpY.value = String(bb.y);
          inpW.value = String(bb.w);
          inpH.value = String(bb.h);
        } else {
          inpX.value = '0.05';
          inpY.value = '0.05';
          inpW.value = '0.9';
          inpH.value = '0.25';
        }
        if (q.stem_crop_public_url) {
          cropLink.href = assetURL(q.stem_crop_public_url);
          cropLink.textContent = '打开';
        } else {
          cropLink.removeAttribute('href');
          cropLink.textContent = '（尚无裁剪图）';
        }
        const url = pagePublicUrl(pageNo);
        if (url) img.src = url;
        syncOverlay();
      }

      selQ.addEventListener('change', () => {
        const q = getQ();
        if (q) applyQuestion(q);
      });
      selPage.addEventListener('change', () => {
        const pn = Number(selPage.value) || 1;
        const url = pagePublicUrl(pn);
        if (url) img.src = url;
        syncOverlay();
      });
      ['#esBboxX', '#esBboxY', '#esBboxW', '#esBboxH'].forEach((sel) => {
        mr.querySelector(sel)?.addEventListener('input', syncOverlay);
      });
      img.addEventListener('load', syncOverlay);

      let initial = questions[0];
      if (initialQuestionID) {
        const hit = questions.find((q) => Number(q.id) === Number(initialQuestionID));
        if (hit) {
          initial = hit;
          selQ.value = String(hit.id);
        }
      }
      applyQuestion(initial);

      mr.querySelector('#esBboxSave')?.addEventListener('click', async () => {
        const q = getQ();
        if (!q) return;
        const pageNo = Number(selPage.value) || 1;
        const x = Math.min(1, Math.max(0, parseFloat(inpX.value) || 0));
        const y = Math.min(1, Math.max(0, parseFloat(inpY.value) || 0));
        const w = Math.min(1, Math.max(0, parseFloat(inpW.value) || 0));
        const h = Math.min(1, Math.max(0, parseFloat(inpH.value) || 0));
        if (w <= 0 || h <= 0) {
          alert('宽、高须大于 0');
          return;
        }
        mr.querySelector('#esBboxBd')?.setAttribute('data-busy', '1');
        try {
          const res = await api('/api/v1/admin/exam-source/questions/' + q.id + '/stem-bbox', {
            method: 'PATCH',
            jsonBody: { page_no: pageNo, x, y, w, h },
          });
          q.stem_page_no = pageNo;
          q.stem_bbox_norm = res.bbox_norm || { x, y, w, h };
          if (res.crop_public_url) {
            q.stem_crop_public_url = res.crop_public_url;
            cropLink.href = assetURL(res.crop_public_url);
            cropLink.textContent = '打开（已更新）';
          }
          alert('已保存');
        } catch (e) {
          if (authRedirectHandled(e)) return;
          alert('保存失败: ' + (e.data && e.data.code ? e.data.code : e.message));
        } finally {
          mr.querySelector('#esBboxBd')?.setAttribute('data-busy', '0');
        }
      });
    })();
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
