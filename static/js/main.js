htmx.config.globalViewTransitions = true;

var pendingCharts = {};

function initCharts() {
	document.querySelectorAll('canvas[id$=Chart]').forEach(function (canvas) {
		if (pendingCharts[canvas.id]) return;
		pendingCharts[canvas.id] = true;

		var existing = Chart.getChart(canvas.id);
		if (existing) existing.destroy();

		var type = canvas.id.replace('Chart', '');
		var endpoint = '/metrics/chart/' + type + '?period=1h';

		fetch(endpoint)
			.then(function (r) { return r.json(); })
			.then(function (data) {
				var datasets = (data.series || []).map(function (s) {
					return { label: s.label, data: s.data || [] };
				});
				new Chart(canvas, {
					type: 'line',
					data: {
						labels: data.labels,
						datasets: datasets
					}
				});
				delete pendingCharts[canvas.id];
			});
	});
}

document.addEventListener('DOMContentLoaded', initCharts);
document.addEventListener('htmx:afterSettle', initCharts);
