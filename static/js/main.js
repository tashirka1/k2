htmx.config.globalViewTransitions = false;

var pendingCharts = {};

function fmtTick(ts, multiDay) {
	var d = new Date(ts);
	var time = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	return multiDay
		? d.toLocaleDateString([], { day: '2-digit', month: '2-digit' }) + ' ' + time
		: time;
}

function getPeriod() {
	return new URLSearchParams(location.search).get('period') || '1h';
}

function initCharts() {
	document.querySelectorAll('canvas[id$=Chart]').forEach(function (canvas) {
		if (pendingCharts[canvas.id] && pendingCharts[canvas.id].el === canvas) return;
		pendingCharts[canvas.id] = { el: canvas, token: Date.now() };

		var old = Chart.getChart(canvas.id);
		if (old) old.destroy();

		var url = canvas.dataset.chart
			? canvas.dataset.chart + '?' + new URLSearchParams({ period: getPeriod() })
			: '/metrics/chart/' + canvas.id.replace('Chart', '') + '?period=' + getPeriod();
		fetch(url).then(function (r) { return r.json(); }).then(function (data) {
			if (pendingCharts[canvas.id].el !== canvas) return;
			var labels = data.labels || [];
			var multiDay = labels.length > 1 &&
				new Date(labels[0]).toDateString() !== new Date(labels[labels.length - 1]).toDateString();
			new Chart(canvas, {
				type: 'line',
				data: {
					labels: labels,
					datasets: (data.series || []).map(function (s) { return { label: s.label, data: s.data || [] }; })
				},
				options: {
					plugins: {
						tooltip: { callbacks: { title: function (items) { return items.length && new Date(items[0].label).toLocaleString() || ''; } } }
					},
					scales: {
						x: {
							ticks: {
								maxRotation: 0, autoSkip: true, maxTicksLimit: 10,
								callback: function (v) { return fmtTick(this.getLabelForValue(v), multiDay); }
							}
						}
					}
				}
			});
			if (pendingCharts[canvas.id].el === canvas) delete pendingCharts[canvas.id];
		}).catch(function () {
			if (pendingCharts[canvas.id] && pendingCharts[canvas.id].el === canvas) delete pendingCharts[canvas.id];
		});
	});
}

function bindPeriod() {
	var sel = document.getElementById('period');
	if (!sel || sel.dataset.bound) return;
	sel.dataset.bound = '1';
	sel.value = getPeriod();
	sel.addEventListener('change', function () {
		var params = new URLSearchParams(location.search);
		params.set('period', this.value);
		history.pushState({}, '', location.pathname + '?' + params.toString());
		pendingCharts = {};
		initCharts();
	});
}

var refreshTimer = null;
var refreshKey = 'k2_refresh';

function refreshMs(value) {
	if (value === 'off') return 0;
	var m = value.match(/^(\d+)([sm])$/);
	if (!m) return 0;
	var n = parseInt(m[1], 10) * (m[2] === 'm' ? 60000 : 1000);
	return n;
}

function fireUpdate() {
	document.querySelectorAll('[hx-trigger="update-data"]').forEach(function (el) {
		el.dispatchEvent(new Event('update-data'));
	});
	initCharts();
}

function applyRefresh() {
	clearInterval(refreshTimer);
	refreshTimer = null;
	var ms = refreshMs(document.getElementById('refresh').value);
	if (ms > 0) refreshTimer = setInterval(fireUpdate, ms);
}

function bindRefresh() {
	var sel = document.getElementById('refresh');
	if (!sel || sel.dataset.bound) return;
	sel.dataset.bound = '1';
	sel.value = localStorage.getItem(refreshKey) || 'off';
	sel.addEventListener('change', function () {
		localStorage.setItem(refreshKey, this.value);
		applyRefresh();
	});
	applyRefresh();
}

document.addEventListener('DOMContentLoaded', initCharts);
document.addEventListener('htmx:afterSettle', initCharts);
document.addEventListener('DOMContentLoaded', bindPeriod);
document.addEventListener('htmx:afterSettle', bindPeriod);
document.addEventListener('DOMContentLoaded', bindRefresh);
