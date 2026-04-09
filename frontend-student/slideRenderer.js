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
      const inputs =      const inputs = [];
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
