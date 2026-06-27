// a11y.js — shared accessibility helpers, loaded before the other
// islands so they can rely on window.blitterA11y existing.
//
// focusTrap(container, opts) keeps Tab/Shift+Tab cycling within an open
// dialog (WCAG 2.4.3 Focus Order / 2.1.2 No Keyboard Trap) and, on
// teardown, optionally returns focus to the element that opened it. It
// returns a deactivate() function — call it when the dialog closes.

(function () {
	'use strict';

	// Elements that can receive focus. Excludes tabindex="-1" (programmatic
	// focus targets) and disabled controls.
	var FOCUSABLE = [
		'a[href]',
		'button:not([disabled])',
		'input:not([disabled])',
		'select:not([disabled])',
		'textarea:not([disabled])',
		'[tabindex]:not([tabindex="-1"])',
	].join(',');

	function visible(el) {
		// offsetParent is null for display:none; also keep the active
		// element even if a parent has zero box (e.g. mid-transition).
		return el === document.activeElement || el.offsetParent !== null ||
			el.offsetWidth > 0 || el.offsetHeight > 0;
	}

	function focusable(container) {
		return Array.prototype.filter.call(
			container.querySelectorAll(FOCUSABLE), visible);
	}

	function focusTrap(container, opts) {
		opts = opts || {};
		var returnTo = opts.returnFocusTo || null;

		function onKey(e) {
			if (e.key !== 'Tab') return;
			var items = focusable(container);
			if (items.length === 0) { e.preventDefault(); return; }
			var first = items[0];
			var last = items[items.length - 1];
			var active = document.activeElement;
			if (e.shiftKey) {
				if (active === first || !container.contains(active)) {
					e.preventDefault();
					last.focus();
				}
			} else if (active === last || !container.contains(active)) {
				e.preventDefault();
				first.focus();
			}
		}

		container.addEventListener('keydown', onKey);

		return function deactivate() {
			container.removeEventListener('keydown', onKey);
			if (returnTo && typeof returnTo.focus === 'function') {
				try { returnTo.focus(); } catch (_) { /* node removed */ }
			}
		};
	}

	window.blitterA11y = { focusTrap: focusTrap, FOCUSABLE: FOCUSABLE, focusable: focusable };
})();
