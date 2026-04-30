// picker.js — Alpine x-data factory for the status-bar module
// picker modal.
//
// The modal is body-mounted in module_picker.templ and starts
// hidden (x-cloak + x-show="open"). Open and close transitions
// run via the standard Alpine event hooks bound in the templ:
//   - status-bar module button dispatches `picker:open`
//   - Escape key dispatches `picker:close` (handled inline)
//
// The module list is preloaded server-side (it's bounded — at most
// ~1k rows in the worst standard-MIB-bundle case), so search is
// pure client-side string matching.
window.picker = function () {
	return {
		open: false,
		query: '',

		matchesRow(el) {
			const q = (this.query || '').toLowerCase();
			if (!q) return true;
			const name = (el.dataset.name || '').toLowerCase();
			return name.includes(q);
		},
	};
};
