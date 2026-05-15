/* Snowy v6 prototype shell behaviour
 * - Theme toggle (light / dark) persisted to localStorage
 * - Mobile preview toggle (clamp app width to 414px)
 * - Marks active nav link based on data-page attribute on <body>
 */

(function () {
  const STORAGE_THEME = 'snowy-v6-theme';
  const STORAGE_PREVIEW = 'snowy-v6-mobile-preview';

  function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    try { localStorage.setItem(STORAGE_THEME, theme); } catch (_) { }
    const btn = document.querySelector('[data-theme-toggle]');
    if (btn) btn.textContent = theme === 'dark' ? '☀︎ 浅色' : '☾ 深色';
  }

  function applyMobilePreview(enabled) {
    const app = document.querySelector('.app');
    if (!app) return;
    app.classList.toggle('app--mobile-preview', enabled);
    try { localStorage.setItem(STORAGE_PREVIEW, enabled ? '1' : '0'); } catch (_) { }
    const btn = document.querySelector('[data-preview-toggle]');
    if (btn) btn.textContent = enabled ? '🖥 桌面预览' : '📱 移动预览';
  }

  function markActiveNav() {
    const page = document.body.dataset.page;
    if (!page) return;
    document.querySelectorAll('.nav__link').forEach((el) => {
      if (el.dataset.page === page) el.classList.add('nav__link--active');
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    let theme = 'light';
    try { theme = localStorage.getItem(STORAGE_THEME) || 'light'; } catch (_) { }
    applyTheme(theme);

    let preview = false;
    try { preview = localStorage.getItem(STORAGE_PREVIEW) === '1'; } catch (_) { }
    applyMobilePreview(preview);

    markActiveNav();

    const themeBtn = document.querySelector('[data-theme-toggle]');
    if (themeBtn) {
      themeBtn.addEventListener('click', function () {
        const cur = document.documentElement.getAttribute('data-theme') || 'light';
        applyTheme(cur === 'dark' ? 'light' : 'dark');
      });
    }

    const previewBtn = document.querySelector('[data-preview-toggle]');
    if (previewBtn) {
      previewBtn.addEventListener('click', function () {
        const app = document.querySelector('.app');
        applyMobilePreview(!app.classList.contains('app--mobile-preview'));
      });
    }

    // accordion fallback for browsers that ignore <details> auto-toggle
    document.querySelectorAll('details.accordion__item').forEach((el) => {
      el.addEventListener('toggle', () => { /* css handles ::after */ });
    });
  });
})();
