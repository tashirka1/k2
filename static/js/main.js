htmx.config.globalViewTransitions = true;

var pendingCharts = {};

function fmtTick(ts, multiDay) {
	var d = new Date(ts);
	var time = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	return multiDay
		? d.toLocaleDateString([], { day: '2-digit', month: '2-digit' }) + ' ' + time
		: time;
}

function initCharts() {
	document.querySelectorAll('canvas[id$=Chart]').forEach(function (canvas) {
		if (pendingCharts[canvas.id]) return;
		pendingCharts[canvas.id] = true;
		var old = Chart.getChart(canvas.id);
		if (old) old.destroy();

		var url = '/metrics/chart/' + canvas.id.replace('Chart', '') + '?period=1h';
		fetch(url).then(function (r) { return r.json(); }).then(function (data) {
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
			delete pendingCharts[canvas.id];
		});
	});
}

document.addEventListener('DOMContentLoaded', initCharts);
document.addEventListener('htmx:afterSettle', initCharts);
